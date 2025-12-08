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

	analysisMenu := appMenu.AddSubmenu("Analysis")

	transformMenu := analysisMenu.AddSubmenu("Data Transformation")
	transformMenu.AddText("Imputation", nil, func(_ *menu.CallbackData) {
		runtime.EventsEmit(ctx, "menu:imputation")
	})
	transformMenu.AddText("Normalization", nil, func(_ *menu.CallbackData) {
		runtime.EventsEmit(ctx, "menu:normalization")
	})

	dimRedMenu := analysisMenu.AddSubmenu("Dimensionality Reduction")
	dimRedMenu.AddText("PCA", nil, func(_ *menu.CallbackData) {
		runtime.EventsEmit(ctx, "menu:pca")
	})
	dimRedMenu.AddText("PHATE", nil, func(_ *menu.CallbackData) {
		runtime.EventsEmit(ctx, "menu:phate")
	})
	dimRedMenu.AddText("Fuzzy Clustering", nil, func(_ *menu.CallbackData) {
		runtime.EventsEmit(ctx, "menu:fuzzy-clustering")
	})

	analysisMenu.AddSeparator()

	diffMenu := analysisMenu.AddSubmenu("Differential Analysis")
	diffMenu.AddText("Limma", nil, func(_ *menu.CallbackData) {
		runtime.EventsEmit(ctx, "menu:limma")
	})
	diffMenu.AddText("QFeatures + Limma", nil, func(_ *menu.CallbackData) {
		runtime.EventsEmit(ctx, "menu:qfeatures-limma")
	})
	diffMenu.AddText("AlphaStats", nil, func(_ *menu.CallbackData) {
		runtime.EventsEmit(ctx, "menu:alphastats")
	})

	analysisMenu.AddSeparator()
	analysisMenu.AddText("Estimation Plot", nil, func(_ *menu.CallbackData) {
		runtime.EventsEmit(ctx, "menu:estimation-plot")
	})
	analysisMenu.AddText("Correlation Matrix", nil, func(_ *menu.CallbackData) {
		runtime.EventsEmit(ctx, "menu:correlation-matrix")
	})

	utilitiesMenu := appMenu.AddSubmenu("Utilities")
	utilitiesMenu.AddText("UniProt Lookup", nil, func(_ *menu.CallbackData) {
		runtime.EventsEmit(ctx, "menu:uniprot")
	})
	utilitiesMenu.AddText("Coverage Map", nil, func(_ *menu.CallbackData) {
		runtime.EventsEmit(ctx, "menu:coverage-map")
	})
	utilitiesMenu.AddText("PTM Remapping", nil, func(_ *menu.CallbackData) {
		runtime.EventsEmit(ctx, "menu:ptm-remap")
	})
	utilitiesMenu.AddText("Peptide Library Check", nil, func(_ *menu.CallbackData) {
		runtime.EventsEmit(ctx, "menu:peptide-check")
	})
	utilitiesMenu.AddSeparator()
	utilitiesMenu.AddText("Format Conversion", nil, func(_ *menu.CallbackData) {
		runtime.EventsEmit(ctx, "menu:format-conversion")
	})

	viewMenu := appMenu.AddSubmenu("View")
	viewMenu.AddText("Home", keys.CmdOrCtrl("1"), func(_ *menu.CallbackData) {
		runtime.EventsEmit(ctx, "menu:view-home")
	})
	viewMenu.AddText("Jobs", keys.CmdOrCtrl("2"), func(_ *menu.CallbackData) {
		runtime.EventsEmit(ctx, "menu:view-jobs")
	})
	viewMenu.AddText("Custom Workflows", keys.CmdOrCtrl("3"), func(_ *menu.CallbackData) {
		runtime.EventsEmit(ctx, "menu:view-plugins")
	})
	viewMenu.AddText("Settings", keys.CmdOrCtrl("4"), func(_ *menu.CallbackData) {
		runtime.EventsEmit(ctx, "menu:view-settings")
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
