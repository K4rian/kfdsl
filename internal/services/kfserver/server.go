package kfserver

import (
	"context"
	"fmt"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/K4rian/kfdsl/internal/services/base"
	"github.com/K4rian/kfdsl/internal/utils"
)

const (
	heartbeatInterval = 6 * time.Second
	heartbeatTimeout  = 30 * heartbeatInterval
	heartbeatSignal   = "Sending updated server details"
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

type KFServer struct {
	*base.BaseService
	configFileName string
	startupMap     string
	gameMode       string
	unsecure       bool
	maxPlayers     int
	mutators       string
	extraArgs      []string
	executable     string
	ready          bool
	stateMu        sync.RWMutex
	// Heartbeat watchdog
	lastHeartbeat  time.Time
	watchdogStop   chan struct{}
	watchdogMu     sync.Mutex
	currentPlayers int
}

func NewKFServer(
	rootDir string,
	configFileName string,
	startupMap string,
	gameMode string,
	unsecure bool,
	maxPlayers int,
	mutators string,
	extraArgs []string,
	ctx context.Context,
) *KFServer {
	kfs := &KFServer{
		BaseService:    base.NewBaseService("KFServer", rootDir, ctx),
		configFileName: configFileName,
		startupMap:     startupMap,
		gameMode:       gameMode,
		unsecure:       unsecure,
		maxPlayers:     maxPlayers,
		mutators:       mutators,
		extraArgs:      extraArgs,
		executable:     path.Join(rootDir, "System", "ucc-bin"),
	}
	kfs.AddLogHandler(kfs.handleCrash)
	kfs.AddLogHandler(kfs.handleHeartbeat)

	kfs.SetPreRestartHook(kfs.stopWatchdog)
	kfs.SetPostRestartHook(kfs.startWatchdog)
	return kfs
}

func (s *KFServer) Start(autoRestart bool) error {
	args := s.buildCommandLine()
	err := s.BaseService.Start(args, autoRestart)
	if err != nil {
		return err
	}
	s.startWatchdog()
	return nil
}

func (s *KFServer) Stop() error {
	s.stopWatchdog()
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
	argsBuilder.WriteString(s.startupMap)
	argsBuilder.WriteString(".rom?game=")
	argsBuilder.WriteString(s.gameMode)
	argsBuilder.WriteString("?VACSecured=")
	argsBuilder.WriteString(fmt.Sprintf("%t", !s.unsecure))
	argsBuilder.WriteString("?MaxPlayers=")
	argsBuilder.WriteString(fmt.Sprintf("%d", s.maxPlayers))

	// Append Mutator(s) if provided
	if s.mutators != "" {
		argsBuilder.WriteString("?Mutator=")
		argsBuilder.WriteString(s.mutators)
	}

	// Specify the configuration file to use
	iniFile := "ini=" + s.configFileName

	// Final command
	args := []string{
		s.executable,
		"server",
		argsBuilder.String(),
		iniFile,
		"-nohomedir",
	}

	// Append extra arguments if provided
	if len(s.extraArgs) > 0 {
		args = append(args, s.extraArgs...)
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

// handleHeartbeat parses the periodic STEAMAUTH line to update runtime state.
// Expected format:
//
//	STEAMAUTH : Sending updated server details - <SERVER_NAME> <CURRENT> | <MAX>
//
// Example:
//
//	STEAMAUTH : Sending updated server details - My Server 2 | 6
func (s *KFServer) handleHeartbeat(line string) bool {
	if !strings.Contains(line, heartbeatSignal) {
		return false
	}

	// Split on " | " to isolate the left side, which contains the current player count
	// as its last token. The right side (max players) is discarded
	parts := strings.SplitN(line, " | ", 2)
	if len(parts) != 2 {
		// Heartbeat detected but unparseable
		// still mark the server as available and reset the watchdog timer
		s.updateHeartbeat(0, false)
		return false
	}

	current, err := parseCurrentPlayers(parts[0])
	if err != nil {
		s.Logger().Warn("Failed to parse heartbeat player count", "error", err)
		s.updateHeartbeat(0, false)
		return false
	}

	s.updateHeartbeat(current, true)
	return false
}

// updateHeartbeat applies a parsed heartbeat to the runtime state under a
// single write lock
func (s *KFServer) updateHeartbeat(currentP int, parsed bool) {
	s.stateMu.Lock()
	wasAvailable := s.ready
	s.lastHeartbeat = time.Now()
	s.ready = true
	if parsed {
		s.currentPlayers = currentP
	}
	s.stateMu.Unlock()

	s.Logger().Debug("Heartbeat received", "players", currentP)

	if !wasAvailable {
		s.Logger().Info("Server is ready")
	}
}

// startWatchdog launches a goroutine that restarts the server if heartbeats stop.
func (s *KFServer) startWatchdog() {
	s.watchdogMu.Lock()
	defer s.watchdogMu.Unlock()

	// Ensure any previous channel is cleaned up before creating a new one
	if s.watchdogStop != nil {
		return
	}
	s.watchdogStop = make(chan struct{})

	go func() {
		ticker := time.NewTicker(heartbeatInterval)
		defer ticker.Stop()

		for {
			select {
			case <-s.watchdogStop:
				return
			case <-ticker.C:
				s.stateMu.RLock()
				last := s.lastHeartbeat
				avail := s.ready
				s.stateMu.RUnlock()

				// Only act once the server is ready
				// Don't trigger on a (potentially long) silent boot phase
				if avail && time.Since(last) > heartbeatTimeout {
					s.Logger().Warn("Heartbeat timeout. Server may be hung, triggering restart")
					s.setReady(false)
					// Restart will call related hooks so the watchdog
					// is safely cycled around the restart
					go s.Restart()
				}
			}
		}
	}()
}

func (s *KFServer) stopWatchdog() {
	s.watchdogMu.Lock()
	defer s.watchdogMu.Unlock()

	if s.watchdogStop != nil {
		close(s.watchdogStop)
		s.watchdogStop = nil
	}
}

func (s *KFServer) setReady(v bool) {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	s.ready = v
}

func parseCurrentPlayers(s string) (int, error) {
	tokens := strings.Fields(s)
	if len(tokens) == 0 {
		return 0, fmt.Errorf("unexpected heartbeat format: %q", s)
	}

	cur, err := strconv.Atoi(tokens[len(tokens)-1])
	if err != nil {
		return 0, fmt.Errorf("current players: %w", err)
	}
	return cur, nil
}
