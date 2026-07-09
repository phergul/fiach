package steam

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/andygrunwald/vdf"
)

type Game struct {
	AppID        string
	Name         string
	InstallDir   string
	LibraryPath  string
	InstallPath  string
	ManifestPath string
}

func ScanInstalledGames(libraryPaths []string) (games []Game, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("scan installed Steam games: %w", err)
		}
	}()

	games = make([]Game, 0)

	for _, libraryPath := range uniqueCleanPaths(libraryPaths) {
		steamAppsPath := filepath.Join(libraryPath, "steamapps")
		entries, err := os.ReadDir(steamAppsPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}

			return nil, fmt.Errorf("read steamapps directory %q: %w", steamAppsPath, err)
		}

		for _, entry := range entries {
			if entry.IsDir() || !isAppManifestFile(entry.Name()) {
				continue
			}

			manifestPath := filepath.Join(steamAppsPath, entry.Name())
			game, err := ParseAppManifest(manifestPath, libraryPath)
			if err != nil || game == nil {
				continue
			}

			installed, err := hasInstalledGameDirectory(*game)
			if err != nil {
				return nil, err
			}
			if !installed {
				continue
			}

			games = append(games, *game)
		}
	}

	return games, nil
}

func hasInstalledGameDirectory(game Game) (bool, error) {
	info, err := os.Stat(game.InstallPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}

		return false, fmt.Errorf("stat install directory %q: %w", game.InstallPath, err)
	}

	return info.IsDir(), nil
}

func ParseAppManifest(manifestPath string, libraryPath string) (game *Game, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("parse Steam app manifest %q: %w", manifestPath, err)
		}
	}()

	manifestPath = strings.TrimSpace(manifestPath)
	if manifestPath == "" {
		return nil, fmt.Errorf("manifest path is empty")
	}

	libraryPath = strings.TrimSpace(libraryPath)
	if libraryPath == "" {
		return nil, fmt.Errorf("library path is empty")
	}

	file, err := os.Open(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("open manifest: %w", err)
	}
	defer file.Close()

	parser := vdf.NewParser(file)
	data, err := parser.Parse()
	if err != nil {
		return nil, fmt.Errorf("parse manifest VDF: %w", err)
	}

	root := toStringMap(data)
	appState := toStringMap(root["AppState"])
	if appState == nil {
		return nil, fmt.Errorf("missing AppState section")
	}

	appID := strings.TrimSpace(stringValue(appState, "appid"))
	name := strings.TrimSpace(stringValue(appState, "name"))
	installDir := strings.TrimSpace(stringValue(appState, "installdir"))
	if appID == "" || name == "" || installDir == "" {
		return nil, fmt.Errorf("manifest is missing appid, name, or installdir")
	}

	libraryPath = filepath.Clean(libraryPath)
	manifestPath = filepath.Clean(manifestPath)

	return &Game{
		AppID:        appID,
		Name:         name,
		InstallDir:   installDir,
		LibraryPath:  libraryPath,
		InstallPath:  filepath.Join(libraryPath, "steamapps", "common", installDir),
		ManifestPath: manifestPath,
	}, nil
}

func isAppManifestFile(name string) bool {
	return strings.HasPrefix(name, "appmanifest_") && strings.HasSuffix(name, ".acf")
}

func stringValue(m map[string]any, key string) string {
	if m == nil {
		return ""
	}

	value, ok := m[key].(string)
	if !ok {
		return ""
	}

	return value
}
