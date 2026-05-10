//go:build windows

package steam

import (
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"
)

func defaultSteamRootCandidates() []string {
	candidates := steamRegistryCandidates()
	candidates = append(candidates, windowsCommonSteamCandidates(
		os.Getenv("SystemDrive"),
		os.Getenv("ProgramFiles(x86)"),
		os.Getenv("ProgramFiles"),
	)...)

	return candidates
}

func steamRegistryCandidates() []string {
	type registryCandidate struct {
		key   registry.Key
		path  string
		value string
	}

	candidates := []registryCandidate{
		{key: registry.CURRENT_USER, path: `Software\Valve\Steam`, value: "SteamPath"},
		{key: registry.CURRENT_USER, path: `Software\Valve\Steam`, value: "InstallPath"},
		{key: registry.LOCAL_MACHINE, path: `Software\Valve\Steam`, value: "InstallPath"},
		{key: registry.LOCAL_MACHINE, path: `Software\WOW6432Node\Valve\Steam`, value: "InstallPath"},
	}

	paths := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		key, err := registry.OpenKey(candidate.key, candidate.path, registry.QUERY_VALUE)
		if err != nil {
			continue
		}

		value, _, err := key.GetStringValue(candidate.value)
		_ = key.Close()
		if err != nil {
			continue
		}

		paths = append(paths, filepath.FromSlash(value))
	}

	return paths
}

func windowsCommonSteamCandidates(systemDrive string, programFilesX86 string, programFiles string) []string {
	candidates := make([]string, 0, 4)
	candidates = appendWindowsSteamCandidate(candidates, programFilesX86)
	candidates = appendWindowsSteamCandidate(candidates, programFiles)

	if strings.TrimSpace(systemDrive) != "" {
		systemDriveRoot := windowsDriveRoot(systemDrive)
		candidates = append(candidates,
			filepath.Join(systemDriveRoot, "Program Files (x86)", "Steam"),
			filepath.Join(systemDriveRoot, "Program Files", "Steam"),
		)
	}

	return candidates
}

func appendWindowsSteamCandidate(candidates []string, parent string) []string {
	parent = strings.TrimSpace(parent)
	if parent == "" {
		return candidates
	}

	return append(candidates, filepath.Join(parent, "Steam"))
}

func windowsDriveRoot(systemDrive string) string {
	systemDrive = strings.TrimSpace(systemDrive)
	if strings.HasSuffix(systemDrive, `\`) || strings.HasSuffix(systemDrive, `/`) {
		return systemDrive
	}

	return systemDrive + `\`
}
