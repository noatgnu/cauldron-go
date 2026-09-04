package services

import (
	"context"
	"fmt"
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
	service := newSettingsServiceInternal(db)
	service.ctx = ctx
	return service
}

func newSettingsServiceInternal(db *DatabaseService) *SettingsService {
	service := &SettingsService{
		db: db,
		config: &models.Config{
			CurtainBackendURL: "https://celsus.muttsu.xyz",
			PluginRegistryURL: "https://cauldron.proteo.info",
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
	if val, ok := settings["useRenvCache"]; ok {
		s.config.UseRenvCache = val == "true"
	}
	if val, ok := settings["venvStoragePath"]; ok {
		s.config.VenvStoragePath = val
	}
	if val, ok := settings["renvStoragePath"]; ok {
		s.config.RenvStoragePath = val
	}
	if val, ok := settings["accessibility.fontScale"]; ok {
		s.config.AccessibilityFontScale = val
	}
	if val, ok := settings["accessibility.highContrast"]; ok {
		s.config.AccessibilityHighContrast = val == "true"
	}
	if val, ok := settings["accessibility.reducedMotion"]; ok {
		s.config.AccessibilityReducedMotion = val == "true"
	}
	if val, ok := settings["accessibility.colorblindPalette"]; ok {
		s.config.AccessibilityColorblindPalette = val
	}
	if val, ok := settings["debugMode"]; ok {
		s.config.DebugMode = val == "true"
	}
	if val, ok := settings["autoCheckForUpdates"]; ok {
		s.config.AutoCheckForUpdates = val == "true"
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
	s.db.SaveSetting("useRenvCache", fmt.Sprintf("%v", s.config.UseRenvCache))
	s.db.SaveSetting("venvStoragePath", s.config.VenvStoragePath)
	s.db.SaveSetting("renvStoragePath", s.config.RenvStoragePath)
	s.db.SaveSetting("accessibility.fontScale", s.config.AccessibilityFontScale)
	s.db.SaveSetting("accessibility.highContrast", fmt.Sprintf("%v", s.config.AccessibilityHighContrast))
	s.db.SaveSetting("accessibility.reducedMotion", fmt.Sprintf("%v", s.config.AccessibilityReducedMotion))
	s.db.SaveSetting("accessibility.colorblindPalette", s.config.AccessibilityColorblindPalette)
	s.db.SaveSetting("debugMode", fmt.Sprintf("%v", s.config.DebugMode))
	s.db.SaveSetting("autoCheckForUpdates", fmt.Sprintf("%v", s.config.AutoCheckForUpdates))
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
	case "useRenvCache":
		return s.config.UseRenvCache
	case "venvStoragePath":
		return s.config.VenvStoragePath
	case "renvStoragePath":
		return s.config.RenvStoragePath
	case "accessibility.fontScale":
		return s.config.AccessibilityFontScale
	case "accessibility.highContrast":
		return s.config.AccessibilityHighContrast
	case "accessibility.reducedMotion":
		return s.config.AccessibilityReducedMotion
	case "accessibility.colorblindPalette":
		return s.config.AccessibilityColorblindPalette
	case "debugMode":
		return s.config.DebugMode
	case "autoCheckForUpdates":
		return s.config.AutoCheckForUpdates
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
	case "useRenvCache":
		s.config.UseRenvCache = value.(bool)
	case "venvStoragePath":
		s.config.VenvStoragePath = value.(string)
		if s.config.VenvStoragePath != "" {
			os.MkdirAll(s.config.VenvStoragePath, 0755)
		}
	case "renvStoragePath":
		s.config.RenvStoragePath = value.(string)
		if s.config.RenvStoragePath != "" {
			os.MkdirAll(s.config.RenvStoragePath, 0755)
		}
	case "accessibility.fontScale":
		s.config.AccessibilityFontScale = value.(string)
	case "accessibility.highContrast":
		s.config.AccessibilityHighContrast = value.(bool)
	case "accessibility.reducedMotion":
		s.config.AccessibilityReducedMotion = value.(bool)
	case "accessibility.colorblindPalette":
		s.config.AccessibilityColorblindPalette = value.(string)
	case "debugMode":
		s.config.DebugMode = value.(bool)
	case "autoCheckForUpdates":
		s.config.AutoCheckForUpdates = value.(bool)
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

	if s.config.AccessibilityFontScale == "" {
		s.config.AccessibilityFontScale = "100"
	}
	if s.config.AccessibilityColorblindPalette == "" {
		s.config.AccessibilityColorblindPalette = "default"
	}

	// Check the DB directly (not the in-memory config) so an explicit "false" isn't mistaken for "never set".
	if val, _ := s.db.GetSetting("autoCheckForUpdates"); val == "" {
		s.config.AutoCheckForUpdates = true
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
