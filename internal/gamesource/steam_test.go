package gamesource

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/phergul/mod-manager/internal/steam"
	"github.com/phergul/mod-manager/internal/storage"
)

func TestSteamSourceLocatesManualSteamPath(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	steamRoot := createSteamRoot(t)
	if err := store.SetSetting(context.Background(), SteamInstallPathSettingKey, steamRoot); err != nil {
		t.Fatalf("SetSetting() error = %v", err)
	}

	source := NewSteamSource(store)
	got, err := source.locateSteamInstallation(context.Background())
	if err != nil {
		t.Fatalf("LocateSteamInstallation() error = %v", err)
	}

	if got.Root != filepath.Clean(steamRoot) {
		t.Fatalf("Root = %q, want %q", got.Root, steamRoot)
	}
}

func TestSteamSourceReturnsClearNotFoundError(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	if err := store.SetSetting(context.Background(), SteamInstallPathSettingKey, filepath.Join(t.TempDir(), "missing")); err != nil {
		t.Fatalf("SetSetting() error = %v", err)
	}

	source := NewSteamSource(store)
	_, err := source.locateSteamInstallation(context.Background())
	if !errors.Is(err, steam.ErrSteamNotFound) {
		t.Fatalf("LocateSteamInstallation() error = %v, want ErrSteamNotFound", err)
	}
	if !strings.Contains(err.Error(), "Steam installation could not be found") {
		t.Fatalf("LocateSteamInstallation() error = %q, want clear message", err.Error())
	}
}

func TestSteamSourceGetsSteamLibrariesFromManualPath(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	steamRoot := createSteamRoot(t)
	extraLibrary := filepath.Join(t.TempDir(), "SteamLibrary")
	writeLibraryFoldersVDF(t, steamRoot, `
"libraryfolders"
{
	"0"
	{
		"path"		"`+steamRoot+`"
	}
	"1"
	{
		"path"		"`+extraLibrary+`"
	}
}
`)
	if err := store.SetSetting(context.Background(), SteamInstallPathSettingKey, steamRoot); err != nil {
		t.Fatalf("SetSetting() error = %v", err)
	}

	source := NewSteamSource(store)
	got, err := source.getSteamLibraries(context.Background())
	if err != nil {
		t.Fatalf("GetSteamLibraries() error = %v", err)
	}

	want := []string{filepath.Clean(steamRoot), filepath.Clean(extraLibrary)}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("GetSteamLibraries() = %#v, want %#v", got, want)
	}
}

