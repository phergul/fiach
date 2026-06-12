package storage

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/phergul/fiach/internal/storage/dbtypes"
)

func TestSetModTagsValidatesNamesAndAssignmentLimit(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertProfileTestGame(t, store, "Final Fantasy VII", "/games/ff7")
	modID := insertProfileTestMod(t, store, gameID, "Aerith Dress", "/mods/aerith")

	for _, name := range []string{"", "bad\nname", strings.Repeat("a", 51)} {
		_, err := store.SetModTags(context.Background(), dbtypes.SetModTagsInput{
			ModID: modID,
			NewTags: []dbtypes.CreateTagInput{
				{Name: name, Color: dbtypes.TagColorPink},
			},
		})
		if err == nil {
			t.Fatalf("SetModTags(%q) error = nil", name)
		}
	}

	newTags := make([]dbtypes.CreateTagInput, 0, 21)
	for index := 0; index < 21; index++ {
		newTags = append(newTags, dbtypes.CreateTagInput{
			Name:  fmt.Sprintf("Tag %02d", index),
			Color: dbtypes.TagColorBlue,
		})
	}
	if _, err := store.SetModTags(context.Background(), dbtypes.SetModTagsInput{
		ModID:   modID,
		NewTags: newTags,
	}); err == nil || !strings.Contains(err.Error(), "at most 20 tags") {
		t.Fatalf("SetModTags() limit error = %v", err)
	}

	gameTags, err := store.ListGameTags(context.Background(), gameID)
	if err != nil {
		t.Fatalf("ListGameTags() error = %v", err)
	}
	if len(gameTags) != 0 {
		t.Fatalf("ListGameTags() = %+v, want failed assignments rolled back", gameTags)
	}
}

func TestSetModTagsCreatesAndListsTagsAlphabetically(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertProfileTestGame(t, store, "Final Fantasy VII", "/games/ff7")
	modID := insertProfileTestMod(t, store, gameID, "Aerith Dress", "/mods/aerith")

	tags, err := store.SetModTags(context.Background(), dbtypes.SetModTagsInput{
		ModID: modID,
		NewTags: []dbtypes.CreateTagInput{
			{Name: " Character ", Color: dbtypes.TagColorPurple},
			{Name: "Aerith", Color: dbtypes.TagColorPink},
		},
	})
	if err != nil {
		t.Fatalf("SetModTags() error = %v", err)
	}
	if len(tags) != 2 || tags[0].Name != "Aerith" || tags[1].Name != "Character" {
		t.Fatalf("SetModTags() = %+v, want alphabetical tags", tags)
	}

	gameTags, err := store.ListGameTags(context.Background(), gameID)
	if err != nil {
		t.Fatalf("ListGameTags() error = %v", err)
	}
	if len(gameTags) != 2 || gameTags[0].NormalizedName != "aerith" || gameTags[1].NormalizedName != "character" {
		t.Fatalf("ListGameTags() = %+v, want normalized alphabetical tags", gameTags)
	}
}

func TestSetModTagsRetainsUnusedTagsAndRejectsCrossGameAssignments(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertProfileTestGame(t, store, "Final Fantasy VII", "/games/ff7")
	otherGameID := insertProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	modID := insertProfileTestMod(t, store, gameID, "Aerith Dress", "/mods/aerith")
	otherModID := insertProfileTestMod(t, store, otherGameID, "SkyUI", "/mods/skyui")

	otherTags, err := store.SetModTags(context.Background(), dbtypes.SetModTagsInput{
		ModID: otherModID,
		NewTags: []dbtypes.CreateTagInput{
			{Name: "Interface", Color: dbtypes.TagColorBlue},
		},
	})
	if err != nil {
		t.Fatalf("SetModTags() other error = %v", err)
	}

	_, err = store.SetModTags(context.Background(), dbtypes.SetModTagsInput{
		ModID:  modID,
		TagIDs: []int64{otherTags[0].ID},
	})
	if err == nil || !strings.Contains(err.Error(), "does not belong to mod game") {
		t.Fatalf("SetModTags() error = %v, want cross-game rejection", err)
	}

	created, err := store.SetModTags(context.Background(), dbtypes.SetModTagsInput{
		ModID: modID,
		NewTags: []dbtypes.CreateTagInput{
			{Name: "Aerith", Color: dbtypes.TagColorPink},
		},
	})
	if err != nil {
		t.Fatalf("SetModTags() create error = %v", err)
	}
	if _, err := store.SetModTags(context.Background(), dbtypes.SetModTagsInput{ModID: modID}); err != nil {
		t.Fatalf("SetModTags() clear error = %v", err)
	}

	gameTags, err := store.ListGameTags(context.Background(), gameID)
	if err != nil {
		t.Fatalf("ListGameTags() error = %v", err)
	}
	if len(gameTags) != 1 || gameTags[0].ID != created[0].ID {
		t.Fatalf("ListGameTags() = %+v, want unused tag retained", gameTags)
	}
}

