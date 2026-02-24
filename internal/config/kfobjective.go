package config

import (
	"github.com/K4rian/kfdsl/internal/config/ini"
)

type KFObjectiveIniFile struct {
	*KFIniFile
}

func NewKFObjectiveIniFile(filePath string) (ServerIniFile, error) {
	iFile := &KFObjectiveIniFile{
		KFIniFile: &KFIniFile{
			GenericIniFile: ini.NewGenericIniFile("KFObjectiveIniFile"),
			filePath:       filePath,
			gameMode:       "KFStoryGame.KFstoryGameInfo",
		},
	}
	if err := iFile.Load(filePath); err != nil {
		return nil, err
	}
	return iFile, nil
}

func (kf *KFObjectiveIniFile) SetGameLength(length int) bool {
	// Objective mode ignores game length config
	return true
}
