package main

import (
	"context"
	"embed"
	"log/slog"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/yukihito-jokyu/DB-checker/internal/config"
	wailshandler "github.com/yukihito-jokyu/DB-checker/internal/handler/wails"
	applogger "github.com/yukihito-jokyu/DB-checker/internal/logger"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := NewApp()
	logger := applogger.New(slog.LevelInfo)
	configStore, err := config.NewDefaultStore()
	if err != nil {
		logger.Error(context.Background(), "config store initialization failed", err)
		return
	}
	appHandler := wailshandler.NewAppHandler(logger, configStore)

	err = wails.Run(&options.App{
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
		logger.Error(context.Background(), "wails run failed", err)
	}
}
