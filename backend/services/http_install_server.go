package services

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

const installServerPort = 50060

type HTTPInstallServer struct {
	server          *http.Server
	protocolHandler *ProtocolHandler
	port            int
	ctx             context.Context
	wailsApp        *application.App
}

func (h *HTTPInstallServer) emitEvent(name string, data interface{}) {
	if h.wailsApp != nil && h.wailsApp.Event != nil {
		h.wailsApp.Event.Emit(name, data)
	}
}

func (h *HTTPInstallServer) SetWailsApp(wailsApp *application.App) {
	h.wailsApp = wailsApp
}

func NewHTTPInstallServer(handler *ProtocolHandler) *HTTPInstallServer {
	return &HTTPInstallServer{
		protocolHandler: handler,
		port:            installServerPort,
	}
}

func (h *HTTPInstallServer) Start(ctx context.Context) error {
	h.ctx = ctx

	mux := http.NewServeMux()

	mux.HandleFunc("/install", h.handleInstall)
	mux.HandleFunc("/", h.handleRoot)
	mux.HandleFunc("/health", h.handleHealth)

	h.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", h.port),
		Handler: mux,
	}

	listener, err := net.Listen("tcp", h.server.Addr)
	if err != nil {
		return fmt.Errorf("failed to start HTTP install server: %w", err)
	}

	log.Printf("[HTTPInstallServer] Started on http://localhost:%d", h.port)
	log.Printf("[HTTPInstallServer] Install endpoint: http://localhost:%d/install?repo=<github-repo-url>", h.port)

	go func() {
		if err := h.server.Serve(listener); err != nil && err != http.ErrServerClosed {
			log.Printf("[HTTPInstallServer] Server error: %v", err)
		}
	}()

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := h.server.Shutdown(shutdownCtx); err != nil {
			log.Printf("[HTTPInstallServer] Shutdown error: %v", err)
		}
	}()

	return nil
}

func (h *HTTPInstallServer) Stop() error {
	if h.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return h.server.Shutdown(ctx)
	}
	return nil
}

func (h *HTTPInstallServer) GetPort() int {
	return h.port
}

func (h *HTTPInstallServer) handleInstall(w http.ResponseWriter, r *http.Request) {
	repoURL := r.URL.Query().Get("repo")

	if repoURL == "" {
		h.renderError(w, "Missing 'repo' parameter", "Please provide a repository URL in the 'repo' query parameter")
		return
	}

	log.Printf("[HTTPInstallServer] Install request for: %s", repoURL)

	go func() {
		isInstalled, err := h.protocolHandler.pluginInstaller.IsPluginInstalled(repoURL)
		if err != nil {
			log.Printf("[HTTPInstallServer] Failed to check if plugin is installed: %v", err)
			h.emitEvent("plugin:install:error", map[string]interface{}{
				"repo":  repoURL,
				"error": err.Error(),
			})
			return
		}

		if isInstalled {
			h.emitEvent("plugin:install:error", map[string]interface{}{
				"repo":  repoURL,
				"error": "Plugin from this repository is already installed",
			})
			return
		}

		pluginInfo, err := h.protocolHandler.pluginInstaller.FetchPluginInfo(repoURL)
		if err != nil {
			log.Printf("[HTTPInstallServer] Failed to fetch plugin info: %v", err)
			h.emitEvent("plugin:install:error", map[string]interface{}{
				"repo":  repoURL,
				"error": fmt.Sprintf("Failed to fetch plugin information: %v", err),
			})
			return
		}

		log.Printf("[HTTPInstallServer] Requesting user confirmation for plugin: %s", pluginInfo.Plugin.Name)

		hasPythonDeps := false
		hasRDeps := false

		hasPythonDeps = pluginInfo.Execution.Requirements.PythonRequirementsFile != "" ||
			(len(pluginInfo.Execution.Requirements.Packages) > 0 && pluginInfo.Runtime.HasEnvironment("python"))
		hasRDeps = pluginInfo.Execution.Requirements.RPackagesFile != "" ||
			(pluginInfo.Execution.Requirements.R != "" && pluginInfo.Runtime.HasEnvironment("r"))

		h.emitEvent("plugin:install:request", map[string]interface{}{
			"repo":                repoURL,
			"name":                pluginInfo.Plugin.Name,
			"id":                  pluginInfo.Plugin.ID,
			"version":             pluginInfo.Plugin.Version,
			"author":              pluginInfo.Plugin.Author,
			"description":         pluginInfo.Plugin.Description,
			"category":            pluginInfo.Plugin.Category,
			"runtimeEnvironments": pluginInfo.Runtime.Environments,
			"hasPythonDeps":       hasPythonDeps,
			"hasRDeps":            hasRDeps,
		})
	}()

	h.renderConfirmationRequest(w, repoURL)
}

