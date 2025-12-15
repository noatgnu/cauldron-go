package services

import (
	"fmt"
	"log"
	"net/url"
	"os/exec"
	"runtime"
)

type ProtocolHandler struct {
	pluginInstaller *PluginInstaller
}

func NewProtocolHandler(installer *PluginInstaller) *ProtocolHandler {
	return &ProtocolHandler{
		pluginInstaller: installer,
	}
}

func (ph *ProtocolHandler) RegisterProtocol() error {
	if runtime.GOOS != "windows" {
		log.Printf("[ProtocolHandler] Protocol registration only supported on Windows")
		return nil
	}

	execPath, err := getExecutablePath()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	commands := []string{
		fmt.Sprintf(`REG ADD "HKCU\Software\Classes\cauldron" /ve /d "URL:Cauldron Protocol" /f`),
		fmt.Sprintf(`REG ADD "HKCU\Software\Classes\cauldron" /v "URL Protocol" /d "" /f`),
		fmt.Sprintf(`REG ADD "HKCU\Software\Classes\cauldron\DefaultIcon" /ve /d "%s,1" /f`, execPath),
		fmt.Sprintf(`REG ADD "HKCU\Software\Classes\cauldron\shell\open\command" /ve /d "\"%s\" \"%%1\"" /f`, execPath),
	}

	for _, cmdStr := range commands {
		cmd := exec.Command("cmd", "/C", cmdStr)
		if err := cmd.Run(); err != nil {
			log.Printf("[ProtocolHandler] Warning: Failed to register protocol: %v", err)
			return fmt.Errorf("failed to register protocol: %w", err)
		}
	}

	log.Printf("[ProtocolHandler] Protocol handler registered successfully")
	return nil
}

func (ph *ProtocolHandler) HandleURL(urlStr string) error {
	log.Printf("[ProtocolHandler] Handling URL: %s", urlStr)

	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	if parsedURL.Scheme != "cauldron" {
		return fmt.Errorf("unsupported protocol scheme: %s", parsedURL.Scheme)
	}

	switch parsedURL.Host {
	case "install":
		return ph.handleInstall(parsedURL)
	default:
		return fmt.Errorf("unknown action: %s", parsedURL.Host)
	}
}

func (ph *ProtocolHandler) handleInstall(parsedURL *url.URL) error {
	query := parsedURL.Query()
	repoURL := query.Get("repo")

	if repoURL == "" {
		return fmt.Errorf("missing 'repo' parameter")
	}

	log.Printf("[ProtocolHandler] Installing plugin from: %s", repoURL)

	if err := ph.pluginInstaller.InstallPlugin(repoURL); err != nil {
		return fmt.Errorf("failed to install plugin: %w", err)
	}

	log.Printf("[ProtocolHandler] Plugin installed successfully from: %s", repoURL)
	return nil
}

func getExecutablePath() (string, error) {
	if runtime.GOOS == "windows" {
		cmd := exec.Command("where", "cauldron-go.exe")
		output, err := cmd.Output()
		if err == nil && len(output) > 0 {
			return string(output), nil
		}
	}

	cmd := exec.Command("powershell", "-Command", "(Get-Process -Id $PID).Path")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return string(output), nil
}
