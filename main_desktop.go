//go:build !server

package main

import (
	"github.com/wailsapp/wails/v3/pkg/application"
)

// runApplication creates the native desktop window and blocks until the app exits.
func runApplication(app *App) error {
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

	return wailsApp.Run()
}
