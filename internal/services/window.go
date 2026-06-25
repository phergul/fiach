package services

import (
	"context"
	"fmt"

	"github.com/phergul/fiach/internal/apperror"
	"github.com/wailsapp/wails/v3/pkg/application"
)

const logsWindowName = "logs"
const devLogsWindowName = "dev-logs"

type WindowService struct {
	app **application.App
}

func NewWindowService(app **application.App) *WindowService {
	return &WindowService{
		app: app,
	}
}

func (s *WindowService) OpenLogsWindow(ctx context.Context) (err error) {
	defer func() {
		if err == nil || apperror.IsUserError(err) {
			return
		}
		err = shellUserError(fmt.Errorf("open logs window: %w", err))
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if s.app == nil || *s.app == nil {
		return apperror.New("The application is not configured.")
	}
	app := *s.app

	if window, ok := app.Window.GetByName(logsWindowName); ok {
		window.Show()
		window.Focus()
		return nil
	}

	window := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:             logsWindowName,
		Title:            "Logs",
		URL:              "/?window=logs",
		Width:            960,
		Height:           720,
		MinWidth:         820,
		MinHeight:        480,
		BackgroundColour: application.NewRGB(37, 36, 34),
	})
	window.Show()
	window.Focus()

	return nil
}

func (s *WindowService) OpenDevLogsWindow(ctx context.Context) (err error) {
	defer func() {
		if err == nil || apperror.IsUserError(err) {
			return
		}
		err = shellUserError(fmt.Errorf("open dev logs window: %w", err))
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if s.app == nil || *s.app == nil {
		return apperror.New("The application is not configured.")
	}
	app := *s.app

	if window, ok := app.Window.GetByName(devLogsWindowName); ok {
		window.Show()
		window.Focus()
		return nil
	}

	window := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:             devLogsWindowName,
		Title:            "Dev Logs",
		URL:              "/?window=dev-logs",
		Width:            960,
		Height:           720,
		MinWidth:         820,
		MinHeight:        480,
		BackgroundColour: application.NewRGB(37, 36, 34),
	})
	window.Show()
	window.Focus()

	return nil
}
