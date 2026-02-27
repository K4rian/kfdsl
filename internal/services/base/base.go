package base

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/K4rian/dslogger"
	"github.com/creack/pty"

	"github.com/K4rian/kfdsl/internal/log"
)

type BaseService struct {
	name        string
	rootDir     string
	autoRestart bool
	args        []string
	cmd         *exec.Cmd
	ctx         context.Context
	cancel      context.CancelFunc
	mu          sync.Mutex
	ptmx        *os.File
	logger      *dslogger.Logger
	done        chan struct{}
	stopping    bool
	execErr     error
	startOnce   sync.Once
}

func NewBaseService(name string, rootDir string, ctx context.Context) *BaseService {
	bs := &BaseService{
		name:    name,
		rootDir: rootDir,
		logger:  log.Logger.WithService(name),
	}
	bs.ctx, bs.cancel = context.WithCancel(ctx)
	return bs
}

func (bs *BaseService) Name() string {
	return bs.name
}

func (bs *BaseService) RootDirectory() string {
	return bs.rootDir
}

// Start initiates the service's process and manages the start/stop lifecycle.
func (bs *BaseService) Start(args []string, autoRestart bool) error {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	if bs.IsRunning() {
		return fmt.Errorf("already running")
	}

	// Store the auto-restart value
	bs.autoRestart = autoRestart

	// Store the new arguments
	bs.args = args

	// Reset the stopping variable
	bs.stopping = false

	// Reset the execution error variable
	bs.execErr = nil

	// Set up the command
	execDir := filepath.Dir(args[0])
	if execDir == "." {
		execDir = ""
	}
	cmd := exec.CommandContext(bs.ctx, args[0], args[1:]...)
	cmd.Dir = execDir
	bs.cmd = cmd

	// Start the process with a pseudo-terminal
	ptmx, err := pty.Start(bs.cmd)
	if err != nil {
		return fmt.Errorf("failed to start pty: %v", err)
	}
	bs.ptmx = ptmx

	// Create a done channel to signal when the process is finished
	bs.done = make(chan struct{})

	// Goroutine to monitor the cancellation context
	bs.startOnce.Do(func() {
		go bs.monitorCancellation()
	})

	// Goroutine to handle the process auto-restart
	if bs.autoRestart {
		go bs.monitorAutoRestart()
	}

	// Goroutine for real-time log capture and wait for process exit
	go func() {
		defer func() {
			bs.mu.Lock()
			if bs.ptmx != nil {
				bs.ptmx.Close()
				bs.ptmx = nil
			}
			bs.cmd = nil
			bs.mu.Unlock()
			close(bs.done)
		}()

		scanner := bufio.NewScanner(ptmx)
		for scanner.Scan() {
			bs.logger.Info(scanner.Text())
		}

		if err := scanner.Err(); err != nil && !errors.Is(err, syscall.EIO) {
			bs.mu.Lock()
			bs.execErr = fmt.Errorf("error reading from pty: %w", err)
			bs.mu.Unlock()
		}

		// Wait for the process to exit
		if err := cmd.Wait(); err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 0 {
				bs.logger.Debug("Process exited normally")
			} else {
				bs.mu.Lock()
				bs.execErr = fmt.Errorf("process exited with error: %v", err)
				bs.mu.Unlock()
			}
		}
	}()
	return nil
}

// Stop attempts to stop the service's process gracefully, then forcefully if necessary.
func (bs *BaseService) Stop() error {
	bs.mu.Lock()

	// The process is already stopped or never started
	if bs.cmd == nil {
		bs.mu.Unlock()
		return nil
	}

	// Prevent auto-restart
	bs.stopping = true

	// Capture what we need before releasing the lock
	ptmx := bs.ptmx
	done := bs.done

	bs.mu.Unlock()

	// Send CTRL+C to gracefully terminate the process
	if ptmx != nil {
		bs.logger.Info("Attempting to send SIGINT...")
		if _, err := ptmx.Write([]byte{3}); err != nil {
			bs.logger.Error("Failed to send SIGINT to pty", "error", err)
		}
	}

	// Wait for a graceful exit
	select {
	case <-done:
		bs.logger.Info("Process exited gracefully")
		return nil
	case <-time.After(2 * time.Second):
	}

	// Process is still alive, escalate to SIGTERM
	bs.mu.Lock()
	cmd := bs.cmd
	bs.mu.Unlock()

	if cmd != nil && cmd.Process != nil {
		bs.logger.Warn("Process did not exit after SIGINT, attempting SIGTERM...")
		if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
			bs.logger.Error("Failed to send SIGTERM, attempting SIGKILL...", "error", err)
			if err := cmd.Process.Kill(); err != nil {
				return fmt.Errorf("failed to force kill process: %v", err)
			}
			bs.logger.Info("Process forcefully killed")
		}
	}

	// Wait for the process to finish after SIGTERM/SIGKILL
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		return fmt.Errorf("process did not exit after SIGTERM/SIGKILL")
	}
	return nil
}

// Wait waits for the process to exit and returns any execution error.
func (bs *BaseService) Wait() error {
	<-bs.done
	return bs.execErr
}

// IsRunning returns whether the service is currently running.
func (bs *BaseService) IsRunning() bool {
	return bs.cmd != nil && bs.cmd.Process != nil && bs.cmd.ProcessState == nil
}

// IsAvailable returns whether the service is available.
func (bs *BaseService) IsAvailable() bool {
	return false
}

// monitorAutoRestart keeps an eye on the process and restarts it.
func (bs *BaseService) monitorAutoRestart() {
	for {
		<-bs.done

		bs.mu.Lock()
		stopping := bs.stopping
		autoRestart := bs.autoRestart
		execErr := bs.execErr
		args := bs.args
		bs.mu.Unlock()

		if stopping || !autoRestart || execErr != nil || bs.ctx.Err() != nil {
			return
		}

		bs.logger.Info(fmt.Sprintf("%s stopped, restarting...", bs.name))
		time.Sleep(2 * time.Second)

		if err := bs.Start(args, autoRestart); err != nil {
			bs.logger.Error("Failed to restart service", "error", err)
			return
		}
	}
}

// monitorCancellation listens for OS interrupt signals and gracefully shuts down the process.
func (bs *BaseService) monitorCancellation() {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	defer signal.Stop(signalChan)

	select {
	case <-signalChan:
		bs.logger.Info("Interrupt received, shutting down...")
		bs.cancel()
		bs.Stop()
	case <-bs.ctx.Done():
		// Context was cancelled externally, nothing to do
	}
}
