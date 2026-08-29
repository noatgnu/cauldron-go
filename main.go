package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed all:frontend/dist/browser
var assets embed.FS

//go:embed resources/appicon.png
var iconPNG []byte

func getAssets() fs.FS {
	subFS, err := fs.Sub(assets, "frontend/dist/browser")
	if err != nil {
		log.Printf("WARNING: Failed to get assets: %v\n", err)
		return assets
	}
	return subFS
}

func getMimeType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	mimeTypes := map[string]string{
		".html":  "text/html; charset=utf-8",
		".css":   "text/css; charset=utf-8",
		".js":    "application/javascript; charset=utf-8",
		".mjs":   "application/javascript; charset=utf-8",
		".json":  "application/json; charset=utf-8",
		".png":   "image/png",
		".jpg":   "image/jpeg",
		".jpeg":  "image/jpeg",
		".gif":   "image/gif",
		".svg":   "image/svg+xml",
		".ico":   "image/x-icon",
		".woff":  "font/woff",
		".woff2": "font/woff2",
		".ttf":   "font/ttf",
		".eot":   "application/vnd.ms-fontobject",
		".map":   "application/json",
	}
	if mime, ok := mimeTypes[ext]; ok {
		return mime
	}
	return "application/octet-stream"
}

func newSPAHandler(assetsFS fs.FS) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}

		// Try to read the file from the embedded filesystem
		content, err := fs.ReadFile(assetsFS, path)
		if err == nil {
			w.Header().Set("Content-Type", getMimeType(path))
			w.WriteHeader(http.StatusOK)
			w.Write(content)
			return
		}

		// If the file is not found and it's not a static asset (likely a SPA route),
		// fall back to index.html
		isStaticAsset := func(path string) bool {
			ext := strings.ToLower(filepath.Ext(path))
			staticExts := []string{".js", ".css", ".png", ".jpg", ".jpeg", ".gif", ".svg", ".ico", ".woff", ".woff2", ".ttf", ".eot", ".map", ".json"}
			for _, e := range staticExts {
				if ext == e {
					return true
				}
			}
			return false
		}

		if !isStaticAsset(path) {
			indexContent, err := fs.ReadFile(assetsFS, "index.html")
			if err == nil {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.WriteHeader(http.StatusOK)
				w.Write(indexContent)
				return
			}
		}

		http.NotFound(w, r)
	})
}

func setupLogRotation(logDir string) (*os.File, error) {
	os.MkdirAll(logDir, 0755)

	today := time.Now().Format("2006-01-02")
	logFileName := fmt.Sprintf("cauldron-%s.log", today)
	logFilePath := filepath.Join(logDir, logFileName)

	logFile, err := os.OpenFile(
		logFilePath,
		os.O_CREATE|os.O_WRONLY|os.O_APPEND,
		0666,
	)
	if err != nil {
		return nil, err
	}

	cleanOldLogs(logDir, 7)

	return logFile, nil
}

func cleanOldLogs(logDir string, maxFiles int) {
	files, err := os.ReadDir(logDir)
	if err != nil {
		return
	}

	var logFiles []string
	for _, file := range files {
		if !file.IsDir() && strings.HasPrefix(file.Name(), "cauldron-") && strings.HasSuffix(file.Name(), ".log") {
			logFiles = append(logFiles, file.Name())
		}
	}

	if len(logFiles) <= maxFiles {
		return
	}

	sort.Strings(logFiles)

	filesToDelete := len(logFiles) - maxFiles
	for i := 0; i < filesToDelete; i++ {
		os.Remove(filepath.Join(logDir, logFiles[i]))
	}
}

func main() {
	if runCLI(os.Args[1:]) {
		return
	}

	userConfigDir, _ := os.UserConfigDir()
	logDir := filepath.Join(userConfigDir, "cauldron")

	logFile, err := setupLogRotation(logDir)
	var logFilePath string
	if err == nil {
		log.SetOutput(logFile)
		defer logFile.Close()
		logFilePath = logFile.Name()
	}

	log.Println("========================================")
	log.Println("Cauldron starting...")
	log.Printf("Log directory: %s\n", logDir)
	fmt.Println("Cauldron starting - logs at:", logDir)

	app := NewApp()
	if logFilePath != "" {
		app.SetLogFilePath(logFilePath)
		log.Printf("Log file path set to: %s\n", logFilePath)
	}

	log.Println("Creating Wails application...")

	wailsApp := application.New(application.Options{
		Name:        "Cauldron",
		Description: "Proteomics data visualization and analysis",
		Icon:        iconPNG,
		Services: []application.Service{
			application.NewService(app),
		},
		Assets: application.AssetOptions{
			Handler: newSPAHandler(getAssets()),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
		OnShutdown: func() {
			app.Shutdown()
		},
	})

	app.SetApplication(wailsApp)

	appMenu := createApplicationMenu(app)
	wailsApp.Menu.SetApplicationMenu(appMenu)

	mainWindow := wailsApp.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "Cauldron",
		Width:            1280,
		Height:           800,
		URL:              "/",
		BackgroundColour: application.NewRGB(27, 38, 54),
		Mac: application.MacWindow{
			TitleBar: application.MacTitleBar{
				AppearsTransparent: true,
			},
		},
	})

	app.SetMainWindow(mainWindow)
	mainWindow.SetMenu(appMenu)

	go app.Initialize()

	err = wailsApp.Run()
	if err != nil {
		log.Printf("ERROR: Wails.Run failed: %v\n", err)
		println("Error:", err.Error())
	}

	log.Println("Cauldron exiting")
}
