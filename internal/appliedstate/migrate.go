package appliedstate

import "fmt"

func FileStatesFromStoredManifest(manifestJSON string, installPath string, profileID int64, appliedAt string) ([]PersistedFileState, error) {
	document, err := DecodeManifest(manifestJSON)
	if err != nil {
		return nil, fmt.Errorf("decode stored manifest: %w", err)
	}

	states, err := BuildFileStatesFromManifest(document, installPath, profileID, appliedAt)
	if err != nil {
		return nil, err
	}

	return states, nil
}
