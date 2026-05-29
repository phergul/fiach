package reshade

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
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
	hasDLL      bool
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

func Scan(root string) (result Result, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("scan ReShade runtime markers: %w", err)
		}
	}()

	root = filepath.Clean(root)
	folders := map[string]*candidateFolder{}

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
			if knownDLL {
				folder.hasDLL = true
			}
		}

		return nil
	})
	if err != nil {
		return Result{}, err
	}

	targets := make([]Target, 0, len(folders))
	for path, folder := range folders {
		if len(folder.executables) == 0 || !folder.hasDLL || !folder.hasSupport {
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
