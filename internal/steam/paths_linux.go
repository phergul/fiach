//go:build linux

package steam

import (
	"os"
	"path/filepath"
)

func defaultSteamRootCandidates() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	return []string{
		filepath.Join(home, ".local", "share", "Steam"),
		filepath.Join(home, ".steam", "steam"),
	}
}
