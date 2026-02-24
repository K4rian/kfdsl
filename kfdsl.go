package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"

	"github.com/K4rian/kfdsl/internal/config"
	"github.com/K4rian/kfdsl/internal/config/secrets"
	"github.com/K4rian/kfdsl/internal/log"
	"github.com/K4rian/kfdsl/internal/mods"
	"github.com/K4rian/kfdsl/internal/services/kfserver"
	"github.com/K4rian/kfdsl/internal/services/steamcmd"
	"github.com/K4rian/kfdsl/internal/settings"
	"github.com/K4rian/kfdsl/internal/utils"
)

type configUpdater[T any] struct {
	name string       // Name
	gv   func() T     // Getter
	sv   func(T) bool // Setter
	nv   any          // Value to set
}

func newConfigUpdater[T any](name string, gv func() T, sv func(T) bool, nv any) configUpdater[T] {
	return configUpdater[T]{
		name: name,
		gv:   gv,
		sv:   sv,
		nv:   nv,
	}
}

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

func startGameServer(sett *settings.KFDSLSettings, ctx context.Context) (*kfserver.KFServer, error) {
	mutators := sett.Mutators.Value()
	if sett.EnableMutLoader.Value() {
		mutators = "MutLoader.MutLoader"
	}

	rootDir := viper.GetString("steamcmd-appinstalldir")
	configFileName := sett.ConfigFile.Value()
	startupMap := sett.StartupMap.Value()
	gameMode := sett.GameMode.Value()
	unsecure := sett.Unsecure.Value()
	maxPlayers := sett.MaxPlayers.Value()
	extraArgs := viper.GetStringSlice("KF_EXTRAARGS")

	gameServer := kfserver.NewKFServer(
		rootDir,
		configFileName,
		startupMap,
		gameMode,
		unsecure,
		maxPlayers,
		mutators,
		extraArgs,
		ctx,
	)

	log.Logger.Debug("Initializing KF Dedicated Server",
		"function", "startGameServer", "rootDir", rootDir, "startupMap", startupMap,
		"gameMode", gameMode, "unsecure", unsecure, "maxPlayers", maxPlayers,
		"mutators", mutators, "extraArgs", extraArgs,
	)

	if !gameServer.IsAvailable() {
		return nil, fmt.Errorf("unable to locate the KF Dedicated Server files in '%s', please install using SteamCMD", gameServer.RootDirectory())
	}

	log.Logger.Info("Updating the KF Dedicated Server configuration file...", "file", configFileName)
	if err := updateConfigFile(sett); err != nil {
		return nil, fmt.Errorf("failed to update the KF Dedicated Server configuration file %s: %w", configFileName, err)
	}
	log.Logger.Info("Server configuration file successfully updated", "file", configFileName)

	if err := installMods(sett); err != nil {
		log.Logger.Error("Failed to install mods", "file", sett.ModsFile.Value(), "error", err)
		return nil, fmt.Errorf("failed to install mods: %w", err)
	}

	if sett.EnableKFPatcher.Value() {
		kfpConfigFilePath := filepath.Join(rootDir, "System", "KFPatcherSettings.ini")
		log.Logger.Info("Updating the KFPatcher configuration file...", "file", kfpConfigFilePath)
		if err := updateKFPatcherConfigFile(sett); err != nil {
			return nil, fmt.Errorf("failed to update the KFPatcher configuration file %s: %w", kfpConfigFilePath, err)
		}
		log.Logger.Info("KFPatcher configuration file successfully updated", "file", kfpConfigFilePath)
	}

	log.Logger.Info("Verifying KF Dedicated Server Steam libraries for updates...")
	updatedLibs, err := updateGameServerSteamLibs()
	if err == nil {
		if len(updatedLibs) > 0 {
			for _, lib := range updatedLibs {
				log.Logger.Info("Steam library successfully updated", "library", lib)
			}
		} else {
			log.Logger.Info("All server Steam libraries are up-to-date")
		}
	} else {
		log.Logger.Error("Unable to update the KF Dedicated Server Steam libraries", "error", err)
	}

	log.Logger.Info("Starting the KF Dedicated Server...", "rootDir", gameServer.RootDirectory(), "autoRestart", sett.AutoRestart.Value())
	if err := gameServer.Start(sett.AutoRestart.Value()); err != nil {
		return nil, fmt.Errorf("failed to start the KF Dedicated Server: %w", err)
	}
	return gameServer, nil
}

