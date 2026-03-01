package launcher

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/K4rian/kfdsl/embed"
	"github.com/K4rian/kfdsl/internal/config"
	"github.com/K4rian/kfdsl/internal/log"
	"github.com/K4rian/kfdsl/internal/services/kfserver"
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

func (l *Launcher) updateConfigFile() error {
	kfiFileName := l.settings.ConfigFile.Value()
	kfiFilePath := filepath.Join(l.settings.ServerInstallDir.Value(), "System", kfiFileName)

	useObjectiveMode := strings.Contains(strings.ToLower(l.settings.GameMode.Value()), "storygameinfo")
	useToyMasterMode := strings.Contains(strings.ToLower(l.settings.GameMode.Value()), "toygameinfo")

	log.Logger.Debug("Starting server configuration file update",
		"function", "updateConfigFile", "file", kfiFilePath)

	// If the specified configuration file doesn't exists,
	// let's extract the corresponding default file
	if !utils.FileExists(kfiFilePath) {
		defaultIniFileName := "KillingFloor.ini"

		log.Logger.Debug("Missing server configuration file. Extracting the default one...",
			"function", "updateConfigFile", "file", kfiFilePath, "defaultFileName", defaultIniFileName)

		if err := l.extractDefaultConfigFile(defaultIniFileName, kfiFilePath); err != nil {
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

	// Objective
	if useObjectiveMode {
		kfi, err = config.NewKFObjectiveIniFile(kfiFilePath)
	} else if useToyMasterMode {
		// Toy Master
		kfi, err = config.NewKFTGIniFile(kfiFilePath)
	} else {
		// Survival
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
		newConfigUpdater(l.settings.ServerName.Name(), func() any { return kfi.GetServerName() }, func(v any) bool { return kfi.SetServerName(v.(string)) }, l.settings.ServerName.Value()),
		newConfigUpdater(l.settings.ShortName.Name(), func() any { return kfi.GetShortName() }, func(v any) bool { return kfi.SetShortName(v.(string)) }, l.settings.ShortName.Value()),
		newConfigUpdater(l.settings.GamePort.Name(), func() any { return kfi.GetGamePort() }, func(v any) bool { return kfi.SetGamePort(v.(int)) }, l.settings.GamePort.Value()),
		newConfigUpdater(l.settings.WebAdminPort.Name(), func() any { return kfi.GetWebAdminPort() }, func(v any) bool { return kfi.SetWebAdminPort(v.(int)) }, l.settings.WebAdminPort.Value()),
		newConfigUpdater(l.settings.GameSpyPort.Name(), func() any { return kfi.GetGameSpyPort() }, func(v any) bool { return kfi.SetGameSpyPort(v.(int)) }, l.settings.GameSpyPort.Value()),
		newConfigUpdater(l.settings.GameDifficulty.Name(), func() any { return kfi.GetGameDifficulty() }, func(v any) bool { return kfi.SetGameDifficulty(v.(int)) }, l.settings.GameDifficulty.Value()),
		newConfigUpdater(l.settings.GameLength.Name(), func() any { return kfi.GetGameLength() }, func(v any) bool { return kfi.SetGameLength(v.(int)) }, l.settings.GameLength.Value()),
		newConfigUpdater(l.settings.FriendlyFire.Name(), func() any { return kfi.GetFriendlyFireRate() }, func(v any) bool { return kfi.SetFriendlyFireRate(v.(float64)) }, l.settings.FriendlyFire.Value()),
		newConfigUpdater(l.settings.MaxPlayers.Name(), func() any { return kfi.GetMaxPlayers() }, func(v any) bool { return kfi.SetMaxPlayers(v.(int)) }, l.settings.MaxPlayers.Value()),
		newConfigUpdater(l.settings.MaxSpectators.Name(), func() any { return kfi.GetMaxSpectators() }, func(v any) bool { return kfi.SetMaxSpectators(v.(int)) }, l.settings.MaxSpectators.Value()),
		newConfigUpdater(l.settings.Password.Name(), func() any { return kfi.GetPassword() }, func(v any) bool { return kfi.SetPassword(v.(string)) }, l.settings.Password.Value()),
		newConfigUpdater(l.settings.Region.Name(), func() any { return kfi.GetRegion() }, func(v any) bool { return kfi.SetRegion(v.(int)) }, l.settings.Region.Value()),
		newConfigUpdater(l.settings.AdminName.Name(), func() any { return kfi.GetAdminName() }, func(v any) bool { return kfi.SetAdminName(v.(string)) }, l.settings.AdminName.Value()),
		newConfigUpdater(l.settings.AdminMail.Name(), func() any { return kfi.GetAdminMail() }, func(v any) bool { return kfi.SetAdminMail(v.(string)) }, l.settings.AdminMail.Value()),
		newConfigUpdater(l.settings.AdminPassword.Name(), func() any { return kfi.GetAdminPassword() }, func(v any) bool { return kfi.SetAdminPassword(v.(string)) }, l.settings.AdminPassword.Value()),
		newConfigUpdater(l.settings.MOTD.Name(), func() any { return kfi.GetMOTD() }, func(v any) bool { return kfi.SetMOTD(v.(string)) }, l.settings.MOTD.Value()),
		newConfigUpdater(l.settings.SpecimenType.Name(), func() any { return kfi.GetSpecimenType() }, func(v any) bool { return kfi.SetSpecimenType(v.(string)) }, l.settings.SpecimenType.Value()),
		newConfigUpdater(l.settings.RedirectURL.Name(), func() any { return kfi.GetRedirectURL() }, func(v any) bool { return kfi.SetRedirectURL(v.(string)) }, l.settings.RedirectURL.Value()),
		newConfigUpdater(l.settings.EnableWebAdmin.Name(), func() any { return kfi.IsWebAdminEnabled() }, func(v any) bool { return kfi.SetWebAdminEnabled(v.(bool)) }, l.settings.EnableWebAdmin.Value()),
		newConfigUpdater(l.settings.EnableMapVote.Name(), func() any { return kfi.IsMapVoteEnabled() }, func(v any) bool { return kfi.SetMapVoteEnabled(v.(bool)) == nil }, l.settings.EnableMapVote.Value()),
		newConfigUpdater(l.settings.MapVoteRepeatLimit.Name(), func() any { return kfi.GetMapVoteRepeatLimit() }, func(v any) bool { return kfi.SetMapVoteRepeatLimit(v.(int)) }, l.settings.MapVoteRepeatLimit.Value()),
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
	if l.settings.Uncap.Value() {
		newClientRate = 15000
	}
	if currentClientRate != newClientRate && !kfi.SetMaxInternetClientRate(newClientRate) {
		log.Logger.Warn("Failed to update the server MaxInternetClientRate configuration",
			"function", "updateConfigFile", "file", kfiFilePath, "confName", "MaxInternetClientRate", "confOldValue", currentClientRate, "confNewValue", newClientRate)
		return fmt.Errorf("[MaxInternetClientRate]: failed to set the new value: %d", newClientRate)
	}

	if err := l.updateConfigFileServerMutators(kfi); err != nil {
		return fmt.Errorf("[ServerMutators]: %w", err)
	}

	if err := l.updateConfigFileMaplist(kfi); err != nil {
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

func (l *Launcher) updateConfigFileServerMutators(iniFile config.ServerIniFile) error {
	mutatorsStr := l.settings.ServerMutators.Value()
	mutatorsList := strings.FieldsFunc(mutatorsStr, func(r rune) bool { return r == ',' })

	log.Logger.Debug("Starting server configuration file mutators update",
		"function", "updateConfigFileServerMutators", "file", iniFile.FilePath(), "mutators", mutatorsList)

	// If KFPatcher is enabled, add its mutator to the list if not already present
	if l.settings.EnableKFPatcher.Value() && !strings.Contains(strings.ToLower(mutatorsStr), "kfpatcher") {
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

func (l *Launcher) updateConfigFileMaplist(iniFile config.ServerIniFile) error {
	gameMode := l.settings.GameMode.RawValue()

	log.Logger.Debug("Starting server configuration file maplist update",
		"function", "updateConfigFileMaplist", "file", iniFile.FilePath(), "gameMode", gameMode)

	sectionName := kfserver.GetGameModeMaplistName(gameMode)
	if sectionName == "" {
		log.Logger.Warn("Undefined maplist section name",
			"function", "updateConfigFileMaplist", "file", iniFile.FilePath(), "gameMode", gameMode)
		return fmt.Errorf("undefined section name for game mode: %s", gameMode)
	}

	mapList := strings.FieldsFunc(l.settings.Maplist.Value(), func(r rune) bool { return r == ',' })

	log.Logger.Debug("Maplist parsed",
		"function", "updateConfigFileMaplist", "file", iniFile.FilePath(), "section", sectionName,
		"gameMode", gameMode, "list", mapList)

	if len(mapList) > 0 {
		if mapList[0] == "all" {
			// Fetch and set all available maps
			gameServerRoot := l.settings.ServerInstallDir.Value()
			gameModePrefix := kfserver.GetGameModeMapPrefix(gameMode)

			installedMaps, err := kfserver.GetInstalledMaps(filepath.Join(gameServerRoot, "Maps"), gameModePrefix)
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

func (l *Launcher) updateKFPatcherConfigFile() error {
	kfpiFilePath := filepath.Join(l.settings.ServerInstallDir.Value(), "System", "KFPatcherSettings.ini")

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
		newConfigUpdater(l.settings.KFPHidePerks.Name(), func() any { return kfpi.IsShowPerksEnabled() }, func(v any) bool { return kfpi.SetShowPerksEnabled(v.(bool)) }, !l.settings.KFPHidePerks.Value()),
		newConfigUpdater(l.settings.KFPDisableZedTime.Name(), func() any { return kfpi.IsZEDTimeEnabled() }, func(v any) bool { return kfpi.SetZEDTimeEnabled(v.(bool)) }, !l.settings.KFPDisableZedTime.Value()),
		newConfigUpdater(l.settings.KFPEnableAllTraders.Name(), func() any { return kfpi.IsAllTradersOpenEnabled() }, func(v any) bool { return kfpi.SetAllTradersOpenEnabled(v.(bool)) }, l.settings.KFPEnableAllTraders.Value()),
		newConfigUpdater(l.settings.KFPAllTradersMessage.Name(), func() any { return kfpi.GetAllTradersMessage() }, func(v any) bool { return kfpi.SetAllTradersMessage(v.(string)) }, l.settings.KFPAllTradersMessage.Value()),
		newConfigUpdater(l.settings.KFPBuyEverywhere.Name(), func() any { return kfpi.IsBuyEverywhereEnabled() }, func(v any) bool { return kfpi.SetBuyEverywhereEnabled(v.(bool)) }, l.settings.KFPBuyEverywhere.Value()),
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

func (l *Launcher) extractDefaultConfigFile(filename string, filePath string) error {
	defaultIniFilePath := filepath.Join("assets/configs", filename)

	log.Logger.Debug("Extracting default configuration file",
		"function", "extractDefaultConfigFile", "sourceFile", defaultIniFilePath, "destFile", filePath)

	if err := embed.ExtractFile(defaultIniFilePath, filePath); err != nil {
		return fmt.Errorf("failed to extract default config file %s: %w", filename, err)
	}
	return nil
}
