//go:build server

package main

import (
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// runApplication starts the headless HTTP server and blocks until shutdown.
// Host/port default to localhost:8080 and can be overridden with the
// WAILS_SERVER_HOST / WAILS_SERVER_PORT environment variables.
func runApplication(app *App) error {
	wailsApp := application.New(application.Options{
		Name:        "Cauldron",
		Description: "Proteomics data visualization and analysis",
		Services: []application.Service{
			application.NewService(app),
		},
		Assets: application.AssetOptions{
			Handler:    newServerHandler(app, getAssets()),
			Middleware: authMiddleware(app),
		},
		Server: application.ServerOptions{},
		OnShutdown: func() {
			app.Shutdown()
		},
	})

	app.SetApplication(wailsApp)

	go app.Initialize()

	log.Println("Server mode: no native window or menu; the app is served over HTTP.")

	return wailsApp.Run()
}