func extractDefaultConfigFile(filename string, filePath string) error {
	defaultIniFilePath := filepath.Join("assets/configs", filename)

	log.Logger.Debug("Extracting default configuration file",
		"function", "extractDefaultConfigFile", "sourceFile", defaultIniFilePath, "destFile", filePath)

	if err := ExtractEmbedFile(defaultIniFilePath, filePath); err != nil {
		return fmt.Errorf("failed to extract default config file %s: %w", filename, err)
	}
	return nil
}

func updateConfigFile(sett *settings.KFDSLSettings) error {
	kfiFileName := sett.ConfigFile.Value()
	kfiFilePath := filepath.Join(viper.GetString("steamcmd-appinstalldir"), "System", kfiFileName)
	tmEnabled := strings.Contains(strings.ToLower(sett.GameMode.Value()), "toygameinfo")

	log.Logger.Debug("Starting server configuration file update",
		"function", "updateConfigFile", "file", kfiFilePath)

	if tmEnabled && !strings.EqualFold(strings.ToLower(kfiFileName), "toygame.ini") {
		log.Logger.Warn("Toy Master game mode is enabled, but the configuration file is not 'ToyGame.ini'. This may cause unexpected behavior",
			"function", "updateConfigFile", "file", kfiFilePath)
	}

	// If the specified configuration file doesn't exists,
	// let's extract the corresponding default file
	if !utils.FileExists(kfiFilePath) {
		defaultIniFileName := "KillingFloor.ini"
		if tmEnabled {
			defaultIniFileName = "ToyGame.ini"
		}

		log.Logger.Debug("Missing server configuration file. Extracting the default one...",
			"function", "updateConfigFile", "file", kfiFilePath, "defaultFileName", defaultIniFileName)

		if err := extractDefaultConfigFile(defaultIniFileName, kfiFilePath); err != nil {
			log.Logger.Warn("Failed to extract the default server configuration file",
				"function", "updateConfigFile", "file", kfiFilePath, "defaultFileName", defaultIniFileName, "error", err)
			return err
		}
		log.Logger.Debug("Default server configuration file successfully extracted",
			"function", "updateConfigFile", "file", kfiFilePath)
	}

	// Read the ini file
	var kfi config.ServerIniFile
	var err error

	// Toy Master support
	if tmEnabled {
		kfi, err = config.NewKFTGIniFile(kfiFilePath)
	} else {
		kfi, err = config.NewKFIniFile(kfiFilePath)
	}
	if err != nil {
		log.Logger.Warn("Failed to read the server configuration file",
			"function", "updateConfigFile", "file", kfiFilePath, "error", err)
		return err
	}

	log.Logger.Debug("Server configuration file successfully loaded",
		"function", "updateConfigFile", "file", kfiFilePath)

	// Generics
	cuList := []configUpdater[any]{
		newConfigUpdater(sett.ServerName.Name(), func() any { return kfi.GetServerName() }, func(v any) bool { return kfi.SetServerName(v.(string)) }, sett.ServerName.Value()),
		newConfigUpdater(sett.ShortName.Name(), func() any { return kfi.GetShortName() }, func(v any) bool { return kfi.SetShortName(v.(string)) }, sett.ShortName.Value()),
		newConfigUpdater(sett.GamePort.Name(), func() any { return kfi.GetGamePort() }, func(v any) bool { return kfi.SetGamePort(v.(int)) }, sett.GamePort.Value()),
		newConfigUpdater(sett.WebAdminPort.Name(), func() any { return kfi.GetWebAdminPort() }, func(v any) bool { return kfi.SetWebAdminPort(v.(int)) }, sett.WebAdminPort.Value()),
		newConfigUpdater(sett.GameSpyPort.Name(), func() any { return kfi.GetGameSpyPort() }, func(v any) bool { return kfi.SetGameSpyPort(v.(int)) }, sett.GameSpyPort.Value()),
		newConfigUpdater(sett.GameDifficulty.Name(), func() any { return kfi.GetGameDifficulty() }, func(v any) bool { return kfi.SetGameDifficulty(v.(int)) }, sett.GameDifficulty.Value()),
		newConfigUpdater(sett.GameLength.Name(), func() any { return kfi.GetGameLength() }, func(v any) bool { return kfi.SetGameLength(v.(int)) }, sett.GameLength.Value()),
		newConfigUpdater(sett.FriendlyFire.Name(), func() any { return kfi.GetFriendlyFireRate() }, func(v any) bool { return kfi.SetFriendlyFireRate(v.(float64)) }, sett.FriendlyFire.Value()),
		newConfigUpdater(sett.MaxPlayers.Name(), func() any { return kfi.GetMaxPlayers() }, func(v any) bool { return kfi.SetMaxPlayers(v.(int)) }, sett.MaxPlayers.Value()),
		newConfigUpdater(sett.MaxSpectators.Name(), func() any { return kfi.GetMaxSpectators() }, func(v any) bool { return kfi.SetMaxSpectators(v.(int)) }, sett.MaxSpectators.Value()),
		newConfigUpdater(sett.Password.Name(), func() any { return kfi.GetPassword() }, func(v any) bool { return kfi.SetPassword(v.(string)) }, sett.Password.Value()),
		newConfigUpdater(sett.Region.Name(), func() any { return kfi.GetRegion() }, func(v any) bool { return kfi.SetRegion(v.(int)) }, sett.Region.Value()),
		newConfigUpdater(sett.AdminName.Name(), func() any { return kfi.GetAdminName() }, func(v any) bool { return kfi.SetAdminName(v.(string)) }, sett.AdminName.Value()),
		newConfigUpdater(sett.AdminMail.Name(), func() any { return kfi.GetAdminMail() }, func(v any) bool { return kfi.SetAdminMail(v.(string)) }, sett.AdminMail.Value()),
		newConfigUpdater(sett.AdminPassword.Name(), func() any { return kfi.GetAdminPassword() }, func(v any) bool { return kfi.SetAdminPassword(v.(string)) }, sett.AdminPassword.Value()),
		newConfigUpdater(sett.MOTD.Value(), func() any { return kfi.GetMOTD() }, func(v any) bool { return kfi.SetMOTD(v.(string)) }, sett.MOTD.Value()),
		newConfigUpdater(sett.SpecimenType.Name(), func() any { return kfi.GetSpecimenType() }, func(v any) bool { return kfi.SetSpecimenType(v.(string)) }, sett.SpecimenType.Value()),
		newConfigUpdater(sett.RedirectURL.Name(), func() any { return kfi.GetRedirectURL() }, func(v any) bool { return kfi.SetRedirectURL(v.(string)) }, sett.RedirectURL.Value()),
		newConfigUpdater(sett.EnableWebAdmin.Name(), func() any { return kfi.IsWebAdminEnabled() }, func(v any) bool { return kfi.SetWebAdminEnabled(v.(bool)) }, sett.EnableWebAdmin.Value()),
		newConfigUpdater(sett.EnableMapVote.Name(), func() any { return kfi.IsMapVoteEnabled() }, func(v any) bool { return kfi.SetMapVoteEnabled(v.(bool)) == nil }, sett.EnableMapVote.Value()),
		newConfigUpdater(sett.MapVoteRepeatLimit.Name(), func() any { return kfi.GetMapVoteRepeatLimit() }, func(v any) bool { return kfi.SetMapVoteRepeatLimit(v.(int)) }, sett.MapVoteRepeatLimit.Value()),
	}
	for _, conf := range cuList {
		currentValue := conf.gv()
		if currentValue != conf.nv {
			if !conf.sv(conf.nv) {
				log.Logger.Warn(fmt.Sprintf("Failed to update the server %s configuration", conf.name),
					"function", "updateConfigFile", "file", kfiFilePath, "confName", conf.name, "confOldValue", currentValue, "confNewValue", conf.nv)
				return fmt.Errorf("[%s]: failed to set the new value: %v", conf.name, conf.nv)
			}
			log.Logger.Debug(fmt.Sprintf("Updated server %s configuration", conf.name),
				"function", "updateConfigFile", "file", kfiFilePath, "confName", conf.name, "confOldValue", currentValue, "confNewValue", conf.nv)
		}
	}

	// Special cases
	currentClientRate := kfi.GetMaxInternetClientRate()
	newClientRate := settings.DefaultMaxInternetClientRate
	if sett.Uncap.Value() {
		newClientRate = 15000
	}
	if currentClientRate != newClientRate && !kfi.SetMaxInternetClientRate(newClientRate) {
		log.Logger.Warn("Failed to update the server MaxInternetClientRate configuration",
			"function", "updateConfigFile", "file", kfiFilePath, "confName", "MaxInternetClientRate", "confOldValue", currentClientRate, "confNewValue", newClientRate)
		return fmt.Errorf("[MaxInternetClientRate]: failed to set the new value: %d", newClientRate)
	}

	if err := updateConfigFileServerMutators(kfi, sett); err != nil {
		return fmt.Errorf("[ServerMutators]: %w", err)
	}

	if err := updateConfigFileMaplist(kfi, sett); err != nil {
		return fmt.Errorf("[Maplist]: %w", err)
	}

	// Save the ini file
	err = kfi.Save(kfiFilePath)
	if err == nil {
		log.Logger.Debug("Server configuration file successfully saved",
			"function", "updateConfigFile", "file", kfiFilePath)
	} else {
		log.Logger.Error("Failed to save the server configuration file",
			"function", "updateConfigFile", "file", kfiFilePath, "error", err)
	}
	return err
}

