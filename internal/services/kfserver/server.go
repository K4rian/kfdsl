package kfserver

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/K4rian/kfdsl/internal/services/base"
	"github.com/K4rian/kfdsl/internal/settings"
	"github.com/K4rian/kfdsl/internal/utils"
)

type KFServer struct {
	*base.BaseService
	settings   *settings.Settings
	executable string
	ready      bool
	stateMu    sync.RWMutex
}

const (
	relExecutablePath = "System/ucc-bin"
)

// UE2 patterns that indicate a fatal crash
var crashPatterns = []string{
	"Critical:",
	"appError called",
	"Fatal error",
	"Assertion failed",
	"Exiting due to error",
	"Signal: SIGSEGV",
}

func New(ctx context.Context, sett *settings.Settings) *KFServer {
	rootDir := sett.ServerInstallDir.Value()
	executable := filepath.Join(rootDir, relExecutablePath)
	workingDir := filepath.Dir(executable)

	kfs := &KFServer{
		BaseService: base.NewBaseService("KFServer", ctx, base.ServiceOptions{
			RootDirectory:    rootDir,
			WorkingDirectory: workingDir,
			AutoRestart:      sett.AutoRestart.Value(),
			MaxRestarts:      sett.MaxRestarts.Value(),
			RestartDelay:     sett.RestartDelay.Value(),
			ShutdownTimeout:  sett.ShutdownTimeout.Value(),
			KillTimeout:      sett.KillTimeout.Value(),
		}),
		settings:   sett,
		executable: executable,
	}
	kfs.AddLogHandler(kfs.handleCrash)
	return kfs
}

func (s *KFServer) Start() error {
	if err := s.BaseService.Start(s.buildCommandLine()); err != nil {
		return err
	}
	return nil
}

func (s *KFServer) Stop() error {
	s.setReady(false)
	return s.BaseService.Stop()
}

// IsInstalled returns true when the server executable is present on disk.
func (s *KFServer) IsInstalled() bool {
	return utils.FileExists(s.executable)
}

// IsReady returns true when the server is live and accepting connections.
func (s *KFServer) IsReady() bool {
	s.stateMu.RLock()
	defer s.stateMu.RUnlock()
	return s.ready
}

func (s *KFServer) buildCommandLine() []string {
	var argsBuilder strings.Builder

	// Base command
	argsBuilder.WriteString(s.settings.StartupMap.Value())
	argsBuilder.WriteString(".rom?game=")
	argsBuilder.WriteString(s.settings.GameMode.Value())
	argsBuilder.WriteString("?VACSecured=")
	argsBuilder.WriteString(fmt.Sprintf("%t", !s.settings.Unsecure.Value()))
	argsBuilder.WriteString("?MaxPlayers=")
	argsBuilder.WriteString(fmt.Sprintf("%d", s.settings.MaxPlayers.Value()))

	// Append Mutator(s) if provided
	mutators := strings.TrimSpace(s.settings.Mutators.Value())
	if s.settings.EnableMutLoader.Value() {
		mutators = "MutLoader.MutLoader"
	}
	if mutators != "" {
		argsBuilder.WriteString("?Mutator=")
		argsBuilder.WriteString(mutators)
	}

	// Base arguments
	args := []string{
		s.executable,
		"server",
		argsBuilder.String(),
		"ini=" + s.settings.ConfigFile.Value(),
		"-nohomedir",
	}

	// Extra arguments
	extraArgs := s.settings.ExtraArgs
	if len(extraArgs) > 0 {
		args = append(args, extraArgs...)
	}
	return args
}

func (s *KFServer) handleCrash(line string) bool {
	for _, pattern := range crashPatterns {
		if strings.Contains(line, pattern) {
			s.Logger().Error("Crash detected", "pattern", pattern, "line", line)
			s.setReady(false)
			return true
		}
	}
	return false
}

func (s *KFServer) setReady(v bool) {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	s.ready = v
}
