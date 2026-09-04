package models

type Config struct {
	ResultStoragePath              string `json:"resultStoragePath"`
	OutputDirectory                string `json:"outputDirectory"`
	PythonPath                     string `json:"pythonPath"`
	RPath                          string `json:"rPath"`
	RLibPath                       string `json:"rLibPath"`
	CurtainBackendURL              string `json:"curtainBackendUrl"`
	PluginRegistryURL              string `json:"pluginRegistryUrl"`
	UseRenvCache                   bool   `json:"useRenvCache"`
	VenvStoragePath                string `json:"venvStoragePath"`
	RenvStoragePath                string `json:"renvStoragePath"`
	AccessibilityFontScale         string `json:"accessibilityFontScale"`
	AccessibilityHighContrast      bool   `json:"accessibilityHighContrast"`
	AccessibilityReducedMotion     bool   `json:"accessibilityReducedMotion"`
	AccessibilityColorblindPalette string `json:"accessibilityColorblindPalette"`
	DebugMode                      bool   `json:"debugMode"`
	AutoCheckForUpdates            bool   `json:"autoCheckForUpdates"`
}