func TestRenameTagMergesAssignmentsAndKeepsTargetColor(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertProfileTestGame(t, store, "Final Fantasy VII", "/games/ff7")
	firstModID := insertProfileTestMod(t, store, gameID, "Aerith Dress", "/mods/aerith")
	secondModID := insertProfileTestMod(t, store, gameID, "Cloud Dress", "/mods/cloud")
	firstTags, err := store.SetModTags(context.Background(), dbtypes.SetModTagsInput{
		ModID: firstModID,
		NewTags: []dbtypes.CreateTagInput{
			{Name: "Heroine", Color: dbtypes.TagColorPink},
		},
	})
	if err != nil {
		t.Fatalf("SetModTags() first error = %v", err)
	}
	targetTags, err := store.SetModTags(context.Background(), dbtypes.SetModTagsInput{
		ModID: secondModID,
		NewTags: []dbtypes.CreateTagInput{
			{Name: "Character", Color: dbtypes.TagColorPurple},
		},
	})
	if err != nil {
		t.Fatalf("SetModTags() second error = %v", err)
	}

	merged, err := store.RenameTag(context.Background(), firstTags[0].ID, " CHARACTER ", dbtypes.TagColorRed)
	if err != nil {
		t.Fatalf("RenameTag() error = %v", err)
	}
	if merged.ID != targetTags[0].ID || merged.Color != dbtypes.TagColorPurple {
		t.Fatalf("RenameTag() = %+v, want target tag and color", merged)
	}

	tagsByModID, err := store.ListTagsForMods(context.Background(), []int64{firstModID, secondModID})
	if err != nil {
		t.Fatalf("ListTagsForMods() error = %v", err)
	}
	if len(tagsByModID[firstModID]) != 1 || tagsByModID[firstModID][0].ID != merged.ID ||
		len(tagsByModID[secondModID]) != 1 || tagsByModID[secondModID][0].ID != merged.ID {
		t.Fatalf("ListTagsForMods() = %+v, want merged assignments", tagsByModID)
	}
}

func TestUpdateModDetailsRollsBackOnInvalidTags(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertProfileTestGame(t, store, "Final Fantasy VII", "/games/ff7")
	modID := insertProfileTestMod(t, store, gameID, "Original", "/mods/original")

	_, _, _, err := store.UpdateModDetails(context.Background(), dbtypes.UpdateModDetailsInput{
		ModID: modID,
		Name:  "Changed",
		Metadata: dbtypes.UpdateModMetadataInput{
			ModID: modID,
			Notes: stringPtr("changed"),
		},
		NewTags: []dbtypes.CreateTagInput{
			{Name: "Invalid", Color: "chartreuse"},
		},
	})
	if err == nil {
		t.Fatal("UpdateModDetails() error = nil, want invalid color error")
	}

	mod, found, err := store.GetMod(context.Background(), modID)
	if err != nil || !found {
		t.Fatalf("GetMod() = %+v, %v, %v", mod, found, err)
	}
	if mod.Name != "Original" {
		t.Fatalf("GetMod().Name = %q, want rollback to Original", mod.Name)
	}
	metadata, found, err := store.GetModMetadata(context.Background(), modID)
	if err != nil || !found {
		t.Fatalf("GetModMetadata() = %+v, %v, %v", metadata, found, err)
	}
	if metadata.Notes != nil {
		t.Fatalf("GetModMetadata().Notes = %v, want rollback", metadata.Notes)
	}
}

func TestEnsureModInstallConfigAndMergeTagsRollsBackTogether(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertProfileTestGame(t, store, "Final Fantasy VII", "/games/ff7")
	modID := insertProfileTestMod(t, store, gameID, "Aerith Dress", "/mods/aerith")

	_, _, err := store.EnsureModInstallConfigAndMergeTags(
		context.Background(),
		dbtypes.CreateModInstallConfigInput{
			ModID:              modID,
			StrategyType:       "generic_copy",
			TargetBase:         "game_root",
			TargetRelativePath: ".",
		},
		dbtypes.SetModTagsInput{
			NewTags: []dbtypes.CreateTagInput{
				{Name: "Invalid", Color: "chartreuse"},
			},
		},
	)
	if err == nil {
		t.Fatal("EnsureModInstallConfigAndMergeTags() error = nil")
	}

	if _, found, err := store.GetModInstallConfig(context.Background(), modID); err != nil {
		t.Fatalf("GetModInstallConfig() error = %v", err)
	} else if found {
		t.Fatal("GetModInstallConfig() found = true, want transaction rollback")
	}
}

func TestUpdateModPackagePreservesTags(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertProfileTestGame(t, store, "Final Fantasy VII", "/games/ff7")
	modID := insertProfileTestMod(t, store, gameID, "Aerith Dress", "/mods/aerith")
	created, err := store.SetModTags(context.Background(), dbtypes.SetModTagsInput{
		ModID: modID,
		NewTags: []dbtypes.CreateTagInput{
			{Name: "Aerith", Color: dbtypes.TagColorPink},
		},
	})
	if err != nil {
		t.Fatalf("SetModTags() error = %v", err)
	}

	if _, err := store.UpdateModPackage(context.Background(), dbtypes.UpdateModPackageInput{
		ModID:              modID,
		SourceType:         dbtypes.ModSourceTypeArchive,
		OriginalSourcePath: "/imports/aerith-update.zip",
	}); err != nil {
		t.Fatalf("UpdateModPackage() error = %v", err)
	}

	tagsByModID, err := store.ListTagsForMods(context.Background(), []int64{modID})
	if err != nil {
		t.Fatalf("ListTagsForMods() error = %v", err)
	}
	if len(tagsByModID[modID]) != 1 || tagsByModID[modID][0].ID != created[0].ID {
		t.Fatalf("ListTagsForMods() = %+v, want tag preserved", tagsByModID)
	}
}
