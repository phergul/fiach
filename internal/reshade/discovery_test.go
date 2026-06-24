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
		len(candidate.APIOptions) != 5 {
		t.Fatalf("candidate = %+v", candidate)
	}
	openGL := candidate.APIOptions[len(candidate.APIOptions)-1]
	if openGL.RenderingAPI != RenderingAPIOpenGL ||
		len(openGL.Proxies) != 1 ||
		openGL.Proxies[0] != "opengl32.dll" {
		t.Fatalf("OpenGL API option = %+v", openGL)
	}
	if result.Warnings[0].Path != filepath.Join("tools", "Broken.exe") {
		t.Fatalf("warning = %+v", result.Warnings[0])
	}
}

func TestDiscoverCandidatesReportsOpenGLProxyEvidence(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	for _, path := range []string{
		filepath.Join(root, "Game.exe"),
		filepath.Join(root, "opengl32.dll"),
	} {
		if err := os.WriteFile(path, []byte("file"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	result, err := DiscoverCandidates(root, DiscoveryOptions{
		InspectArchitecture: func(string) (Architecture, error) {
			return ArchitectureX64, nil
		},
		ReadMetadata: func(string) (winversion.Metadata, error) {
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
	evidence := result.Candidates[0].ProxyEvidence
	if len(evidence) != 1 ||
		evidence[0].Filename != "opengl32.dll" ||
		!evidence[0].IsReShade ||
		evidence[0].RuntimeVersion != "6.7.3" {
		t.Fatalf("proxy evidence = %+v", evidence)
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

func TestDiscoverCandidatesAllowsManagedForeignProxy(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	executable := filepath.Join(root, "bin", "Game.exe")
	proxy := filepath.Join(root, "bin", "dxgi.dll")
	for _, path := range []string{executable, proxy} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("file"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	result, err := DiscoverCandidates(root, DiscoveryOptions{
		AllowedForeignProxyPaths: []string{proxy},
		InspectArchitecture: func(string) (Architecture, error) {
			return ArchitectureX64, nil
		},
		ReadMetadata: func(string) (winversion.Metadata, error) {
			return winversion.Metadata{
				ProductName:      "OptiScaler",
				OriginalFilename: "OptiScaler.dll",
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
	if len(candidate.Conflicts) != 0 ||
		len(candidate.ProxyEvidence) != 1 ||
		candidate.ProxyEvidence[0].Conflict != "" {
		t.Fatalf("candidate = %+v", candidate)
	}
}
