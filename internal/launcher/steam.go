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
	"github.com/K4rian/kfdsl/internal/settings"
)

const (
	KF_APPID = 215360
)

func startSteamCMD(sett *settings.KFDSLSettings, ctx context.Context) error {
	rootDir := viper.GetString("steamcmd-root")
	steamCMD := steamcmd.NewSteamCMD(rootDir, ctx)

	log.Logger.Debug("Initializing SteamCMD",
		"function", "startSteamCMD", "rootDir", rootDir)

	if !steamCMD.IsAvailable() {
		return fmt.Errorf("SteamCMD not found in %s. Please install it manually", steamCMD.RootDirectory())
	}

	// Read Steam Account Credentials
	if err := readSteamCredentials(sett); err != nil {
		return fmt.Errorf("failed to read Steam credentials: %w", err)
	}

	// Generate the Steam install script
	installScript := filepath.Join(rootDir, "kfds_install_script.txt")
	serverInstallDir := viper.GetString("steamcmd-appinstalldir")

	log.Logger.Info("Writing the KF Dedicated Server install script...", "scriptPath", installScript)
	if err := steamCMD.WriteScript(
		installScript,
		sett.SteamLogin,
		sett.SteamPassword,
		serverInstallDir,
		KF_APPID,
		!sett.NoValidate.Value(),
	); err != nil {
		return err
	}
	log.Logger.Info("Install script was successfully written", "scriptPath", installScript)

	log.Logger.Info("Starting SteamCMD...", "rootDir", steamCMD.RootDirectory(), "appInstallDir", serverInstallDir)
	if err := steamCMD.RunScript(installScript); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}

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

func readSteamCredentials(sett *settings.KFDSLSettings) error {
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
	sett.SteamLogin = steamUsername
	sett.SteamPassword = steamPassword
	return nil
}