func updateConfigFileServerMutators(iniFile config.ServerIniFile, sett *settings.KFDSLSettings) error {
	mutatorsStr := sett.ServerMutators.Value()
	mutatorsList := strings.FieldsFunc(mutatorsStr, func(r rune) bool { return r == ',' })

	log.Logger.Debug("Starting server configuration file mutators update",
		"function", "updateConfigFileServerMutators", "file", iniFile.FilePath(), "mutators", mutatorsList)

	// If KFPatcher is enabled, add its mutator to the list if not already present
	if sett.EnableKFPatcher.Value() && !strings.Contains(strings.ToLower(mutatorsStr), "kfpatcher") {
		log.Logger.Debug("KFPatcher is enabled, adding its mutator to the server mutator list",
			"function", "updateConfigFileServerMutators", "file", iniFile.FilePath(), "mutator", "KFPatcher.Mut")
		mutatorsList = append(mutatorsList, "KFPatcher.Mut")
	}

	// Update mutators or clear if empty
	if len(mutatorsList) > 0 {
		if err := iniFile.SetServerMutators(mutatorsList); err != nil {
			log.Logger.Warn("Failed to set server mutators",
				"function", "updateConfigFileServerMutators", "file", iniFile.FilePath(), "mutators", mutatorsList, "error", err)
			return err
		}
		log.Logger.Debug("Server mutators successfully updated",
			"function", "updateConfigFileServerMutators", "file", iniFile.FilePath(), "mutators", mutatorsList)
	} else {
		if err := iniFile.ClearServerMutators(); err != nil {
			log.Logger.Warn("Failed to clear existing server mutators",
				"function", "updateConfigFileServerMutators", "file", iniFile.FilePath(), "error", err)
			return err
		}
		log.Logger.Debug("Server mutators cleared",
			"function", "updateConfigFileServerMutators", "file", iniFile.FilePath())
	}
	return nil
}

