package main

import (
	"context"
	"embed"
	_ "embed"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/phergul/fiach/internal/diagnostics"
	"github.com/phergul/fiach/internal/gamesource"
	"github.com/phergul/fiach/internal/services"
	"github.com/phergul/fiach/internal/services/dto"
	"github.com/phergul/fiach/internal/storage"
	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	diagnosticsManager, err := diagnostics.NewManager(diagnostics.Options{
		LogPath: filepath.Join(application.Path(application.PathDataHome), "fiach", "logs", diagnostics.DefaultLogFileName),
	})
	if err != nil {
		slog.Error("Failed to initialize diagnostics", "error", err)
		os.Exit(1)
	}
	logger := diagnosticsManager.Logger()

	store, err := storage.Open(context.Background(), storage.Options{})
	if err != nil {
		logger.Error("Failed to open storage", diagnostics.ErrorAttr(err))
		_ = diagnosticsManager.Close()
		os.Exit(1)
	}

	if err := store.MigrateUp(); err != nil {
		closeStoreWithLog(store, logger, "Failed to close storage after migration error")
		logger.Error("Failed to migrate storage", diagnostics.ErrorAttr(err))
		_ = diagnosticsManager.Close()
		os.Exit(1)
	}

	steamSource := gamesource.NewSteamSource(store)
	gamesService := services.NewGamesService(store, logger, steamSource)

	var app *application.App
	app = application.New(application.Options{
		Name:        "fiach",
		Description: "A general-purpose mod manager for any game",
		Services: []application.Service{
			application.NewService(services.NewModService(store, logger)),
			application.NewService(services.NewProfileService(store, logger)),
			application.NewService(services.NewSettingsService(store, logger)),
			application.NewService(services.NewReshadeService(store, logger)),
			application.NewService(gamesService),

			application.NewService(services.NewDiagnosticsService(diagnosticsManager)),
			application.NewService(services.NewWindowService(&app)),
		},
		OnShutdown: func() {
			closeStoreWithLog(store, logger, "Failed to close storage")
			if err := diagnosticsManager.Close(); err != nil {
				slog.Error("Failed to close diagnostics", "error", err)
			}
		},
		Assets: application.AssetOptions{
			Handler:    application.AssetFileServerFS(assets),
			Middleware: gamesource.NewSteamArtworkMiddleware(steamSource),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	diagnosticLogEntries, unsubscribeDiagnosticLogEntries := diagnosticsManager.Subscribe()
	defer unsubscribeDiagnosticLogEntries()
	go emitDiagnosticLogEntries(app, diagnosticLogEntries)

	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:  "main",
		Title: "Fiach",
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 50,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarDefault,
		},
		BackgroundColour: application.NewRGB(27, 38, 54),
		Width:            1920,
		Height:           1080,
		MinWidth:         1000,
		MinHeight:        800,
		URL:              "/",
	})

	err = app.Run()

	if err != nil {
		closeStoreWithLog(store, logger, "Failed to close storage after app error")
		logger.Error("Application failed", diagnostics.ErrorAttr(err))
		_ = diagnosticsManager.Close()
		os.Exit(1)
	}
}

func closeStoreWithLog(store *storage.Store, logger *slog.Logger, message string) {
	if err := store.Close(); err != nil {
		logger.Error(message, diagnostics.ErrorAttr(err))
	}
}

func emitDiagnosticLogEntries(app *application.App, entries <-chan diagnostics.LogEntry) {
	for entry := range entries {
		window, ok := app.Window.GetByName("logs")
		if !ok {
			continue
		}

		window.EmitEvent("diagnostics:log-entry", dto.DiagnosticLogEntry{
			Timestamp: entry.Timestamp,
			Level:     entry.Level,
			Operation: entry.Operation,
			Message:   entry.Message,
			Details:   entry.Details,
		})
	}
}
