package services

import (
	"context"

	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/menu/keys"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type AppCallbacks interface {
	OpenLogFile() error
	OpenLogDirectory() error
	HandleQuit()
}

func BuildApplicationMenu(ctx context.Context, appCallbacks AppCallbacks) *menu.Menu {
	appMenu := menu.NewMenu()

	fileMenu := appMenu.AddSubmenu("File")
	fileMenu.AddText("Import Data", keys.CmdOrCtrl("o"), func(_ *menu.CallbackData) {
		runtime.EventsEmit(ctx, "menu:import-data")
	})
	fileMenu.AddSeparator()
	fileMenu.AddText("Settings", keys.CmdOrCtrl(","), func(_ *menu.CallbackData) {
		runtime.EventsEmit(ctx, "menu:settings")
	})
	fileMenu.AddSeparator()
	fileMenu.AddText("Quit", keys.CmdOrCtrl("q"), func(_ *menu.CallbackData) {
		appCallbacks.HandleQuit()
	})

	viewMenu := appMenu.AddSubmenu("View")
	viewMenu.AddText("Home", keys.CmdOrCtrl("1"), func(_ *menu.CallbackData) {
		runtime.EventsEmit(ctx, "menu:view-home")
	})
	viewMenu.AddText("Jobs", keys.CmdOrCtrl("2"), func(_ *menu.CallbackData) {
		runtime.EventsEmit(ctx, "menu:view-jobs")
	})
	viewMenu.AddText("Installed Plugins", keys.CmdOrCtrl("3"), func(_ *menu.CallbackData) {
		runtime.EventsEmit(ctx, "menu:view-plugin-list")
	})

	windowMenu := appMenu.AddSubmenu("Window")
	windowMenu.AddText("Minimize", keys.CmdOrCtrl("m"), func(_ *menu.CallbackData) {
		runtime.WindowMinimise(ctx)
	})
	windowMenu.AddText("Zoom", nil, func(_ *menu.CallbackData) {
		runtime.WindowToggleMaximise(ctx)
	})

	helpMenu := appMenu.AddSubmenu("Help")
	helpMenu.AddText("About Cauldron", nil, func(_ *menu.CallbackData) {
		runtime.EventsEmit(ctx, "menu:about")
	})
	helpMenu.AddText("Documentation", keys.Key("F1"), func(_ *menu.CallbackData) {
		runtime.EventsEmit(ctx, "menu:docs")
	})
	helpMenu.AddSeparator()
	helpMenu.AddText("Open Log File", nil, func(_ *menu.CallbackData) {
		if err := appCallbacks.OpenLogFile(); err != nil {
			runtime.LogErrorf(ctx, "Failed to open log file: %v", err)
		}
	})
	helpMenu.AddText("Open Log Directory", nil, func(_ *menu.CallbackData) {
		if err := appCallbacks.OpenLogDirectory(); err != nil {
			runtime.LogErrorf(ctx, "Failed to open log directory: %v", err)
		}
	})

	return appMenu
}
