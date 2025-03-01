package dependencies

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

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

type Dependency struct {
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
}

func ParseDependencies(filename string) (map[string]*Dependency, error) {
	jsonFile, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer jsonFile.Close()

	var items map[string]*Dependency
	err = json.NewDecoder(jsonFile).Decode(&items)
	if err != nil {
		return nil, err
	}

	return items, nil
}

func (d *Dependency) installDependencies(dir string, deps map[string]*Dependency, installed map[string]bool) error {
	for _, name := range d.DependOn {
		dep, ok := deps[name]
		if !ok {
			return fmt.Errorf("dependency %s not found", name)
		}
		if err := dep.Install(dir, name, deps, installed); err != nil {
			return err
		}
	}
	return nil
}

func (d *Dependency) isDownloadRequired(dir string) bool {
	for _, item := range d.InstallItems {
		if item.Checksum == "" {
			continue
		}

		itemPath := filepath.Join(dir, item.Path, item.Name)
		if utils.FileExists(itemPath) {
			if match, err := utils.FileMatchesChecksum(itemPath, item.Checksum); err != nil || !match {
				log.Logger.Debug("Checksum mismatch, download required", "path", itemPath, "checksum", item.Checksum)
				return true
			}
		}
	}

	return false
}

func (d *Dependency) download(dir, name string) (string, error) {
	if !d.isDownloadRequired(dir) {
		return "", nil
	}

	log.Logger.Debug("Downloading dependency", "name", name, "url", d.DownloadURL)
	filename, err := utils.DownloadFile(d.DownloadURL, d.Checksum)
	if err != nil {
		return "", fmt.Errorf("failed to download %s: %w", name, err)
	}
	log.Logger.Debug("Dependency download complete", "name", name)
	return filename, nil
}

func (d *Dependency) installFile(dir, filename string, item InstallItem) error {
	log.Logger.Debug("Installing dependency file", "name", item.Name, "dir", dir, "path", item.Path, "from", filename)
	path, err := utils.CreateDirIfNotExists(dir, item.Path)
	if err != nil {
		return err
	}
	return utils.MoveFile(filename, filepath.Join(path, item.Name), item.Checksum)
}

func (d *Dependency) installFiles(dir, filename string) error {
	if len(d.InstallItems) > 1 && !d.Extract {
		return fmt.Errorf("dependency contains multiple files but is not marked for extraction")
	}

	log.Logger.Debug("Installing dependency files")
	if len(d.InstallItems) == 1 {
		return d.installFile(dir, filename, d.InstallItems[0])
	}
	return d.installArchive(dir, filename)
}

func (d *Dependency) installArchive(dir, archive string) error {
	// Unpack item in temporary directory then move them one by one
	tempDir, err := os.MkdirTemp("", "*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	log.Logger.Debug("Extracting dependency archive", "archive", archive, "to", tempDir)
	if err := utils.UnzipFile(archive, tempDir); err != nil {
		return err
	}

	for _, item := range d.InstallItems {
		if err := d.installFile(dir, filepath.Join(tempDir, item.Name), item); err != nil {
			return err
		}
	}
	return nil
}

func (d *Dependency) Install(dir string, name string, deps map[string]*Dependency, installed map[string]bool) error {
	log.Logger.Debug("Installing dependency", "name", name)

	if alreadyInstalled, ok := installed[name]; ok && alreadyInstalled {
		log.Logger.Debug("Dependency already installed, no actions needed", "name", name)
		return nil
	}

	if err := d.installDependencies(dir, deps, installed); err != nil {
		return err
	}

	filename, err := d.download(dir, name)
	if err != nil {
		return err
	}

	if filename != "" {
		if err := d.installFiles(dir, filename); err != nil {
			return err
		}
	}

	installed[name] = true
	return nil
}