func updateConfigFileMaplist(iniFile config.ServerIniFile, sett *settings.KFDSLSettings) error {
	gameMode := sett.GameMode.RawValue()

	log.Logger.Debug("Starting server configuration file maplist update",
		"function", "updateConfigFileMaplist", "file", iniFile.FilePath(), "gameMode", gameMode)

	sectionName := kfserver.GetGameModeMaplistName(gameMode)
	if sectionName == "" {
		log.Logger.Warn("Undefined maplist section name",
			"function", "updateConfigFileMaplist", "file", iniFile.FilePath(), "gameMode", gameMode)
		return fmt.Errorf("undefined section name for game mode: %s", gameMode)
	}

	mapList := strings.FieldsFunc(sett.Maplist.Value(), func(r rune) bool { return r == ',' })

	log.Logger.Debug("Maplist parsed",
		"function", "updateConfigFileMaplist", "file", iniFile.FilePath(), "section", sectionName,
		"gameMode", gameMode, "list", mapList)

	if len(mapList) > 0 {
		if mapList[0] == "all" {
			// Fetch and set all available maps
			gameServerRoot := viper.GetString("steamcmd-appinstalldir")
			gameModePrefix := kfserver.GetGameModeMapPrefix(gameMode)

			installedMaps, err := kfserver.GetInstalledMaps(path.Join(gameServerRoot, "Maps"), gameModePrefix)
			if err != nil {
				log.Logger.Warn("Unable to fetch installed maps",
					"function", "updateConfigFileMaplist", "file", iniFile.FilePath(), "gameMode", gameMode)
				return fmt.Errorf("unable to fetch available maps for game mode '%s': %w", gameMode, err)
			}

			log.Logger.Debug("Using all maps for the current game mode",
				"function", "updateConfigFileMaplist", "file", iniFile.FilePath(), "section", sectionName,
				"gameMode", gameMode, "gameModePrefix", gameModePrefix, "serverRootDir", gameServerRoot, "installedMaps", installedMaps)

			mapList = installedMaps
		}

		// Set the map list in the configuration file
		if err := iniFile.SetMaplist(sectionName, mapList); err != nil {
			log.Logger.Warn("Failed to set maplist",
				"function", "updateConfigFileMaplist", "file", iniFile.FilePath(), "section", sectionName, "error", err)
			return err
		}
		log.Logger.Debug("Maplist successfully updated",
			"function", "updateConfigFileMaplist", "file", iniFile.FilePath(), "section", sectionName)
	} else {
		// Clear the maplist
		if err := iniFile.ClearMaplist(sectionName); err != nil {
			log.Logger.Warn("Failed to clear maplist",
				"function", "updateConfigFileMaplist", "file", iniFile.FilePath(), "section", sectionName, "error", err)
			return err
		}
		log.Logger.Debug("Maplist cleared",
			"function", "updateConfigFileMaplist", "file", iniFile.FilePath(), "section", sectionName)
	}
	return nil
}

