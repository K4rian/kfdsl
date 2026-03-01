package steamcmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/K4rian/kfdsl/internal/services/base"
	"github.com/K4rian/kfdsl/internal/utils"
)

type SteamCMD struct {
	*base.BaseService
	executable string
}

const (
	relScriptPath = "steamcmd.sh"
)

func New(ctx context.Context, rootDir string) *SteamCMD {
	// opts := base.DefaultServiceOptions()
	// opts.RootDirectory = rootDir
	// opts.WorkingDirectory = rootDir
	// opts.AutoRestart = false

	return &SteamCMD{
		BaseService: base.NewBaseService("SteamCMD", ctx, base.ServiceOptions{
			RootDirectory:    rootDir,
			WorkingDirectory: rootDir,
			AutoRestart:      false,
		}),
		executable: filepath.Join(rootDir, relScriptPath),
	}
}

func (s *SteamCMD) Run(args ...string) error {
	args = append([]string{s.executable}, args...)
	return s.BaseService.Start(args)
}

func (s *SteamCMD) RunScript(fileName string) error {
	if !utils.FileExists(fileName) {
		return fmt.Errorf("script file %s not found", fileName)
	}
	return s.Run("+runscript", fileName, "+quit")
}

func (s *SteamCMD) WriteScript(fileName string, loginUser string, loginPassword string, installDir string, appID int, validate bool) error {
	file, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("cannot create script file %s: %v", fileName, err)
	}
	defer file.Close()

	validateStr := "validate"
	if !validate {
		validateStr = ""
	}

	content := fmt.Sprintf(
		"force_install_dir %s\nlogin %s %s\napp_update %d %s\nquit",
		installDir,
		loginUser,
		loginPassword,
		appID,
		validateStr,
	)

	if _, err = file.WriteString(content); err != nil {
		return fmt.Errorf("cannot write script file %s: %v", fileName, err)
	}
	return nil
}

func (s *SteamCMD) IsInstalled() bool {
	return utils.FileExists(s.executable)
}

func (s *SteamCMD) IsReady() bool {
	return s.IsInstalled()
}
