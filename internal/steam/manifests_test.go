package steam

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestParseAppManifestExtractsGameFields(t *testing.T) {
	t.Parallel()

	libraryPath := t.TempDir()
	manifestPath := writeAppManifest(t, libraryPath, "appmanifest_489830.acf", `
"AppState"
{
	"appid"		"489830"
	"name"		"The Elder Scrolls V: Skyrim Special Edition"
	"installdir"		"Skyrim Special Edition"
}
`)

	got, err := ParseAppManifest(manifestPath, libraryPath)
	if err != nil {
		t.Fatalf("ParseAppManifest() error = %v", err)
	}

	wantInstallPath := filepath.Join(libraryPath, "steamapps", "common", "Skyrim Special Edition")
	if got.AppID != "489830" {
		t.Fatalf("AppID = %q, want 489830", got.AppID)
	}
	if got.Name != "The Elder Scrolls V: Skyrim Special Edition" {
		t.Fatalf("Name = %q, want Skyrim name", got.Name)
	}
	if got.InstallDir != "Skyrim Special Edition" {
		t.Fatalf("InstallDir = %q, want install dir", got.InstallDir)
	}
	if got.LibraryPath != filepath.Clean(libraryPath) {
		t.Fatalf("LibraryPath = %q, want %q", got.LibraryPath, libraryPath)
	}
	if got.InstallPath != wantInstallPath {
		t.Fatalf("InstallPath = %q, want %q", got.InstallPath, wantInstallPath)
	}
	if got.ManifestPath != filepath.Clean(manifestPath) {
		t.Fatalf("ManifestPath = %q, want %q", got.ManifestPath, manifestPath)
	}
}

func TestParseAppManifestReturnsErrorsForInvalidInputs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		manifestPath func(t *testing.T) string
		libraryPath  func(t *testing.T) string
		wantErr      string
	}{
		{
			name: "empty manifest path",
			manifestPath: func(t *testing.T) string {
				t.Helper()
				return ""
			},
			libraryPath: func(t *testing.T) string {
				t.Helper()
				return t.TempDir()
			},
			wantErr: "manifest path is empty",
		},
		{
			name: "empty library path",
			manifestPath: func(t *testing.T) string {
				t.Helper()
				return writeAppManifest(t, t.TempDir(), "appmanifest_1.acf", validManifest("1", "Game", "Game"))
			},
			libraryPath: func(t *testing.T) string {
				t.Helper()
				return ""
			},
			wantErr: "library path is empty",
		},
		{
			name: "malformed vdf",
			manifestPath: func(t *testing.T) string {
				t.Helper()
				return writeAppManifest(t, t.TempDir(), "appmanifest_1.acf", `"AppState"`)
			},
			libraryPath: func(t *testing.T) string {
				t.Helper()
				return t.TempDir()
			},
			wantErr: "parse manifest VDF",
		},
		{
			name: "missing AppState",
			manifestPath: func(t *testing.T) string {
				t.Helper()
				return writeAppManifest(t, t.TempDir(), "appmanifest_1.acf", `"Other" {}`)
			},
			libraryPath: func(t *testing.T) string {
				t.Helper()
				return t.TempDir()
			},
			wantErr: "missing AppState section",
		},
		{
			name: "missing required field",
			manifestPath: func(t *testing.T) string {
				t.Helper()
				return writeAppManifest(t, t.TempDir(), "appmanifest_1.acf", `
"AppState"
{
	"appid"		"1"
	"name"		"Game"
}
`)
			},
			libraryPath: func(t *testing.T) string {
				t.Helper()
				return t.TempDir()
			},
			wantErr: "manifest is missing appid, name, or installdir",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := ParseAppManifest(tt.manifestPath(t), tt.libraryPath(t))
			if err == nil {
				t.Fatal("ParseAppManifest() error = nil, want error")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("ParseAppManifest() error = %q, want substring %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestScanInstalledGamesReturnsGamesFromMultipleLibraries(t *testing.T) {
	t.Parallel()

	libraryOne := t.TempDir()
	libraryTwo := t.TempDir()
	writeAppManifest(t, libraryOne, "appmanifest_1.acf", validManifest("1", "Game One", "GameOne"))
	writeAppManifest(t, libraryTwo, "appmanifest_2.acf", validManifest("2", "Game Two", "GameTwo"))

	got, err := ScanInstalledGames([]string{libraryOne, libraryTwo})
	if err != nil {
		t.Fatalf("ScanInstalledGames() error = %v", err)
	}

	want := []Game{
		{
			AppID:        "1",
			Name:         "Game One",
			InstallDir:   "GameOne",
			LibraryPath:  filepath.Clean(libraryOne),
			InstallPath:  filepath.Join(libraryOne, "steamapps", "common", "GameOne"),
			ManifestPath: filepath.Join(libraryOne, "steamapps", "appmanifest_1.acf"),
		},
		{
			AppID:        "2",
			Name:         "Game Two",
			InstallDir:   "GameTwo",
			LibraryPath:  filepath.Clean(libraryTwo),
			InstallPath:  filepath.Join(libraryTwo, "steamapps", "common", "GameTwo"),
			ManifestPath: filepath.Join(libraryTwo, "steamapps", "appmanifest_2.acf"),
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ScanInstalledGames() = %#v, want %#v", got, want)
	}
}

func TestScanInstalledGamesIgnoresInvalidAndNonMatchingFiles(t *testing.T) {
	t.Parallel()

	libraryPath := t.TempDir()
	writeAppManifest(t, libraryPath, "appmanifest_1.acf", validManifest("1", "Game One", "GameOne"))
	writeAppManifest(t, libraryPath, "appmanifest_2.acf", `"AppState"`)
	writeAppManifest(t, libraryPath, "appmanifest_3.acf", `
"AppState"
{
	"appid"		"3"
	"name"		"Incomplete"
}
`)
	writeSteamAppsFile(t, libraryPath, "notes.txt", validManifest("4", "Ignored", "Ignored"))
	writeSteamAppsFile(t, libraryPath, "appmanifest_5.tmp", validManifest("5", "Ignored", "Ignored"))

	got, err := ScanInstalledGames([]string{libraryPath})
	if err != nil {
		t.Fatalf("ScanInstalledGames() error = %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("ScanInstalledGames() returned %d games, want 1: %#v", len(got), got)
	}
	if got[0].AppID != "1" {
		t.Fatalf("AppID = %q, want only valid game 1", got[0].AppID)
	}
}

func TestScanInstalledGamesReturnsEmptyForLibrariesWithoutManifests(t *testing.T) {
	t.Parallel()

	libraryPath := t.TempDir()
	mkdirAll(t, filepath.Join(libraryPath, "steamapps"))

	got, err := ScanInstalledGames([]string{libraryPath})
	if err != nil {
		t.Fatalf("ScanInstalledGames() error = %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("ScanInstalledGames() returned %d games, want 0", len(got))
	}
}

func writeAppManifest(t *testing.T, libraryPath string, name string, content string) string {
	t.Helper()

	return writeSteamAppsFile(t, libraryPath, name, content)
}

func writeSteamAppsFile(t *testing.T, libraryPath string, name string, content string) string {
	t.Helper()

	steamAppsPath := filepath.Join(libraryPath, "steamapps")
	mkdirAll(t, steamAppsPath)

	path := filepath.Join(steamAppsPath, name)
	writeFile(t, path, content)
	return path
}

func validManifest(appID string, name string, installDir string) string {
	return `
"AppState"
{
	"appid"		"` + appID + `"
	"name"		"` + name + `"
	"installdir"		"` + installDir + `"
}
`
}