func (h *HTTPInstallServer) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	html := `<!DOCTYPE html>
<html>
<head>
	<title>Cauldron Plugin Installer</title>
	<style>
		body {
			font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
			max-width: 800px;
			margin: 50px auto;
			padding: 20px;
			background: #f5f5f5;
		}
		.container {
			background: white;
			padding: 40px;
			border-radius: 8px;
			box-shadow: 0 2px 10px rgba(0,0,0,0.1);
		}
		h1 {
			color: #333;
			margin-top: 0;
		}
		.info {
			background: #e3f2fd;
			padding: 15px;
			border-radius: 4px;
			border-left: 4px solid #2196f3;
			margin: 20px 0;
		}
		code {
			background: #f5f5f5;
			padding: 2px 6px;
			border-radius: 3px;
			font-family: 'Courier New', monospace;
		}
		.example {
			background: #f5f5f5;
			padding: 15px;
			border-radius: 4px;
			margin: 15px 0;
			overflow-x: auto;
		}
		a {
			color: #2196f3;
			text-decoration: none;
		}
		a:hover {
			text-decoration: underline;
		}
	</style>
</head>
<body>
	<div class="container">
		<h1>🧪 Cauldron Plugin Installer</h1>

		<div class="info">
			<strong>ℹ️ Server is running</strong><br>
			This HTTP server allows you to install Cauldron plugins via simple HTTP links.
		</div>

		<h2>How to Use</h2>
		<p>To install a plugin, use the following URL format:</p>

		<div class="example">
			<code>http://localhost:` + fmt.Sprintf("%d", h.port) + `/install?repo=&lt;github-repo-url&gt;</code>
		</div>

		<h3>Example:</h3>
		<div class="example">
			<code>http://localhost:` + fmt.Sprintf("%d", h.port) + `/install?repo=https://github.com/username/plugin-repo</code>
		</div>

		<h2>For Plugin Developers</h2>
		<p>You can provide installation links in your README.md files on GitHub:</p>

		<div class="example">
			<code>&lt;a href="http://localhost:` + fmt.Sprintf("%d", h.port) + `/install?repo=https://github.com/username/your-plugin"&gt;Install Plugin&lt;/a&gt;</code>
		</div>

		<p style="margin-top: 30px; color: #666; font-size: 14px;">
			Note: This server only runs when Cauldron is active.
		</p>
	</div>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, html)
}

func (h *HTTPInstallServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"status":"ok","port":%d}`, h.port)
}

func (h *HTTPInstallServer) renderConfirmationRequest(w http.ResponseWriter, repoURL string) {
	tmpl := template.Must(template.New("confirmation").Parse(`<!DOCTYPE html>
<html>
<head>
	<title>Confirm Installation</title>
	<style>
		body {
			font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
			max-width: 600px;
			margin: 50px auto;
			padding: 20px;
			background: #f5f5f5;
		}
		.container {
			background: white;
			padding: 40px;
			border-radius: 8px;
			box-shadow: 0 2px 10px rgba(0,0,0,0.1);
			text-align: center;
		}
		h1 {
			color: #2196f3;
			margin: 0 0 20px 0;
		}
		.repo-url {
			background: #f5f5f5;
			padding: 10px;
			border-radius: 4px;
			margin: 20px 0;
			word-break: break-all;
			font-family: 'Courier New', monospace;
			font-size: 14px;
		}
		.info {
			color: #666;
			margin-top: 20px;
			line-height: 1.6;
		}
		.highlight {
			background: #fff3cd;
			padding: 15px;
			border-radius: 4px;
			border-left: 4px solid #ffc107;
			margin: 20px 0;
		}
		.security-note {
			background: #e3f2fd;
			padding: 15px;
			border-radius: 4px;
			border-left: 4px solid #2196f3;
			margin: 20px 0;
			font-size: 14px;
		}
	</style>
	<script>
		setTimeout(function() {
			// Try to close the window/tab
			window.close();
			// If window.close() doesn't work (e.g., for tabs not opened by script),
			// redirect to the root page after a brief delay
			setTimeout(function() {
				if (!window.closed) {
					window.location.href = '/';
				}
			}, 100);
		}, 3000);
	</script>
</head>
<body>
	<div class="container">
		<h1>Confirmation Required</h1>
		<p>Plugin installation request from:</p>
		<div class="repo-url">{{.RepoURL}}</div>
		<div class="highlight">
			<strong>Check the Cauldron application to review and confirm</strong>
		</div>
		<div class="security-note">
			For your security, you'll be asked to review the plugin details before installation proceeds.
		</div>
		<div class="info">
			A confirmation dialog will appear in the Cauldron application<br>
			showing the plugin name, author, and source repository.
		</div>
		<p style="color: #999; font-size: 14px; margin-top: 30px;">
			This window will close automatically in 3 seconds...
		</p>
	</div>
</body>
</html>`))

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl.Execute(w, map[string]interface{}{
		"RepoURL": repoURL,
	})
}

func (h *HTTPInstallServer) renderError(w http.ResponseWriter, title string, message string) {
	tmpl := template.Must(template.New("error").Parse(`<!DOCTYPE html>
<html>
<head>
	<title>Installation Error</title>
	<style>
		body {
			font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
			max-width: 600px;
			margin: 50px auto;
			padding: 20px;
			background: #f5f5f5;
		}
		.container {
			background: white;
			padding: 40px;
			border-radius: 8px;
			box-shadow: 0 2px 10px rgba(0,0,0,0.1);
			text-align: center;
		}
		.error-icon {
			font-size: 64px;
			margin-bottom: 20px;
		}
		h1 {
			color: #f44336;
			margin: 0 0 10px 0;
		}
		.error-message {
			background: #ffebee;
			padding: 15px;
			border-radius: 4px;
			margin: 20px 0;
			color: #c62828;
			border-left: 4px solid #f44336;
		}
		.back-btn {
			margin-top: 30px;
			padding: 10px 20px;
			background: #2196f3;
			color: white;
			border: none;
			border-radius: 4px;
			cursor: pointer;
			font-size: 16px;
			text-decoration: none;
			display: inline-block;
		}
		.back-btn:hover {
			background: #1976d2;
		}
	</style>
</head>
<body>
	<div class="container">
		<div class="error-icon">❌</div>
		<h1>{{.Title}}</h1>
		<div class="error-message">{{.Message}}</div>
		<a href="/" class="back-btn">Back to Home</a>
	</div>
</body>
</html>`))

	w.WriteHeader(http.StatusBadRequest)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl.Execute(w, map[string]interface{}{
		"Title":   title,
		"Message": message,
	})
}
