package mods

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/K4rian/kfdsl/internal/log"
	"github.com/K4rian/kfdsl/internal/utils"
)

type Author struct {
	Name    string `json:"name"`
	Website string `json:"website"`
}

type InstallItem struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	Type     string `json:"type"`
	Checksum string `json:"checksum,omitempty"`
}

type Mod struct {
	Version      string        `json:"version"`
	Description  string        `json:"description"`
	Authors      []Author      `json:"authors"`
	License      string        `json:"license"`
	ProjectURL   string        `json:"project_url"`
	DownloadURL  string        `json:"download_url"`
	Checksum     string        `json:"checksum,omitempty"`
	Extract      bool          `json:"extract"`
	InstallItems []InstallItem `json:"install"`
	DependOn     []string      `json:"depend_on"`
	Enabled      bool          `json:"enabled,omitempty"`
}

type installResult struct {
	name string
	err  error
}

func (m *Mod) isDownloadRequired(dir string) bool {
	for _, item := range m.InstallItems {
		if item.Checksum == "" {
			continue
		}

		itemPath := filepath.Join(dir, item.Path, item.Name)
		log.Logger.Debug("Checking mod file", "path", itemPath, "checksum", item.Checksum)

		if !utils.FileExists(itemPath) {
			return true
		}

		if match, err := utils.FileMatchesChecksum(itemPath, item.Checksum); err != nil || !match {
			log.Logger.Debug("Checksum mismatch, download required", "path", itemPath, "checksum", item.Checksum)
			return true
		}
	}

	return false
}

func (m *Mod) download(dir, name string) (string, error) {
	if !m.isDownloadRequired(dir) {
		return "", nil
	}

	log.Logger.Debug("Downloading mod", "name", name, "url", m.DownloadURL)
	filename, err := utils.DownloadFile(m.DownloadURL, m.Checksum)
	if err != nil {
		return "", fmt.Errorf("failed to download %s: %w", name, err)
	}
	log.Logger.Debug("Mod download complete", "name", name)
	return filename, nil
}

func (m *Mod) installFile(dir, filename string, item InstallItem) error {
	log.Logger.Debug("Installing mod file", "name", item.Name, "dir", dir, "path", item.Path, "from", filename)
	path, err := utils.CreateDirIfNotExists(dir, item.Path)
	if err != nil {
		return err
	}
	return utils.MoveFile(filename, filepath.Join(path, item.Name), item.Checksum)
}

func (m *Mod) installFiles(dir, filename string) error {
	if len(m.InstallItems) > 1 && !m.Extract {
		return fmt.Errorf("mod contains multiple files but is not marked for extraction")
	}

	log.Logger.Debug("Installing mod files")
	if len(m.InstallItems) == 1 {
		return m.installFile(dir, filename, m.InstallItems[0])
	}
	return m.installArchive(dir, filename)
}

func (m *Mod) installArchive(dir, archive string) error {
	// Unpack item in temporary directory then move them one by one
	tempDir, err := os.MkdirTemp("", "*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	log.Logger.Debug("Extracting mod archive", "archive", archive, "to", tempDir)
	if err := utils.UnzipFile(archive, tempDir); err != nil {
		return err
	}

	for _, item := range m.InstallItems {
		if err := m.installFile(dir, filepath.Join(tempDir, item.Name), item); err != nil {
			return err
		}
	}
	return nil
}

func (m *Mod) install(dir string, name string) error {
	if !m.Enabled {
		log.Logger.Debug("Skipping installation of mod, it is disabled", "name", name)
		return nil
	}

	if !m.isDownloadRequired(dir) {
		log.Logger.Debug("Skipping installation of mod, it is already installed", "name", name)
		return nil
	}

	log.Logger.Debug("Installing mod", "name", name)

	filename, err := m.download(dir, name)
	if err != nil {
		return err
	}

	if filename != "" {
		if err := m.installFiles(dir, filename); err != nil {
			return err
		}
	}

	return nil
}

func (m *Mod) resolveDependencies(allMods map[string]*Mod, visited map[string]bool) []string {
	if len(m.DependOn) == 0 {
		return nil
	}

	var deps []string
	for _, name := range m.DependOn {
		if visited[name] {
			log.Logger.Warn("Circular dependency detected, skipping", "name", name)
			continue
		}
		dep, ok := allMods[name]
		if !ok {
			log.Logger.Warn("Dependency not found", "name", name)
			continue
		}
		visited[name] = true
		dep.Enabled = true
		deps = append(deps, name)
		deps = append(deps, dep.resolveDependencies(allMods, visited)...)
	}
	return deps
}

func resolveModsToInstall(allMods map[string]*Mod) []string {
	var m []string
	for name, mod := range allMods {
		if !mod.Enabled {
			continue
		}
		visited := map[string]bool{name: true}
		deps := mod.resolveDependencies(allMods, visited)
		m = append(m, deps...)
		m = append(m, name)
	}
	return utils.RemoveDuplicates(m)
}

// buildInstallWaves groups mods into waves where each wave's mods
// have all their dependencies satisfied by previous waves.
func buildInstallWaves(modList map[string]*Mod, toInstall []string) [][]string {
	remaining := make(map[string]bool)
	for _, name := range toInstall {
		remaining[name] = true
	}

	var waves [][]string
	for len(remaining) > 0 {
		var wave []string
		for _, name := range toInstall {
			if !remaining[name] {
				continue
			}
			mod := modList[name]
			ready := true
			for _, dep := range mod.DependOn {
				if remaining[dep] {
					ready = false
					break
				}
			}
			if ready {
				wave = append(wave, name)
			}
		}
		if len(wave) == 0 {
			// Unresolvable deps — break to avoid infinite loop
			for name := range remaining {
				wave = append(wave, name)
			}
		}
		for _, name := range wave {
			delete(remaining, name)
		}
		waves = append(waves, wave)
	}
	return waves
}

func InstallMods(dir string, modList map[string]*Mod, installed *[]string) error {
	toInstall := resolveModsToInstall(modList)
	log.Logger.Debug("Mods to install", "mods", strings.Join(toInstall, " / "))

	waves := buildInstallWaves(modList, toInstall)
	var allErrs []error

	for i, wave := range waves {
		log.Logger.Debug("Installing mod wave", "wave", i+1, "mods", strings.Join(wave, " / "))

		results := make(chan installResult, len(wave))
		var wg sync.WaitGroup

		for _, name := range wave {
			mod := modList[name]
			wg.Add(1)
			go func(name string, mod *Mod) {
				defer wg.Done()
				results <- installResult{name, mod.install(dir, name)}
			}(name, mod)
		}

		wg.Wait()
		close(results)

		for r := range results {
			if r.err != nil {
				log.Logger.Error("Failed to install mod", "name", r.name, "error", r.err)
				allErrs = append(allErrs, fmt.Errorf("%s: %w", r.name, r.err))
			} else {
				*installed = append(*installed, r.name)
			}
		}
	}

	if len(allErrs) > 0 {
		return errors.Join(allErrs...)
	}
	return nil
}

func ParseModsFile(filename string) (map[string]*Mod, error) {
	jsonFile, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer jsonFile.Close()

	var items map[string]*Mod
	err = json.NewDecoder(jsonFile).Decode(&items)
	if err != nil {
		return nil, err
	}

	return items, nil
}