func updateKFPatcherConfigFile(sett *settings.KFDSLSettings) error {
	kfpiFilePath := filepath.Join(viper.GetString("steamcmd-appinstalldir"), "System", "KFPatcherSettings.ini")

	log.Logger.Debug("Starting KFPatcher configuration file update",
		"function", "updateKFPatcherConfigFile", "file", kfpiFilePath)

	// Read the ini file
	kfpi, err := config.NewKFPIniFile(kfpiFilePath)
	if err != nil {
		log.Logger.Warn("Failed to read the KFPatcher configuration file",
			"function", "updateKFPatcherConfigFile", "file", kfpiFilePath, "error", err)
		return err
	}

	log.Logger.Debug("KFPatcher configuration file successfully loaded",
		"function", "updateKFPatcherConfigFile", "file", kfpiFilePath)

	cuList := []configUpdater[any]{
		newConfigUpdater(sett.KFPHidePerks.Name(), func() any { return kfpi.IsShowPerksEnabled() }, func(v any) bool { return kfpi.SetShowPerksEnabled(v.(bool)) }, !sett.KFPHidePerks.Value()),
		newConfigUpdater(sett.KFPDisableZedTime.Name(), func() any { return kfpi.IsZEDTimeEnabled() }, func(v any) bool { return kfpi.SetZEDTimeEnabled(v.(bool)) }, !sett.KFPDisableZedTime.Value()),
		newConfigUpdater(sett.KFPEnableAllTraders.Name(), func() any { return kfpi.IsAllTradersOpenEnabled() }, func(v any) bool { return kfpi.SetAllTradersOpenEnabled(v.(bool)) }, sett.KFPEnableAllTraders.Value()),
		newConfigUpdater(sett.KFPAllTradersMessage.Name(), func() any { return kfpi.GetAllTradersMessage() }, func(v any) bool { return kfpi.SetAllTradersMessage(v.(string)) }, sett.KFPAllTradersMessage.Value()),
		newConfigUpdater(sett.KFPBuyEverywhere.Name(), func() any { return kfpi.IsBuyEverywhereEnabled() }, func(v any) bool { return kfpi.SetBuyEverywhereEnabled(v.(bool)) }, sett.KFPBuyEverywhere.Value()),
	}
	for _, conf := range cuList {
		currentValue := conf.gv()
		if currentValue != conf.nv {
			if !conf.sv(conf.nv) {
				log.Logger.Warn(fmt.Sprintf("Failed to update KFPatcher %s configuration", conf.name),
					"function", "updateKFPatcherConfigFile", "file", kfpiFilePath, "confName", conf.name, "confOldValue", currentValue, "confNewValue", conf.nv)
				return fmt.Errorf("[%s]: failed to set the new value: %v", conf.name, conf.nv)
			}
			log.Logger.Debug(fmt.Sprintf("Updated KFPatcher %s configuration", conf.name),
				"function", "updateKFPatcherConfigFile", "file", kfpiFilePath, "confName", conf.name, "confOldValue", currentValue, "confNewValue", conf.nv)
		}
	}

	// Save the ini file
	err = kfpi.Save(kfpiFilePath)
	if err == nil {
		log.Logger.Debug("KFPatcher configuration file successfully saved",
			"function", "updateKFPatcherConfigFile", "file", kfpiFilePath)
	} else {
		log.Logger.Error("Failed to save the KFPatcher configuration file",
			"function", "updateKFPatcherConfigFile", "file", kfpiFilePath, "error", err)
	}
	return err
}

