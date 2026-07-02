package updater

import (
	"context"
	"fmt"

	"github.com/phergul/fiach/internal/appmode"
	"github.com/phergul/fiach/internal/storage"
	"github.com/phergul/fiach/internal/version"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/updater"
	"github.com/wailsapp/wails/v3/pkg/updater/providers/github"
)

const githubRepository = "phergul/fiach"

func Init(app *application.App, store *storage.Store) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("init application updater: %w", err)
		}
	}()

	if appmode.IsDev() {
		return nil
	}

	themeID, err := store.GetThemeID(context.Background())
	if err != nil {
		return fmt.Errorf("read theme setting: %w", err)
	}

	provider, err := github.New(github.Config{
		Repository:    githubRepository,
		ChecksumAsset: "SHA256SUMS",
		AssetMatcher:  assetMatcher,
	})
	if err != nil {
		return err
	}

	return app.Updater.Init(updater.Config{
		CurrentVersion: version.Version,
		Providers:      []updater.Provider{provider},
		Window:         builtinWindow(themeID),
	})
}
