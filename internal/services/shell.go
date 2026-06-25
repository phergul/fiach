package services

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/phergul/fiach/internal/apperror"
	"github.com/phergul/fiach/internal/fileops"
	"github.com/wailsapp/wails/v3/pkg/application"
)

type ShellService struct {
	app **application.App
}

func NewShellService(app **application.App) *ShellService {
	return &ShellService{
		app: app,
	}
}

func (s *ShellService) OpenDirectory(ctx context.Context, path string) (err error) {
	defer func() {
		if err == nil || apperror.IsUserError(err) {
			return
		}
		err = shellUserError(fmt.Errorf("open directory in file manager: %w", err))
	}()

	cleanPath, err := fileops.CleanRequiredAbsPath("directory path", path)
	if err != nil {
		return err
	}

	info, err := os.Stat(cleanPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("directory %q does not exist", cleanPath)
		}
		return fmt.Errorf("stat directory %q: %w", cleanPath, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("path %q is not a directory", cleanPath)
	}

	if s.app == nil || *s.app == nil {
		return apperror.New("The application is not configured.")
	}

	return (*s.app).Env.OpenFileManager(cleanPath, false)
}