func TestSteamSourceReturnsLibraryFolderParseError(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	steamRoot := createSteamRoot(t)
	writeLibraryFoldersVDF(t, steamRoot, `"libraryfolders"`)
	if err := store.SetSetting(context.Background(), SteamInstallPathSettingKey, steamRoot); err != nil {
		t.Fatalf("SetSetting() error = %v", err)
	}

	source := NewSteamSource(store)
	_, err := source.getSteamLibraries(context.Background())
	if err == nil {
		t.Fatal("GetSteamLibraries() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "get Steam libraries") {
		t.Fatalf("GetSteamLibraries() error = %q, want library context", err.Error())
	}
	if !strings.Contains(err.Error(), "parse libraryfolders.vdf") {
		t.Fatalf("GetSteamLibraries() error = %q, want parse context", err.Error())
	}
}

func TestSteamSourceGetsInstalledSteamGames(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	steamRoot := createSteamRoot(t)
	extraLibrary := filepath.Join(t.TempDir(), "SteamLibrary")
	writeLibraryFoldersVDF(t, steamRoot, `
"libraryfolders"
{
	"0"
	{
		"path"		"`+steamRoot+`"
	}
	"1"
	{
		"path"		"`+extraLibrary+`"
	}
}
`)
	writeAppManifest(t, steamRoot, "appmanifest_1.acf", validManifest("1", "Game One", "GameOne"))
	writeAppManifest(t, extraLibrary, "appmanifest_2.acf", validManifest("2", "Game Two", "GameTwo"))
	if err := store.SetSetting(context.Background(), SteamInstallPathSettingKey, steamRoot); err != nil {
		t.Fatalf("SetSetting() error = %v", err)
	}

	source := NewSteamSource(store)
	got, err := source.getInstalledSteamGames(context.Background())
	if err != nil {
		t.Fatalf("GetInstalledSteamGames() error = %v", err)
	}

	want := []steam.Game{
		{
			AppID:        "1",
			Name:         "Game One",
			InstallDir:   "GameOne",
			LibraryPath:  filepath.Clean(steamRoot),
			InstallPath:  filepath.Join(steamRoot, "steamapps", "common", "GameOne"),
			ManifestPath: filepath.Join(steamRoot, "steamapps", "appmanifest_1.acf"),
		},
		{
			AppID:        "2",
			Name:         "Game Two",
			InstallDir:   "GameTwo",
			LibraryPath:  filepath.Clean(extraLibrary),
			InstallPath:  filepath.Join(extraLibrary, "steamapps", "common", "GameTwo"),
			ManifestPath: filepath.Join(extraLibrary, "steamapps", "appmanifest_2.acf"),
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("GetInstalledSteamGames() = %#v, want %#v", got, want)
	}
}

func TestSteamSourceReturnsInstalledGamesLibraryFolderError(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	steamRoot := createSteamRoot(t)
	writeLibraryFoldersVDF(t, steamRoot, `"libraryfolders"`)
	if err := store.SetSetting(context.Background(), SteamInstallPathSettingKey, steamRoot); err != nil {
		t.Fatalf("SetSetting() error = %v", err)
	}

	source := NewSteamSource(store)
	_, err := source.getInstalledSteamGames(context.Background())
	if err == nil {
		t.Fatal("GetInstalledSteamGames() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "get installed Steam games") {
		t.Fatalf("GetInstalledSteamGames() error = %q, want installed games context", err.Error())
	}
	if !strings.Contains(err.Error(), "get Steam libraries") {
		t.Fatalf("GetInstalledSteamGames() error = %q, want library context", err.Error())
	}
}

func TestSteamArtworkMiddlewareServesExistingArtwork(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	steamRoot := createSteamRoot(t)
	imageDir := filepath.Join(steamRoot, "appcache", "librarycache", "400")
	mkdirAll(t, imageDir)
	writeFile(t, filepath.Join(imageDir, "library_600x900.png"), "\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR")

	if err := store.SetSetting(context.Background(), SteamInstallPathSettingKey, steamRoot); err != nil {
		t.Fatalf("SetSetting() error = %v", err)
	}

	handler := newSteamArtworkTestHandler(NewSteamSource(store))
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/artwork/steam/400/banner", nil)

	handler.ServeHTTP(response, request)

	result := response.Result()
	defer result.Body.Close()

	if result.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%q", result.StatusCode, http.StatusOK, response.Body.String())
	}
	if got := result.Header.Get("Content-Type"); got != "image/png" {
		t.Fatalf("Content-Type = %q, want image/png", got)
	}
	if got := result.Header.Get("Cache-Control"); got != steamArtworkCache {
		t.Fatalf("Cache-Control = %q, want %q", got, steamArtworkCache)
	}
	if body := response.Body.String(); !strings.HasPrefix(body, "\x89PNG") {
		t.Fatalf("body = %q, want PNG content", body)
	}
}

func TestSteamArtworkMiddlewareSupportsHead(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	steamRoot := createSteamRoot(t)
	imageDir := filepath.Join(steamRoot, "appcache", "librarycache", "400")
	mkdirAll(t, imageDir)
	writeFile(t, filepath.Join(imageDir, "library_600x900.jpg"), "\xff\xd8\xff\xe0\x00\x10JFIF")

	if err := store.SetSetting(context.Background(), SteamInstallPathSettingKey, steamRoot); err != nil {
		t.Fatalf("SetSetting() error = %v", err)
	}

	handler := newSteamArtworkTestHandler(NewSteamSource(store))
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodHead, "/artwork/steam/400/banner", nil)

	handler.ServeHTTP(response, request)

	result := response.Result()
	defer result.Body.Close()

	if result.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%q", result.StatusCode, http.StatusOK, response.Body.String())
	}
	if got := result.Header.Get("Content-Type"); got != "image/jpeg" {
		t.Fatalf("Content-Type = %q, want image/jpeg", got)
	}
	body, err := io.ReadAll(result.Body)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if len(body) != 0 {
		t.Fatalf("body length = %d, want 0", len(body))
	}
}

func TestSteamArtworkMiddlewareReturnsNotFoundForMissingArtwork(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	steamRoot := createSteamRoot(t)
	if err := store.SetSetting(context.Background(), SteamInstallPathSettingKey, steamRoot); err != nil {
		t.Fatalf("SetSetting() error = %v", err)
	}

	handler := newSteamArtworkTestHandler(NewSteamSource(store))
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/artwork/steam/400/banner", nil)

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNotFound)
	}
}

func TestSteamArtworkMiddlewareReturnsNotFoundForInvalidRoutes(t *testing.T) {
	t.Parallel()

	handler := newSteamArtworkTestHandler(nil)
	tests := []string{
		"/artwork/steam",
		"/artwork/steam/",
		"/artwork/steam/abc/banner",
		"/artwork/steam/400",
		"/artwork/steam/400/banner/extra",
		"/artwork/steam/400/icon",
		"/artwork/steam/40a/banner",
	}

	for _, path := range tests {
		t.Run(path, func(t *testing.T) {
			t.Parallel()

			response := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, path, nil)

			handler.ServeHTTP(response, request)

			if response.Code != http.StatusNotFound {
				t.Fatalf("status = %d, want %d", response.Code, http.StatusNotFound)
			}
		})
	}
}

