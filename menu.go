package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed resources/menu-icons/*.png
var menuIconsFS embed.FS

// menuIcon returns the PNG bytes for a menu-icons/<name>.png asset. The set
// of names is fixed at build time (see cmd/menu-icon-generator), so a lookup
// failure here means the embed and this call site have drifted apart.
func menuIcon(name string) []byte {
	data, err := menuIconsFS.ReadFile("resources/menu-icons/" + name + ".png")
	if err != nil {
		log.Printf("[createApplicationMenu] Missing menu icon %q: %v", name, err)
		return nil
	}
	return data
}

func createApplicationMenu(app *App) *application.Menu {
	menu := application.NewMenu()

	fileMenu := menu.AddSubmenu("File")
	fileMenu.Add("Import Data").
		SetAccelerator("CmdOrCtrl+O").
		SetBitmap(menuIcon("import-data")).
		OnClick(func(ctx *application.Context) {
			if app.wailsApp != nil {
				app.wailsApp.Event.Emit("menu:import-data", nil)
			}
		})
	fileMenu.AddSeparator()
	fileMenu.Add("Settings").
		SetAccelerator("CmdOrCtrl+,").
		SetBitmap(menuIcon("settings")).
		OnClick(func(ctx *application.Context) {
			if app.wailsApp != nil {
				app.wailsApp.Event.Emit("menu:settings", nil)
			}
		})
	fileMenu.AddSeparator()
	fileMenu.Add("Quit").
		SetAccelerator("CmdOrCtrl+Q").
		SetBitmap(menuIcon("quit")).
		OnClick(func(ctx *application.Context) {
			app.HandleQuit()
		})

	viewMenu := menu.AddSubmenu("View")
	viewMenu.Add("Home").
		SetAccelerator("CmdOrCtrl+1").
		SetBitmap(menuIcon("home")).
		OnClick(func(ctx *application.Context) {
			if app.wailsApp != nil {
				app.wailsApp.Event.Emit("menu:view-home", nil)
			}
		})
	viewMenu.Add("Jobs").
		SetAccelerator("CmdOrCtrl+2").
		SetBitmap(menuIcon("jobs")).
		OnClick(func(ctx *application.Context) {
			if app.wailsApp != nil {
				app.wailsApp.Event.Emit("menu:view-jobs", nil)
			}
		})
	viewMenu.Add("Installed Plugins").
		SetAccelerator("CmdOrCtrl+3").
		SetBitmap(menuIcon("plugins")).
		OnClick(func(ctx *application.Context) {
			if app.wailsApp != nil {
				app.wailsApp.Event.Emit("menu:view-plugin-list", nil)
			}
		})

	windowMenu := menu.AddSubmenu("Window")
	windowMenu.Add("Minimize").
		SetAccelerator("CmdOrCtrl+M").
		SetBitmap(menuIcon("minimize")).
		OnClick(func(ctx *application.Context) {
			if app.mainWindow != nil {
				app.mainWindow.Minimise()
			}
		})
	windowMenu.Add("Zoom").
		SetBitmap(menuIcon("zoom")).
		OnClick(func(ctx *application.Context) {
			if app.mainWindow != nil {
				app.mainWindow.ToggleMaximise()
			}
		})

	helpMenu := menu.AddSubmenu("Help")
	helpMenu.Add("About Cauldron").
		SetBitmap(menuIcon("about")).
		OnClick(func(ctx *application.Context) {
			if app.wailsApp != nil {
				app.wailsApp.Event.Emit("menu:about", nil)
			}
		})
	helpMenu.Add("Documentation").
		SetAccelerator("F1").
		SetBitmap(menuIcon("documentation")).
		OnClick(func(ctx *application.Context) {
			if app.wailsApp != nil {
				app.wailsApp.Event.Emit("menu:docs", nil)
			}
		})
	helpMenu.AddSeparator()
	helpMenu.Add("Open Log File").
		SetBitmap(menuIcon("log-file")).
		OnClick(func(ctx *application.Context) {
			if err := app.OpenLogFile(); err != nil {
				log.Printf("Failed to open log file: %v", err)
			}
		})
	helpMenu.Add("Open Log Directory").
		SetBitmap(menuIcon("log-directory")).
		OnClick(func(ctx *application.Context) {
			if err := app.OpenLogDirectory(); err != nil {
				log.Printf("Failed to open log directory: %v", err)
			}
		})

	return menu
}
