package models

type Config struct {
	ResultStoragePath string `json:"resultStoragePath"`
	OutputDirectory   string `json:"outputDirectory"`
	PythonPath        string `json:"pythonPath"`
	RPath             string `json:"rPath"`
	RLibPath          string `json:"rLibPath"`
	CurtainBackendURL string `json:"curtainBackendUrl"`
}
