package services

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/noatgnu/cauldron-go/backend/models"
)

type SettingsService struct {
	ctx    context.Context
	db     *DatabaseService
	config *models.Config
}

func NewSettingsService(ctx context.Context, db *DatabaseService) *SettingsService {
	service := &SettingsService{
		ctx: ctx,
		db:  db,
		config: &models.Config{
			CurtainBackendURL: "https://celsus.muttsu.xyz",
		},
	}

	service.Load()
	service.initializeDefaults()
	service.Save()

	return service
}

func (s *SettingsService) Load() error {
	settings, err := s.db.GetAllSettings()
	if err != nil {
		return err
	}

	if val, ok := settings["resultStoragePath"]; ok {
		s.config.ResultStoragePath = val
	}
	if val, ok := settings["outputDirectory"]; ok {
		s.config.OutputDirectory = val
	}
	if val, ok := settings["pythonPath"]; ok {
		s.config.PythonPath = val
	}
	if val, ok := settings["rPath"]; ok {
		s.config.RPath = val
	}
	if val, ok := settings["rLibPath"]; ok {
		s.config.RLibPath = val
	}
	if val, ok := settings["curtainBackendUrl"]; ok {
		s.config.CurtainBackendURL = val
	}

	return nil
}

func (s *SettingsService) Save() error {
	s.db.SaveSetting("resultStoragePath", s.config.ResultStoragePath)
	s.db.SaveSetting("outputDirectory", s.config.OutputDirectory)
	s.db.SaveSetting("pythonPath", s.config.PythonPath)
	s.db.SaveSetting("rPath", s.config.RPath)
	s.db.SaveSetting("rLibPath", s.config.RLibPath)
	s.db.SaveSetting("curtainBackendUrl", s.config.CurtainBackendURL)
	return nil
}

func (s *SettingsService) Get(key string) interface{} {
	switch key {
	case "resultStoragePath":
		return s.config.ResultStoragePath
	case "outputDirectory":
		return s.config.OutputDirectory
	case "pythonPath":
		return s.config.PythonPath
	case "rPath":
		return s.config.RPath
	case "rLibPath":
		return s.config.RLibPath
	case "curtainBackendUrl":
		return s.config.CurtainBackendURL
	}
	return nil
}

func (s *SettingsService) Set(key string, value interface{}) error {
	switch key {
	case "resultStoragePath":
		s.config.ResultStoragePath = value.(string)
	case "outputDirectory":
		s.config.OutputDirectory = value.(string)
		os.MkdirAll(s.config.OutputDirectory, 0755)
	case "pythonPath":
		s.config.PythonPath = value.(string)
	case "rPath":
		s.config.RPath = value.(string)
	case "rLibPath":
		s.config.RLibPath = value.(string)
	case "curtainBackendUrl":
		s.config.CurtainBackendURL = value.(string)
	}
	return s.Save()
}

func (s *SettingsService) GetConfig() *models.Config {
	return s.config
}

func (s *SettingsService) initializeDefaults() {
	if s.config.ResultStoragePath == "" {
		userConfigDir, _ := os.UserConfigDir()
		s.config.ResultStoragePath = filepath.Join(userConfigDir, "cauldron", "results")
		os.MkdirAll(s.config.ResultStoragePath, 0755)
	}

	if s.config.OutputDirectory == "" {
		homeDir, _ := os.UserHomeDir()
		documentsDir := filepath.Join(homeDir, "Documents")
		if runtime.GOOS == "windows" {
			documentsDir = filepath.Join(homeDir, "Documents")
		} else if runtime.GOOS == "darwin" {
			documentsDir = filepath.Join(homeDir, "Documents")
		} else {
			documentsDir = filepath.Join(homeDir, "Documents")
		}
		s.config.OutputDirectory = filepath.Join(documentsDir, "CauldronOutputs")
		os.MkdirAll(s.config.OutputDirectory, 0755)
	}

	if s.config.CurtainBackendURL == "" {
		s.config.CurtainBackendURL = "https://celsus.muttsu.xyz"
	}
}

func (s *SettingsService) DetectPythonPath() (string, error) {
	execPath, err := os.Executable()
	if err == nil {
		bundledPython := ""
		if runtime.GOOS == "windows" {
			bundledPython = filepath.Join(filepath.Dir(execPath), "resources", "python", "python.exe")
		} else {
			bundledPython = filepath.Join(filepath.Dir(execPath), "resources", "python", "bin", "python3")
		}

		if _, err := os.Stat(bundledPython); err == nil {
			return bundledPython, nil
		}
	}

	pythonCmd := "python3"
	if runtime.GOOS == "windows" {
		pythonCmd = "python"
	}

	path, err := exec.LookPath(pythonCmd)
	if err == nil {
		return path, nil
	}

	return "", err
}

func (s *SettingsService) DetectRPath() (string, error) {
	execPath, err := os.Executable()
	if err == nil {
		bundledR := ""
		if runtime.GOOS == "windows" {
			bundledR = filepath.Join(filepath.Dir(execPath), "resources", "r", "bin", "Rscript.exe")
		} else {
			bundledR = filepath.Join(filepath.Dir(execPath), "resources", "r", "bin", "Rscript")
		}

		if _, err := os.Stat(bundledR); err == nil {
			rLibPath := filepath.Join(filepath.Dir(execPath), "resources", "r", "library")
			s.config.RLibPath = rLibPath
			return bundledR, nil
		}
	}

	path, err := exec.LookPath("Rscript")
	if err == nil {
		return path, nil
	}

	return "", err
}
