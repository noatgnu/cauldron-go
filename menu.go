package main

import (
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"
)

func createApplicationMenu(app *App) *application.Menu {
	menu := application.NewMenu()

	fileMenu := menu.AddSubmenu("File")
	fileMenu.Add("Import Data").
		SetAccelerator("CmdOrCtrl+O").
		OnClick(func(ctx *application.Context) {
			if app.wailsApp != nil {
				app.wailsApp.Event.Emit("menu:import-data", nil)
			}
		})
	fileMenu.AddSeparator()
	fileMenu.Add("Settings").
		SetAccelerator("CmdOrCtrl+,").
		OnClick(func(ctx *application.Context) {
			if app.wailsApp != nil {
				app.wailsApp.Event.Emit("menu:settings", nil)
			}
		})
	fileMenu.AddSeparator()
	fileMenu.Add("Quit").
		SetAccelerator("CmdOrCtrl+Q").
		OnClick(func(ctx *application.Context) {
			app.HandleQuit()
		})

	viewMenu := menu.AddSubmenu("View")
	viewMenu.Add("Home").
		SetAccelerator("CmdOrCtrl+1").
		OnClick(func(ctx *application.Context) {
			if app.wailsApp != nil {
				app.wailsApp.Event.Emit("menu:view-home", nil)
			}
		})
	viewMenu.Add("Jobs").
		SetAccelerator("CmdOrCtrl+2").
		OnClick(func(ctx *application.Context) {
			if app.wailsApp != nil {
				app.wailsApp.Event.Emit("menu:view-jobs", nil)
			}
		})
	viewMenu.Add("Installed Plugins").
		SetAccelerator("CmdOrCtrl+3").
		OnClick(func(ctx *application.Context) {
			if app.wailsApp != nil {
				app.wailsApp.Event.Emit("menu:view-plugin-list", nil)
			}
		})

	windowMenu := menu.AddSubmenu("Window")
	windowMenu.Add("Minimize").
		SetAccelerator("CmdOrCtrl+M").
		OnClick(func(ctx *application.Context) {
			if app.mainWindow != nil {
				app.mainWindow.Minimise()
			}
		})
	windowMenu.Add("Zoom").
		OnClick(func(ctx *application.Context) {
			if app.mainWindow != nil {
				app.mainWindow.ToggleMaximise()
			}
		})

	helpMenu := menu.AddSubmenu("Help")
	helpMenu.Add("About Cauldron").
		OnClick(func(ctx *application.Context) {
			if app.wailsApp != nil {
				app.wailsApp.Event.Emit("menu:about", nil)
			}
		})
	helpMenu.Add("Documentation").
		SetAccelerator("F1").
		OnClick(func(ctx *application.Context) {
			if app.wailsApp != nil {
				app.wailsApp.Event.Emit("menu:docs", nil)
			}
		})
	helpMenu.AddSeparator()
	helpMenu.Add("Open Log File").
		OnClick(func(ctx *application.Context) {
			if err := app.OpenLogFile(); err != nil {
				log.Printf("Failed to open log file: %v", err)
			}
		})
	helpMenu.Add("Open Log Directory").
		OnClick(func(ctx *application.Context) {
			if err := app.OpenLogDirectory(); err != nil {
				log.Printf("Failed to open log directory: %v", err)
			}
		})

	return menu
}
