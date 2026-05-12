package main

import (
	"context"
	"embed"
	_ "embed"
	"log"

	"github.com/phergul/mod-manager/internal/services"
	"github.com/phergul/mod-manager/internal/storage"
	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	store, err := storage.Open(context.Background(), storage.Options{})
	if err != nil {
		log.Fatalf("failed to open storage: %v", err)
	}

	if err := store.MigrateUp(); err != nil {
		if closeErr := store.Close(); closeErr != nil {
			log.Printf("failed to close storage after migration error: %v", closeErr)
		}
		log.Fatalf("failed to migrate storage: %v", err)
	}

	app := application.New(application.Options{
		Name:        "mod-manager",
		Description: "General Mod Manager",
		Services: []application.Service{
			application.NewService(services.NewModService(store)),
			application.NewService(services.NewProfileService(store)),
			application.NewService(services.NewSteamService(store)),
		},
		OnShutdown: func() {
			if err := store.Close(); err != nil {
				log.Printf("failed to close storage: %v", err)
			}
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title: "Manager",
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 50,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarDefault,
		},
		BackgroundColour: application.NewRGB(27, 38, 54),
		Width:            1920,
		Height:           1080,
		URL:              "/",
	})

	err = app.Run()

	if err != nil {
		if closeErr := store.Close(); closeErr != nil {
			log.Printf("failed to close storage after app error: %v", closeErr)
		}
		log.Fatal(err)
	}
}
