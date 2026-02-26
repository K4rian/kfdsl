package embed

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/K4rian/kfdsl/internal/utils"
)

//go:embed assets/configs/*
var embeddedFiles embed.FS

func ExtractFile(embedPath, targetPath string) error {
	if utils.FileExists(targetPath) {
		return nil
	}

	// Read embedded file
	data, err := embeddedFiles.ReadFile(embedPath)
	if err != nil {
		return fmt.Errorf("failed to read embedded file: %w", err)
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(targetPath), 0644); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write file to disk
	err = os.WriteFile(targetPath, data, fs.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	return nil
}
