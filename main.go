package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	wailshandler "github.com/yukihito-jokyu/DB-checker/internal/handler/wails"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := NewApp()
	appHandler := wailshandler.NewAppHandler()

	err := wails.Run(&options.App{
		Title:  "DB-checker",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		Bind: []interface{}{
			appHandler,
		},
	})
	if err != nil {
		println("Error:", err.Error())
	}
}
