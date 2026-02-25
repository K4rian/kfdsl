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

func Run() {
	// Build the root command and execute it
	rootCmd := cmd.BuildRootCommand()
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}

	// Get the settings
	sett := settings.Get()

	// Init the logger
	if err := log.Init(
		sett.LogLevel.Value(),
		sett.LogFile.Value(),
		sett.LogFileFormat.Value(),
		sett.LogMaxSize.Value(),
		sett.LogMaxBackups.Value(),
		sett.LogMaxAge.Value(),
		sett.LogToFile.Value(),
	); err != nil {
		fmt.Printf("ERROR: Failed to init the logger: %v", err)
		os.Exit(1)
	}
	log.Logger.Debug("Log system initialized",
		"function", "main")

	// Create a cancel context
	ctx, cancel := context.WithCancel(context.Background())
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	// Print all settings
	sett.Print()

	// Start SteamCMD, if enabled
	if !sett.NoSteam.Value() {
		start := time.Now()
		if err := startSteamCMD(sett, ctx); err != nil {
			log.Logger.Error("SteamCMD raised an error", "error", err)
			os.Exit(1)
		}
		log.Logger.Debug("SteamCMD process completed",
			"function", "main", "elapsedTime", time.Since(start))
	} else {
		log.Logger.Debug("SteamCMD is disabled and won't be started",
			"function", "main")
	}

	var server *kfserver.KFServer
	var startTime time.Time

	defer func() {
		log.Logger.Info("Shutting down the KF Dedicated Server...")
		cancel()

		if server != nil && server.IsRunning() {
			log.Logger.Debug("Waiting for the KF Dedicated Server to stop...",
				"function", "main")
			if err := server.Wait(); err != nil {
				log.Logger.Error("KF Dedicated Server raised an error during shutdown", "error", err)
			}
		}
		log.Logger.Info("KF Dedicated Server has been stopped.")
		log.Logger.Debug("KF Dedicated Server process completed",
			"function", "main", "elapsedTime", time.Since(startTime))
	}()

	// Start the Killing Floor Dedicated Server
	startTime = time.Now()
	server, err := startGameServer(sett, ctx)
	if err != nil {
		log.Logger.Error("KF Dedicated Server raised an error", "error", err)
	}

	<-signalChan
	signal.Stop(signalChan)
	log.Logger.Debug("Program finished, exiting now...",
		"function", "main")
}
