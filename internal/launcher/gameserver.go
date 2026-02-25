package launcher

import (
	"context"
	"fmt"
	"path"
	"path/filepath"

	"github.com/spf13/viper"

	"github.com/K4rian/kfdsl/internal/log"
	"github.com/K4rian/kfdsl/internal/services/kfserver"
	"github.com/K4rian/kfdsl/internal/settings"
	"github.com/K4rian/kfdsl/internal/utils"
)

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
