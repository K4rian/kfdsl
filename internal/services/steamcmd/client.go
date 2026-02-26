package steamcmd

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/K4rian/kfdsl/internal/services/base"
	"github.com/K4rian/kfdsl/internal/utils"
)

type SteamCMD struct {
	*base.BaseService
}

func NewSteamCMD(rootDir string, ctx context.Context) *SteamCMD {
	scmd := &SteamCMD{
		BaseService: base.NewBaseService("SteamCMD", rootDir, ctx),
	}
	return scmd
}

func (s *SteamCMD) Run(args ...string) error {
	args = append([]string{filepath.Join(s.RootDirectory(), "steamcmd.sh")}, args...)
	return s.BaseService.Start(args, false)
}

func (s *SteamCMD) RunScript(fileName string) error {
	if !utils.FileExists(fileName) {
		return fmt.Errorf("script file %s not found", fileName)
	}

	args := []string{
		"+runscript", fileName,
		"+quit",
	}
	return s.Run(args...)
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

func (s *SteamCMD) IsAvailable() bool {
	return utils.FileExists(path.Join(s.RootDirectory(), "steamcmd.sh"))
}
