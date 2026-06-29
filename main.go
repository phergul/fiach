package main

import (
	"context"
	"embed"
	_ "embed"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/phergul/fiach/internal/appmode"
	"github.com/phergul/fiach/internal/devlog"
	"github.com/phergul/fiach/internal/diagnostics"
	"github.com/phergul/fiach/internal/gamesource"
	"github.com/phergul/fiach/internal/injection"
	"github.com/phergul/fiach/internal/services"
	"github.com/phergul/fiach/internal/services/dto"
	"github.com/phergul/fiach/internal/storage"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	diagnosticsManager, err := diagnostics.NewManager(diagnostics.Options{
		LogPath: filepath.Join(appmode.DataRoot(), "logs", diagnostics.DefaultLogFileName),
	})
	if err != nil {
		slog.Error("Failed to initialize diagnostics", "error", err)
		os.Exit(1)
	}
	logger := diagnosticsManager.Logger()
	devlog.SetLogger(logger)

	store, err := storage.Open(context.Background(), storage.Options{
		AppName: appmode.DataDirName(),
	})
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
	profileService := services.NewProfileService(store, logger)

	injectionCoordinator := injection.NewCoordinator(store)

	var app *application.App
	app = application.New(application.Options{
		Name:        "fiach",
		Description: "A general-purpose mod manager for any game",
		Services: []application.Service{
			application.NewService(services.NewModService(store, logger)),
			application.NewService(profileService),
			application.NewService(services.NewDeploymentReviewService(store, profileService, logger)),
			application.NewService(services.NewSettingsService(store, logger)),
			application.NewService(services.NewReshadeService(store, logger, injectionCoordinator)),
			application.NewService(services.NewOptiScalerService(store, logger, injectionCoordinator)),
			application.NewService(gamesService),

			application.NewService(services.NewDiagnosticsService(diagnosticsManager)),
			application.NewService(services.NewDevService(store.Path())),
			application.NewService(services.NewWindowService(&app)),
			application.NewService(services.NewShellService(&app)),
			application.NewService(appmode.NewRuntime(app)),
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

	mainWindow := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:  "main",
		Title: "Fiach",
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 50,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarDefault,
		},
		BackgroundColour: application.NewRGB(27, 38, 54),
		EnableFileDrop:   true,
		Width:            1920,
		Height:           1080,
		MinWidth:         1000,
		MinHeight:        800,
		URL:              "/",
	})
	mainWindow.OnWindowEvent(events.Common.WindowFilesDropped, func(event *application.WindowEvent) {
		files := event.Context().DroppedFiles()
		if len(files) == 0 {
			return
		}

		mainWindow.EmitEvent("files-dropped", map[string]any{
			"files": files,
		})
	})

	devlog.SetEmitter(func(entry devlog.Entry) {
		window, ok := app.Window.GetByName("dev-logs")
		if !ok {
			return
		}

		window.EmitEvent("dev:log-entry", dto.DevLogEntry{
			Timestamp: entry.Timestamp,
			Message:   entry.Message,
		})
	})
	devlog.Logf("dev mode enabled — data root: %s", appmode.DataRoot())
	devlog.Logf("using database: %s", store.Path())

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
