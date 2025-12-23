package services

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	goruntime "runtime"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type ProtocolHandler struct {
	pluginInstaller *PluginInstaller
	ctx             context.Context
}

func NewProtocolHandler(installer *PluginInstaller) *ProtocolHandler {
	return &ProtocolHandler{
		pluginInstaller: installer,
	}
}

func (ph *ProtocolHandler) SetContext(ctx context.Context) {
	ph.ctx = ctx
}

func (ph *ProtocolHandler) RegisterProtocol() error {
	if goruntime.GOOS != "windows" {
		log.Printf("[ProtocolHandler] Protocol registration only supported on Windows")
		return nil
	}

	execPath, err := getExecutablePath()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	type regCommand struct {
		args []string
		desc string
	}

	commands := []regCommand{
		{
			args: []string{"ADD", `HKCU\Software\Classes\cauldron`, "/ve", "/d", "URL:Cauldron Protocol", "/f"},
			desc: "Create protocol key",
		},
		{
			args: []string{"ADD", `HKCU\Software\Classes\cauldron`, "/v", "URL Protocol", "/d", "", "/f"},
			desc: "Set URL Protocol value",
		},
		{
			args: []string{"ADD", `HKCU\Software\Classes\cauldron\DefaultIcon`, "/ve", "/d", execPath + ",1", "/f"},
			desc: "Set default icon",
		},
		{
			args: []string{"ADD", `HKCU\Software\Classes\cauldron\shell\open\command`, "/ve", "/d", fmt.Sprintf(`"%s" "%%1"`, execPath), "/f"},
			desc: "Set command handler",
		},
	}

	for _, regCmd := range commands {
		cmd := exec.Command("REG", regCmd.args...)
		hideConsoleWindow(cmd)
		output, err := cmd.CombinedOutput()
		if err != nil {
			log.Printf("[ProtocolHandler] Warning: Failed to %s: %v", regCmd.desc, err)
			log.Printf("[ProtocolHandler] Command: REG %v", regCmd.args)
			log.Printf("[ProtocolHandler] Output: %s", string(output))
			return fmt.Errorf("failed to %s: %w", regCmd.desc, err)
		}
		log.Printf("[ProtocolHandler] Successfully: %s", regCmd.desc)
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

	log.Printf("[ProtocolHandler] Install request for: %s", repoURL)

	isInstalled, err := ph.pluginInstaller.IsPluginInstalled(repoURL)
	if err != nil {
		log.Printf("[ProtocolHandler] Failed to check if plugin is installed: %v", err)
		runtime.EventsEmit(ph.ctx, "plugin:install:error", map[string]interface{}{
			"repo":  repoURL,
			"error": err.Error(),
		})
		return fmt.Errorf("failed to check if plugin is installed: %w", err)
	}

	if isInstalled {
		runtime.EventsEmit(ph.ctx, "plugin:install:error", map[string]interface{}{
			"repo":  repoURL,
			"error": "Plugin from this repository is already installed",
		})
		return fmt.Errorf("plugin from this repository is already installed")
	}

	pluginInfo, err := ph.pluginInstaller.FetchPluginInfo(repoURL)
	if err != nil {
		log.Printf("[ProtocolHandler] Failed to fetch plugin info: %v", err)
		runtime.EventsEmit(ph.ctx, "plugin:install:error", map[string]interface{}{
			"repo":  repoURL,
			"error": fmt.Sprintf("Failed to fetch plugin information: %v", err),
		})
		return fmt.Errorf("failed to fetch plugin information: %w", err)
	}

	log.Printf("[ProtocolHandler] Requesting user confirmation for plugin: %s", pluginInfo.Plugin.Name)
	runtime.EventsEmit(ph.ctx, "plugin:install:request", map[string]interface{}{
		"repo":        repoURL,
		"name":        pluginInfo.Plugin.Name,
		"id":          pluginInfo.Plugin.ID,
		"version":     pluginInfo.Plugin.Version,
		"author":      pluginInfo.Plugin.Author,
		"description": pluginInfo.Plugin.Description,
		"category":    pluginInfo.Plugin.Category,
	})

	return nil
}

func getExecutablePath() (string, error) {
	if goruntime.GOOS == "windows" {
		path, err := os.Executable()
		if err != nil {
			log.Printf("[ProtocolHandler] Failed to get executable path: %v", err)
			return "", err
		}

		path = strings.ReplaceAll(path, "/", "\\")

		log.Printf("[ProtocolHandler] Executable path: %s", path)
		return path, nil
	}

	return "", fmt.Errorf("protocol registration not supported on %s", goruntime.GOOS)
}
