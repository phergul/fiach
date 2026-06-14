package reshade

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/phergul/fiach/internal/winversion"
)

func TestDiscoverCandidatesReturnsValidExecutablesAndWarnings(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	valid := filepath.Join(root, "bin", "Game.exe")
	invalid := filepath.Join(root, "tools", "Broken.exe")
	for _, path := range []string{valid, invalid} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("exe"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	result, err := DiscoverCandidates(root, DiscoveryOptions{
		InspectArchitecture: func(path string) (Architecture, error) {
			if strings.EqualFold(path, invalid) {
				return "", errors.New("invalid PE")
			}
			return ArchitectureX64, nil
		},
		ReadMetadata: func(string) (winversion.Metadata, error) {
			return winversion.Metadata{}, errors.New("no metadata")
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Candidates) != 1 || len(result.Warnings) != 1 {
		t.Fatalf("DiscoverCandidates() = %+v", result)
	}
	candidate := result.Candidates[0]
	if candidate.Architecture != ArchitectureX64 ||
		candidate.ExecutableRelativePath != filepath.Join("bin", "Game.exe") ||
		len(candidate.APIOptions) != 4 {
		t.Fatalf("candidate = %+v", candidate)
	}
	if result.Warnings[0].Path != filepath.Join("tools", "Broken.exe") {
		t.Fatalf("warning = %+v", result.Warnings[0])
	}
}

func TestDiscoverCandidatesReportsForeignAndAmbiguousProxies(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	executable := filepath.Join(root, "Game.exe")
	for _, path := range []string{
		executable,
		filepath.Join(root, "dxgi.dll"),
		filepath.Join(root, "d3d11.dll"),
		filepath.Join(root, "d3d12.dll"),
	} {
		if err := os.WriteFile(path, []byte("file"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	result, err := DiscoverCandidates(root, DiscoveryOptions{
		InspectArchitecture: func(string) (Architecture, error) {
			return ArchitectureX64, nil
		},
		ReadMetadata: func(path string) (winversion.Metadata, error) {
			if strings.EqualFold(filepath.Base(path), "d3d12.dll") {
				return winversion.Metadata{
					ProductName:      "OptiScaler",
					OriginalFilename: "OptiScaler.dll",
				}, nil
			}
			return winversion.Metadata{
				ProductName:      "ReShade",
				OriginalFilename: "ReShade64.dll",
				ProductVersion:   "6.7.3",
			}, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Candidates) != 1 {
		t.Fatalf("candidates = %+v", result.Candidates)
	}
	candidate := result.Candidates[0]
	if len(candidate.ProxyEvidence) != 3 || len(candidate.Conflicts) != 2 {
		t.Fatalf("candidate = %+v", candidate)
	}
}
