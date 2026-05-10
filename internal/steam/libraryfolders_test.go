package steam

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestParseLibraryFoldersReturnsDefaultAndAdditionalLibraries(t *testing.T) {
	t.Parallel()

	root := createSteamRoot(t)
	extraLibrary := filepath.Join(t.TempDir(), "SteamLibrary")
	writeLibraryFoldersVDF(t, root, `
"libraryfolders"
{
	"0"
	{
		"path"		"`+root+`"
	}
	"1"
	{
		"path"		"`+extraLibrary+`"
	}
}
`)

	got, err := ParseLibraryFolders(mustValidateSteamRoot(t, root))
	if err != nil {
		t.Fatalf("ParseLibraryFolders() error = %v", err)
	}

	want := []string{filepath.Clean(root), filepath.Clean(extraLibrary)}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ParseLibraryFolders() = %#v, want %#v", got, want)
	}
}

func TestParseLibraryFoldersDedupesAndIgnoresInvalidEntries(t *testing.T) {
	t.Parallel()

	root := createSteamRoot(t)
	extraLibrary := filepath.Join(t.TempDir(), "SteamLibrary")
	writeLibraryFoldersVDF(t, root, `
"libraryfolders"
{
	"0"
	{
		"path"		"`+root+`"
	}
	"1"
	{
		"path"		"`+extraLibrary+`"
	}
	"2"
	{
		"label"		"missing path"
	}
	"contentstatsid"		"123"
	"not-number"
	{
		"path"		"/ignored"
	}
	"3"
	{
		"path"		"`+extraLibrary+`"
	}
}
`)

	got, err := ParseLibraryFolders(mustValidateSteamRoot(t, root))
	if err != nil {
		t.Fatalf("ParseLibraryFolders() error = %v", err)
	}

	want := []string{filepath.Clean(root), filepath.Clean(extraLibrary)}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ParseLibraryFolders() = %#v, want %#v", got, want)
	}
}

func TestParseLibraryFoldersReturnsOnlyDefaultWhenVDFHasNoValidAdditionalPaths(t *testing.T) {
	t.Parallel()

	root := createSteamRoot(t)
	writeLibraryFoldersVDF(t, root, `
"libraryfolders"
{
	"0"
	{
		"label"		"missing path"
	}
}
`)

	got, err := ParseLibraryFolders(mustValidateSteamRoot(t, root))
	if err != nil {
		t.Fatalf("ParseLibraryFolders() error = %v", err)
	}

	want := []string{filepath.Clean(root)}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ParseLibraryFolders() = %#v, want %#v", got, want)
	}
}

func TestParseLibraryFoldersReturnsErrorsForInvalidInputs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		paths   func(t *testing.T) *SteamPaths
		wantErr string
	}{
		{
			name: "nil paths",
			paths: func(t *testing.T) *SteamPaths {
				t.Helper()
				return nil
			},
			wantErr: "Steam paths are not configured",
		},
		{
			name: "missing file",
			paths: func(t *testing.T) *SteamPaths {
				t.Helper()
				return &SteamPaths{Root: t.TempDir(), LibraryVDF: filepath.Join(t.TempDir(), "missing.vdf")}
			},
			wantErr: "open libraryfolders.vdf",
		},
		{
			name: "malformed vdf",
			paths: func(t *testing.T) *SteamPaths {
				t.Helper()
				root := createSteamRoot(t)
				writeLibraryFoldersVDF(t, root, `"libraryfolders"`)
				return mustValidateSteamRoot(t, root)
			},
			wantErr: "parse libraryfolders.vdf",
		},
		{
			name: "missing libraryfolders key",
			paths: func(t *testing.T) *SteamPaths {
				t.Helper()
				root := createSteamRoot(t)
				writeLibraryFoldersVDF(t, root, `"not_libraryfolders" {}`)
				return mustValidateSteamRoot(t, root)
			},
			wantErr: "missing libraryfolders key",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := ParseLibraryFolders(tt.paths(t))
			if err == nil {
				t.Fatal("ParseLibraryFolders() error = nil, want error")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("ParseLibraryFolders() error = %q, want substring %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestToStringMap(t *testing.T) {
	t.Parallel()

	got := toStringMap(map[any]any{
		"a": "b",
		1:   "ignored",
	})
	want := map[string]any{"a": "b"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("toStringMap() = %#v, want %#v", got, want)
	}
	if got := toStringMap([]string{"unsupported"}); got != nil {
		t.Fatalf("toStringMap() = %#v, want nil", got)
	}
}

func TestIsNumeric(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in   string
		want bool
	}{
		{in: "0", want: true},
		{in: "123", want: true},
		{in: "1a", want: false},
		{in: "-1", want: false},
		{in: "", want: false},
	}

	for _, tt := range tests {
		if got := isNumeric(tt.in); got != tt.want {
			t.Fatalf("isNumeric(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func writeLibraryFoldersVDF(t *testing.T, root string, content string) {
	t.Helper()

	writeFile(t, filepath.Join(root, "steamapps", "libraryfolders.vdf"), content)
}

func mustValidateSteamRoot(t *testing.T, root string) *SteamPaths {
	t.Helper()

	paths, err := ValidateSteamRoot(root)
	if err != nil {
		t.Fatalf("ValidateSteamRoot() error = %v", err)
	}

	return paths
}