func updateGameServerSteamLibs() ([]string, error) {
	ret := []string{}
	rootDir := viper.GetString("steamcmd-appinstalldir")
	systemDir := path.Join(rootDir, "System")
	libsDir := path.Join(viper.GetString("steamcmd-root"), "linux32")

	log.Logger.Debug("Starting server Steam libraries update",
		"function", "updateGameServerSteamLibs", "rootDir", rootDir, "systemDir", systemDir, "libsDir", libsDir)

	libs := map[string]string{
		path.Join(libsDir, "steamclient.so"):  path.Join(systemDir, "steamclient.so"),
		path.Join(libsDir, "libtier0_s.so"):   path.Join(systemDir, "libtier0_s.so"),
		path.Join(libsDir, "libvstdlib_s.so"): path.Join(systemDir, "libvstdlib_s.so"),
	}

	for srcFile, dstFile := range libs {
		identical, err := utils.SHA1Compare(srcFile, dstFile)
		if err != nil {
			log.Logger.Warn("Error comparing file checksums",
				"function", "updateGameServerSteamLibs", "sourceFile", srcFile, "destFile", dstFile, "error", err)
			return ret, fmt.Errorf("error comparing files %s and %s: %w", srcFile, dstFile, err)
		}

		if !identical {
			log.Logger.Debug("Files differ, updating destination file",
				"function", "updateGameServerSteamLibs", "sourceFile", srcFile, "destFile", dstFile)
			if err := utils.CopyAndReplaceFile(srcFile, dstFile); err != nil {
				return ret, err
			}
			log.Logger.Debug("Successfully updated game server library",
				"function", "updateGameServerSteamLibs", "sourceFile", srcFile, "destFile", dstFile)
			ret = append(ret, dstFile)
		} else {
			log.Logger.Debug("Files are already identical, skipping update",
				"function", "updateGameServerSteamLibs", "sourceFile", srcFile, "destFile", dstFile)
		}
	}

	log.Logger.Debug("Server Steam libraries update complete",
		"function", "updateGameServerSteamLibs", "updatedFilesCount", len(ret))
	return ret, nil
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

func installMods(sett *settings.KFDSLSettings) error {
	filename := sett.ModsFile.Value()

	if filename == "" {
		log.Logger.Info("No mods file specified, skipping mods installation")
		return nil
	}
	if !utils.FileExists(filename) {
		log.Logger.Warn("Mods file not found, skipping mods installation", "file", filename)
		return nil
	}

	log.Logger.Debug("Starting mods installation process")
	m, err := mods.ParseModsFile(filename)
	if err != nil {
		return fmt.Errorf("failed to parse mods file %s: %w", filename, err)
	}

	installed := make([]string, 0)
	mods.InstallMods(viper.GetString("steamcmd-appinstalldir"), m, &installed)

	log.Logger.Debug("Completed mods installation process")
	log.Logger.Info("The following mods were installed:", "mods", strings.Join(installed, " / "))

	return nil
}
