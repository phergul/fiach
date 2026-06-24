package reshade

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"

	"github.com/phergul/fiach/internal/winversion"
)

type Target struct {
	Path        string
	Executables []string
}

type Result struct {
	Targets []Target
}

type candidateFolder struct {
	executables []string
	proxies     []string
	hasSupport  bool
}

var reshadeDLLNames = map[string]struct{}{
	"d3d9.dll":      {},
	"d3d10core.dll": {},
	"d3d11.dll":     {},
	"d3d12.dll":     {},
	"dxgi.dll":      {},
	"opengl32.dll":  {},
}

func ScanManaged(root string, managedChainedTargets []string) (result Result, err error) {
	return scan(root, managedChainedTargets, winversion.Read)
}

type metadataReader func(string) (winversion.Metadata, error)

func scan(root string, managedChainedTargets []string, readMetadata metadataReader) (result Result, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("scan ReShade runtime markers: %w", err)
		}
	}()

	root = filepath.Clean(root)
	folders := map[string]*candidateFolder{}
	managedTargets := make(map[string]bool, len(managedChainedTargets))
	for _, target := range managedChainedTargets {
		managedTargets[strings.ToLower(filepath.Clean(target))] = true
	}

	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return fmt.Errorf("walk %q: %w", path, walkErr)
		}
		if path == root {
			return nil
		}

		name := strings.ToLower(entry.Name())
		parent := filepath.Dir(path)
		folder := folders[parent]
		if folder == nil {
			folder = &candidateFolder{}
			folders[parent] = folder
		}

		if entry.IsDir() {
			if name == "reshade-shaders" {
				folder.hasSupport = true
			}
			return nil
		}

		switch {
		case strings.EqualFold(filepath.Ext(name), ".exe"):
			folder.executables = append(folder.executables, path)
		case name == "reshade.ini":
			folder.hasSupport = true
		default:
			_, knownDLL := reshadeDLLNames[name]
			isManagedChainedRuntime := (name == "reshade64.dll" || name == "reshade32.dll") &&
				managedTargets[strings.ToLower(filepath.Clean(parent))]
			if knownDLL || isManagedChainedRuntime {
				folder.proxies = append(folder.proxies, path)
			}
		}

		return nil
	})
	if err != nil {
		return Result{}, err
	}

	targets := make([]Target, 0, len(folders))
	for path, folder := range folders {
		if len(folder.executables) == 0 || !folder.hasSupport || !hasReShadeProxy(folder.proxies, readMetadata) {
			continue
		}
		sort.Strings(folder.executables)
		targets = append(targets, Target{
			Path:        path,
			Executables: append([]string(nil), folder.executables...),
		})
	}
	sort.Slice(targets, func(i int, j int) bool {
		return targets[i].Path < targets[j].Path
	})

	return Result{Targets: targets}, nil
}

func hasReShadeProxy(paths []string, readMetadata metadataReader) bool {
	for _, path := range paths {
		metadata, err := readMetadata(path)
		if err != nil {
			continue
		}
		originalFilename := strings.TrimSpace(metadata.OriginalFilename)
		if strings.EqualFold(strings.TrimSpace(metadata.ProductName), "ReShade") &&
			(strings.EqualFold(originalFilename, "ReShade64.dll") ||
				strings.EqualFold(originalFilename, "ReShade32.dll")) {
			return true
		}
	}
	return false
}
