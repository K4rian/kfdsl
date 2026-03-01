package launcher

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/K4rian/kfdsl/cmd"
	"github.com/K4rian/kfdsl/internal/log"
	"github.com/K4rian/kfdsl/internal/services/kfserver"
	"github.com/K4rian/kfdsl/internal/settings"
)

type Launcher struct {
	settings *settings.Settings
}

func New() *Launcher {
	return &Launcher{
		settings: &settings.Settings{},
	}
}

func (l *Launcher) Run() error {
	// Build the root command and execute it
	rootCmd := cmd.BuildRootCommand(l.settings)
	if err := rootCmd.Execute(); err != nil {
		return err
	}

	// Init the logger
	if err := log.Init(
		l.settings.LogLevel.Value(),
		l.settings.LogFile.Value(),
		l.settings.LogFileFormat.Value(),
		l.settings.LogMaxSize.Value(),
		l.settings.LogMaxBackups.Value(),
		l.settings.LogMaxAge.Value(),
		l.settings.LogToFile.Value(),
	); err != nil {
		return fmt.Errorf("failed to init the logger: %v", err)
	}
	log.Logger.Debug("Log system initialized",
		"function", "Run")

	// Create a cancel context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	// Print all settings
	l.settings.Print()

	// Start SteamCMD, if enabled
	if !l.settings.NoSteam.Value() {
		start := time.Now()
		if err := l.startSteamCMD(ctx); err != nil {
			// log.Logger.Error("SteamCMD raised an error", "error", err)
			return fmt.Errorf("steamcmd raises an error: %w", err)
		}
		log.Logger.Debug("SteamCMD process completed",
			"function", "Run", "elapsedTime", time.Since(start))
	} else {
		log.Logger.Debug("SteamCMD is disabled and won't be started",
			"function", "Run")
	}

	var server *kfserver.KFServer
	var startTime time.Time

	defer func() {
		log.Logger.Info("Shutting down the KF Dedicated Server...")

		if server != nil && server.IsRunning() {
			log.Logger.Debug("Waiting for the KF Dedicated Server to stop...",
				"function", "Run")
			if err := server.Wait(); err != nil {
				log.Logger.Error("KF Dedicated Server raised an error during shutdown", "error", err)
			}
		}
		log.Logger.Info("KF Dedicated Server has been stopped.")
		log.Logger.Debug("KF Dedicated Server process completed",
			"function", "Run", "elapsedTime", time.Since(startTime))
	}()

	// Start the Killing Floor Dedicated Server
	startTime = time.Now()
	server, err := l.startGameServer(ctx)
	if err != nil {
		log.Logger.Error("KF Dedicated Server raised an error", "error", err)
	}

	<-signalChan
	signal.Stop(signalChan)
	cancel()

	log.Logger.Debug("Program finished, exiting now...",
		"function", "Run")

	return nil
}
