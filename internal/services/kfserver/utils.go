package kfserver

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

func GetInstalledMaps(dir string, prefix string) ([]string, error) {
	var filteredFiles []string

	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		fileName := file.Name()
		if strings.HasPrefix(strings.ToLower(fileName), "kf-menu") || !strings.HasPrefix(fileName, prefix) {
			continue
		}

		ext := filepath.Ext(fileName)
		if ext != ".rom" {
			continue
		}
		filteredFiles = append(filteredFiles, strings.TrimSuffix(fileName, ext))
	}
	return filteredFiles, nil
}

func GetGameModeMapPrefix(gamemode string) string {
	modes := map[string]string{
		"survival":  "KF-",
		"objective": "KFO-",
		"toymaster": "TOY-",
	}
	return modes[strings.ToLower(gamemode)]
}

func GetGameModeMaplistName(gamemode string) string {
	mlist := map[string]string{
		"survival":  "KFMod.KFMaplist",
		"objective": "KFStoryGame.KFOMapList",
		"toymaster": "KFCharPuppets.TOYMapList",
	}
	return mlist[strings.ToLower(gamemode)]
}

func GetSeasonalSpecimenType() string {
	currentMonth := time.Now().Month()

	switch currentMonth {
	case time.June, time.July, time.August:
		return "ET_SummerSideshow"
	case time.October:
		return "ET_HillbillyHorror"
	case time.December:
		return "ET_TwistedChristmas"
	default:
		return "ET_None"
	}
}