func TestSteamArtworkMiddlewareRejectsUnsupportedMethods(t *testing.T) {
	t.Parallel()

	handler := newSteamArtworkTestHandler(nil)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/artwork/steam/400/banner", nil)

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusMethodNotAllowed)
	}
	if got := response.Header().Get("Allow"); got != "GET, HEAD" {
		t.Fatalf("Allow = %q, want GET, HEAD", got)
	}
}

func TestSteamArtworkMiddlewarePassesThroughUnrelatedRoutes(t *testing.T) {
	t.Parallel()

	next := http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		rw.WriteHeader(http.StatusTeapot)
	})
	handler := NewSteamArtworkMiddleware(nil)(next)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/assets/app.css", nil)

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusTeapot {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusTeapot)
	}
}

func TestSteamArtworkMiddlewareCachesSuccessfulArtworkRoot(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	steamRoot := createSteamRoot(t)
	imageDir := filepath.Join(steamRoot, "appcache", "librarycache", "400")
	mkdirAll(t, imageDir)
	writeFile(t, filepath.Join(imageDir, "library_600x900.png"), "\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR")

	if err := store.SetSetting(context.Background(), SteamInstallPathSettingKey, steamRoot); err != nil {
		t.Fatalf("SetSetting() error = %v", err)
	}

	handler := newSteamArtworkTestHandler(NewSteamSource(store))
	firstResponse := httptest.NewRecorder()
	handler.ServeHTTP(firstResponse, httptest.NewRequest(http.MethodGet, "/artwork/steam/400/banner", nil))
	if firstResponse.Code != http.StatusOK {
		t.Fatalf("first status = %d, want %d; body=%q", firstResponse.Code, http.StatusOK, firstResponse.Body.String())
	}

	if err := store.SetSetting(context.Background(), SteamInstallPathSettingKey, filepath.Join(t.TempDir(), "missing")); err != nil {
		t.Fatalf("SetSetting() error = %v", err)
	}

	secondResponse := httptest.NewRecorder()
	handler.ServeHTTP(secondResponse, httptest.NewRequest(http.MethodGet, "/artwork/steam/400/banner", nil))
	if secondResponse.Code != http.StatusOK {
		t.Fatalf("second status = %d, want %d; body=%q", secondResponse.Code, http.StatusOK, secondResponse.Body.String())
	}
}

func TestSteamArtworkMiddlewareReturnsServerErrorForSteamLookupFailure(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	if err := store.SetSetting(context.Background(), SteamInstallPathSettingKey, filepath.Join(t.TempDir(), "missing")); err != nil {
		t.Fatalf("SetSetting() error = %v", err)
	}

	handler := newSteamArtworkTestHandler(NewSteamSource(store))
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/artwork/steam/400/banner", nil)

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusInternalServerError)
	}
}

func newSteamArtworkTestHandler(source *SteamSource) http.Handler {
	next := http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		rw.WriteHeader(http.StatusNotFound)
	})

	return NewSteamArtworkMiddleware(source)(next)
}

func openMigratedStore(t *testing.T) *storage.Store {
	t.Helper()

	store, err := storage.Open(context.Background(), storage.Options{DataDir: t.TempDir()})
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	if err := store.MigrateUp(); err != nil {
		t.Fatalf("MigrateUp() error = %v", err)
	}

	return store
}

func closeStore(t *testing.T, store *storage.Store) {
	t.Helper()

	if err := store.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

func createSteamRoot(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	mkdirAll(t, filepath.Join(root, "steamapps"))
	mkdirAll(t, filepath.Join(root, "userdata"))
	writeFile(t, filepath.Join(root, "steamapps", "libraryfolders.vdf"))

	return root
}

func mkdirAll(t *testing.T, path string) {
	t.Helper()

	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", path, err)
	}
}

func writeFile(t *testing.T, path string, content ...string) {
	t.Helper()

	fileContent := "x"
	if len(content) > 0 {
		fileContent = content[0]
	}

	if err := os.WriteFile(path, []byte(fileContent), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}

func writeLibraryFoldersVDF(t *testing.T, root string, content string) {
	t.Helper()

	writeFile(t, filepath.Join(root, "steamapps", "libraryfolders.vdf"), content)
}

func writeAppManifest(t *testing.T, libraryPath string, name string, content string) string {
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
