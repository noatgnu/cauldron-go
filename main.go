package main

import (
	"embed"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist/browser
var assets embed.FS

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
	userConfigDir, _ := os.UserConfigDir()
	logDir := filepath.Join(userConfigDir, "cauldron")

	logFile, err := setupLogRotation(logDir)
	if err == nil {
		log.SetOutput(logFile)
		defer logFile.Close()
	}

	log.Println("========================================")
	log.Println("Cauldron starting...")
	log.Printf("Log directory: %s\n", logDir)
	fmt.Println("Cauldron starting - logs at:", logDir)

	// Create an instance of the app structure
	app := NewApp()

	log.Println("Creating Wails application...")

	// Create application with options
	err = wails.Run(&options.App{
		Title:  "Cauldron",
		Width:  1280,
		Height: 800,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		log.Printf("ERROR: Wails.Run failed: %v\n", err)
		println("Error:", err.Error())
	}

	log.Println("Cauldron exiting")
}
