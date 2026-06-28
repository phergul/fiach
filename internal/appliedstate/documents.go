package appliedstate

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
)

const DocumentVersion = 2

type ProfileCompositionDocument struct {
	Version   int                     `json:"version"`
	ProfileID int64                   `json:"profileId"`
	Mods      []ProfileCompositionMod `json:"mods"`
}

type ProfileCompositionMod struct {
	ModID            int64  `json:"modId"`
	Enabled          bool   `json:"enabled"`
	LoadOrder        int64  `json:"loadOrder"`
	SourcePath       string `json:"sourcePath"`
	PackageUpdatedAt string `json:"packageUpdatedAt"`
}

type EncodedSnapshot struct {
	JSON string
	Hash string
}

func BuildProfileCompositionDocument(profileID int64, mods []ProfileCompositionMod) ProfileCompositionDocument {
	copiedMods := make([]ProfileCompositionMod, len(mods))
	copy(copiedMods, mods)
	sort.SliceStable(copiedMods, func(i int, j int) bool {
		if copiedMods[i].LoadOrder != copiedMods[j].LoadOrder {
			return copiedMods[i].LoadOrder < copiedMods[j].LoadOrder
		}

		return copiedMods[i].ModID < copiedMods[j].ModID
	})

	return ProfileCompositionDocument{
		Version:   DocumentVersion,
		ProfileID: profileID,
		Mods:      copiedMods,
	}
}

func EncodeProfileCompositionSnapshot(document ProfileCompositionDocument) (snapshot EncodedSnapshot, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("marshal profile composition snapshot: %w", err)
		}
	}()

	encoded, err := json.Marshal(document)
	if err != nil {
		return EncodedSnapshot{}, err
	}

	sum := sha256.Sum256(encoded)
	return EncodedSnapshot{
		JSON: string(encoded),
		Hash: hex.EncodeToString(sum[:]),
	}, nil
}
