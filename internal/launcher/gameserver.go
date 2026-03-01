package launcher

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/K4rian/kfdsl/internal/log"
	"github.com/K4rian/kfdsl/internal/services/kfserver"
	"github.com/K4rian/kfdsl/internal/utils"
)

func (l *Launcher) startGameServer(ctx context.Context) (*kfserver.KFServer, error) {
	mutators := l.settings.Mutators.Value()
	if l.settings.EnableMutLoader.Value() {
		mutators = "MutLoader.MutLoader"
	}

	rootDir := l.settings.ServerInstallDir.Value()
	configFileName := l.settings.ConfigFile.Value()
	gameServer := kfserver.New(ctx, l.settings)

	log.Logger.Debug("Initializing KF Dedicated Server",
		"function", "startGameServer",
		"rootDir", rootDir,
		"startupMap", l.settings.StartupMap.Value(),
		"gameMode", l.settings.GameMode.Value(),
		"unsecure", l.settings.Unsecure.Value(),
		"maxPlayers", l.settings.MaxPlayers.Value(),
		"mutators", mutators,
		"extraArgs", l.settings.ExtraArgs,
	)

	if !gameServer.IsInstalled() {
		return nil, fmt.Errorf("unable to locate the KF Dedicated Server files in '%s', please install using SteamCMD", gameServer.Options().RootDirectory)
	}

	log.Logger.Info("Updating the KF Dedicated Server configuration file...", "file", configFileName)
	if err := l.updateConfigFile(); err != nil {
		return nil, fmt.Errorf("failed to update the KF Dedicated Server configuration file %s: %w", configFileName, err)
	}
	log.Logger.Info("Server configuration file successfully updated", "file", configFileName)

	if err := l.installMods(); err != nil {
		log.Logger.Error("Failed to install mods", "file", l.settings.ModsFile.Value(), "error", err)
		return nil, fmt.Errorf("failed to install mods: %w", err)
	}

	if l.settings.EnableKFPatcher.Value() {
		kfpConfigFilePath := filepath.Join(rootDir, "System", "KFPatcherSettings.ini")
		log.Logger.Info("Updating the KFPatcher configuration file...", "file", kfpConfigFilePath)
		if err := l.updateKFPatcherConfigFile(); err != nil {
			return nil, fmt.Errorf("failed to update the KFPatcher configuration file %s: %w", kfpConfigFilePath, err)
		}
		log.Logger.Info("KFPatcher configuration file successfully updated", "file", kfpConfigFilePath)
	}

	log.Logger.Info("Verifying KF Dedicated Server Steam libraries for updates...")
	updatedLibs, err := l.updateGameServerSteamLibs()
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

	log.Logger.Info("Starting the KF Dedicated Server...", "rootDir", gameServer.Options().RootDirectory, "autoRestart", l.settings.AutoRestart.Value())
	if err := gameServer.Start(); err != nil {
		return nil, fmt.Errorf("failed to start the KF Dedicated Server: %w", err)
	}
	return gameServer, nil
}

func (l *Launcher) updateGameServerSteamLibs() ([]string, error) {
	ret := []string{}
	rootDir := l.settings.ServerInstallDir.Value()
	systemDir := filepath.Join(rootDir, "System")
	libsDir := filepath.Join(l.settings.SteamCMDRoot.Value(), "linux32")

	log.Logger.Debug("Starting server Steam libraries update",
		"function", "updateGameServerSteamLibs", "rootDir", rootDir, "systemDir", systemDir, "libsDir", libsDir)

	libs := map[string]string{
		filepath.Join(libsDir, "steamclient.so"):  filepath.Join(systemDir, "steamclient.so"),
		filepath.Join(libsDir, "libtier0_s.so"):   filepath.Join(systemDir, "libtier0_s.so"),
		filepath.Join(libsDir, "libvstdlib_s.so"): filepath.Join(systemDir, "libvstdlib_s.so"),
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
