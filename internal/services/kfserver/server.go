package kfserver

import (
	"context"
	"fmt"
	"path"
	"strings"
	"sync"

	"github.com/K4rian/kfdsl/internal/services/base"
	"github.com/K4rian/kfdsl/internal/utils"
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
	return kfs
}

func (s *KFServer) Start(autoRestart bool) error {
	args := s.buildCommandLine()
	err := s.BaseService.Start(args, autoRestart)
	if err != nil {
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

func (s *KFServer) setReady(v bool) {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	s.ready = v
}
