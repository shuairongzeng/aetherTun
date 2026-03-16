package main

import (
	"embed"
	"log"

	"github.com/shuairongzeng/aether/internal/tray"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := NewApp()
	trayController := tray.NewController(
		app.CurrentRuntimeStatus,
		func() { _ = app.StartCore() },
		func() { _ = app.StopCore() },
		app.ShowWindow,
		app.OpenLogDirectory,
		app.Quit,
	)
	go trayController.Run()

	err := wails.Run(&options.App{
		Title:         "Aether",
		Width:         1080,
		Height:        760,
		MinWidth:      920,
		MinHeight:     640,
		AssetServer:   &assetserver.Options{Assets: assets},
		OnStartup:     app.startup,
		OnBeforeClose: app.beforeClose,
		DisableResize: false,
		Frameless:     false,
		Fullscreen:    false,
		StartHidden:   false,
		Bind: []interface{}{
			app,
		},
	})
	if err != nil {
		log.Fatal(err)
	}
}
