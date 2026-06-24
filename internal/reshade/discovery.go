package reshade

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/phergul/fiach/internal/winversion"
)

var supportedLocalProxies = []string{
	"d3d9.dll",
	"d3d10.dll",
	"d3d10core.dll",
	"d3d11.dll",
	"d3d12.dll",
	"dxgi.dll",
	"opengl32.dll",
}

type DiscoveryOptions struct {
	InspectArchitecture      func(string) (Architecture, error)
	ReadMetadata             func(string) (winversion.Metadata, error)
	AllowedForeignProxyPaths []string
}

func DiscoverCandidates(gameRoot string, options DiscoveryOptions) (result DiscoveryResult, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("discover managed ReShade candidates: %w", err)
		}
	}()
	if options.InspectArchitecture == nil {
		options.InspectArchitecture = inspectPEArchitecture
	}
	if options.ReadMetadata == nil {
		options.ReadMetadata = winversion.Read
	}
	options.AllowedForeignProxyPaths = normalizeDiscoveryAllowedProxyPaths(options.AllowedForeignProxyPaths)
	gameRoot, err = filepath.Abs(gameRoot)
	if err != nil {
		return DiscoveryResult{}, err
	}
	result = DiscoveryResult{
		Candidates: []Candidate{},
		Warnings:   []DiscoveryWarning{},
	}
	err = filepath.WalkDir(gameRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			if path == gameRoot {
				return walkErr
			}
			result.Warnings = append(result.Warnings, discoveryWarning(gameRoot, path, walkErr))
			if entry != nil && entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".exe") {
			return nil
		}
		architecture, inspectErr := options.InspectArchitecture(path)
		if inspectErr != nil {
			result.Warnings = append(result.Warnings, discoveryWarning(gameRoot, path, inspectErr))
			return nil
		}
		executableRelativePath, relativeErr := filepath.Rel(gameRoot, path)
		if relativeErr != nil {
			return relativeErr
		}
		targetPath := filepath.Dir(path)
		targetRelativePath, relativeErr := filepath.Rel(gameRoot, targetPath)
		if relativeErr != nil {
			return relativeErr
		}
		evidence, conflicts := inspectProxyEvidence(targetPath, options)
		result.Candidates = append(result.Candidates, Candidate{
			TargetRelativePath:     targetRelativePath,
			ExecutableRelativePath: executableRelativePath,
			Architecture:           architecture,
			APIOptions:             localAPIOptions(),
			ProxyEvidence:          evidence,
			Conflicts:              conflicts,
		})
		return nil
	})
	if err != nil {
		return DiscoveryResult{}, err
	}
	sort.Slice(result.Candidates, func(i int, j int) bool {
		return strings.ToLower(result.Candidates[i].ExecutableRelativePath) <
			strings.ToLower(result.Candidates[j].ExecutableRelativePath)
	})
	sort.Slice(result.Warnings, func(i int, j int) bool {
		return strings.ToLower(result.Warnings[i].Path) < strings.ToLower(result.Warnings[j].Path)
	})
	return result, nil
}

func inspectProxyEvidence(targetPath string, options DiscoveryOptions) ([]ProxyEvidence, []string) {
	evidence := make([]ProxyEvidence, 0, len(supportedLocalProxies))
	var conflicts []string
	reShadeCount := 0
	for _, filename := range supportedLocalProxies {
		path := filepath.Join(targetPath, filename)
		info, err := os.Stat(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		item := ProxyEvidence{
			Filename: filename,
			Exists:   true,
		}
		if err != nil || !info.Mode().IsRegular() {
			item.Conflict = "The proxy cannot be inspected as a regular file."
			conflicts = append(conflicts, filename+": "+item.Conflict)
			evidence = append(evidence, item)
			continue
		}
		metadata, metadataErr := options.ReadMetadata(path)
		if metadataErr != nil || !isReShadeMetadata(metadata) {
			if discoveryAllowsForeignProxy(path, options) {
				evidence = append(evidence, item)
				continue
			}
			item.Conflict = "An existing foreign or unidentified rendering proxy blocks managed ReShade."
			conflicts = append(conflicts, filename+": "+item.Conflict)
			evidence = append(evidence, item)
			continue
		}
		item.IsReShade = true
		item.RuntimeVersion = runtimeVersionFromMetadata(metadata)
		item.Architecture, _ = options.InspectArchitecture(path)
		reShadeCount++
		evidence = append(evidence, item)
	}
	if reShadeCount > 1 {
		conflicts = append(conflicts, "Multiple ReShade DirectX proxies were detected in the same target.")
	}
	return evidence, conflicts
}

func normalizeDiscoveryAllowedProxyPaths(paths []string) []string {
	result := make([]string, 0, len(paths))
	for _, path := range paths {
		absolute, err := filepath.Abs(path)
		if err != nil {
			continue
		}
		result = append(result, strings.ToLower(filepath.Clean(absolute)))
	}
	return result
}

func discoveryAllowsForeignProxy(path string, options DiscoveryOptions) bool {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	key := strings.ToLower(filepath.Clean(absolute))
	for _, allowed := range options.AllowedForeignProxyPaths {
		if key == allowed {
			return true
		}
	}
	return false
}

func localAPIOptions() []APIProxyOptions {
	return []APIProxyOptions{
		{
			RenderingAPI: RenderingAPID3D9,
			Proxies:      []string{"d3d9.dll"},
		},
		{
			RenderingAPI: RenderingAPID3D10,
			Proxies:      []string{"d3d10.dll", "d3d10core.dll", "dxgi.dll"},
		},
		{
			RenderingAPI: RenderingAPID3D11,
			Proxies:      []string{"d3d11.dll", "dxgi.dll"},
		},
		{
			RenderingAPI: RenderingAPID3D12,
			Proxies:      []string{"d3d12.dll", "dxgi.dll"},
		},
		{
			RenderingAPI: RenderingAPIOpenGL,
			Proxies:      []string{"opengl32.dll"},
		},
	}
}

func discoveryWarning(gameRoot string, path string, err error) DiscoveryWarning {
	relativePath, relativeErr := filepath.Rel(gameRoot, path)
	if relativeErr != nil {
		relativePath = path
	}
	return DiscoveryWarning{
		Path:    relativePath,
		Message: err.Error(),
	}
}

func isReShadeMetadata(metadata winversion.Metadata) bool {
	originalFilename := strings.TrimSpace(metadata.OriginalFilename)
	return strings.EqualFold(strings.TrimSpace(metadata.ProductName), "ReShade") &&
		(strings.EqualFold(originalFilename, "ReShade64.dll") ||
			strings.EqualFold(originalFilename, "ReShade32.dll"))
}

func runtimeVersionFromMetadata(metadata winversion.Metadata) string {
	version := firstSemanticVersion(metadata.ProductVersion)
	if version == "" {
		version = firstSemanticVersion(metadata.FileVersion)
	}
	return version
}
