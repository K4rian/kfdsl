package launcher

import (
	"fmt"
	"strings"

	"github.com/K4rian/kfdsl/internal/log"
	"github.com/K4rian/kfdsl/internal/mods"
	"github.com/K4rian/kfdsl/internal/utils"
)

func (l *Launcher) installMods() error {
	filename := l.settings.ModsFile.Value()

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
	mods.InstallMods(l.settings.ServerInstallDir.Value(), m, &installed)

	log.Logger.Debug("Completed mods installation process")
	log.Logger.Info("The following mods were installed:", "mods", strings.Join(installed, " / "))

	return nil
}
