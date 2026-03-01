package launcher

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/viper"

	"github.com/K4rian/kfdsl/internal/config/secrets"
	"github.com/K4rian/kfdsl/internal/log"
	"github.com/K4rian/kfdsl/internal/services/steamcmd"
)

func (l *Launcher) startSteamCMD(ctx context.Context) error {
	rootDir := l.settings.SteamCMDRoot.Value()
	steamCMD := steamcmd.New(ctx, rootDir)

	log.Logger.Debug("Initializing SteamCMD",
		"function", "startSteamCMD", "rootDir", rootDir)

	if !steamCMD.IsInstalled() {
		return fmt.Errorf("SteamCMD not found in %s. Please install it manually", steamCMD.Options().RootDirectory)
	}

	// Read Steam Account Credentials
	if err := l.readSteamCredentials(); err != nil {
		return fmt.Errorf("failed to read Steam credentials: %w", err)
	}

	// Generate the Steam install script
	installScript := filepath.Join(rootDir, "kfds_install_script.txt")
	serverInstallDir := l.settings.ServerInstallDir.Value()

	log.Logger.Info("Writing the KF Dedicated Server install script...", "scriptPath", installScript)
	if err := steamCMD.WriteScript(
		installScript,
		l.settings.SteamLogin,
		l.settings.SteamPassword,
		serverInstallDir,
		KF_APPID,
		!l.settings.NoValidate.Value(),
	); err != nil {
		return err
	}
	log.Logger.Info("Install script was successfully written", "scriptPath", installScript)

	log.Logger.Info("Starting SteamCMD...", "rootDir", steamCMD.Options().RootDirectory, "appInstallDir", serverInstallDir)
	if err := steamCMD.RunScript(installScript); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}

	// Purge the install script file once SteamCMD process exits
	defer func() {
		if err := os.Remove(installScript); err != nil && !os.IsNotExist(err) {
			log.Logger.Warn("Could not delete install script file", "scriptPath", installScript, "error", err)
		} else {
			log.Logger.Info("Install script was successfully deleted", "scriptPath", installScript)
		}
	}()

	// Block until SteamCMD finishes
	log.Logger.Debug("Wait till SteamCMD finishes",
		"function", "startSteamCMD", "rootDir", rootDir)
	start := time.Now()
	if err := steamCMD.Wait(); err != nil {
		return err
	}
	log.Logger.Debug("SteamCMD process completed",
		"function", "startSteamCMD", "rootDir", rootDir, "elapsedTime", time.Since(start))
	return nil
}

func (l *Launcher) readSteamCredentials() error {
	var fromEnv bool

	defer func() {
		if fromEnv {
			_ = os.Unsetenv("STEAMACC_USERNAME")
			_ = os.Unsetenv("STEAMACC_PASSWORD")
		}
	}()

	log.Logger.Debug("Starting Steam credential retrieval",
		"function", "readSteamCredentials")

	// Try reading from Docker Secrets
	log.Logger.Debug("Attempting to read from Docker Secrets",
		"function", "readSteamCredentials")
	steamUsername, errUser := secrets.Read("steamacc_username")
	steamPassword, errPass := secrets.Read("steamacc_password")

	// Fallback to environment variables if secrets are missing
	if errUser != nil {
		log.Logger.Debug("Secret not found, falling back to environment variable",
			"function", "readSteamCredentials", "secret", "steamacc_username", "error", errUser)
		steamUsername = viper.GetString("STEAMACC_USERNAME")
		fromEnv = true
	}
	if errPass != nil {
		log.Logger.Debug("Secret not found, falling back to environment variable",
			"function", "readSteamCredentials", "secret", "steamacc_password", "error", errPass)
		steamPassword = viper.GetString("STEAMACC_PASSWORD")
		fromEnv = true
	}

	// Ensure both credentials are present
	if steamUsername == "" || steamPassword == "" {
		log.Logger.Debug("Missing Steam credentials, aborting",
			"function", "readSteamCredentials", "steamUsernameEmpty", steamUsername == "", "steamPasswordEmpty", steamPassword == "")
		return fmt.Errorf("incomplete credentials: Steam username and password are required")
	}

	// Update the settings
	log.Logger.Debug("Successfully retrieved credentials, updating settings",
		"function", "readSteamCredentials")

	l.settings.SteamLogin = steamUsername
	l.settings.SteamPassword = steamPassword
	return nil
}
