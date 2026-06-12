package services

import (
	"context"
	"testing"

	"github.com/phergul/fiach/internal/services/dto"
)

func TestModServiceUpdatesDetailsWithTagsAtomically(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertServiceProfileTestGame(t, store, "Final Fantasy VII", "/games/ff7")
	modID := insertServiceProfileTestMod(t, store, gameID, "Original", "/mods/original")
	service := NewModService(store, testLogger())

	updated, err := service.UpdateModDetails(context.Background(), dto.UpdateModDetailsInput{
		ModID: modID,
		Name:  "Aerith Dress",
		Metadata: dto.UpdateModMetadataInput{
			ModID:       modID,
			Version:     dto.ModMetadataFieldUpdate{Mode: dto.ModMetadataFieldUpdateModeReset},
			Author:      dto.ModMetadataFieldUpdate{Mode: dto.ModMetadataFieldUpdateModeReset},
			Description: dto.ModMetadataFieldUpdate{Mode: dto.ModMetadataFieldUpdateModeReset},
			SourceURL:   dto.ModMetadataFieldUpdate{Mode: dto.ModMetadataFieldUpdateModeReset},
		},
		NewTags: []dto.CreateTagInput{
			{Name: "Aerith", Color: dto.TagColorPink},
			{Name: "Character", Color: dto.TagColorPurple},
		},
	})
	if err != nil {
		t.Fatalf("UpdateModDetails() error = %v", err)
	}
	if updated.Name != "Aerith Dress" || len(updated.Tags) != 2 ||
		updated.Tags[0].Name != "Aerith" || updated.Tags[1].Name != "Character" {
		t.Fatalf("UpdateModDetails() = %+v, want renamed mod with tags", updated)
	}

	_, err = service.UpdateModDetails(context.Background(), dto.UpdateModDetailsInput{
		ModID: modID,
		Name:  "Should Roll Back",
		Metadata: dto.UpdateModMetadataInput{
			ModID:       modID,
			Version:     dto.ModMetadataFieldUpdate{Mode: dto.ModMetadataFieldUpdateModeReset},
			Author:      dto.ModMetadataFieldUpdate{Mode: dto.ModMetadataFieldUpdateModeReset},
			Description: dto.ModMetadataFieldUpdate{Mode: dto.ModMetadataFieldUpdateModeReset},
			SourceURL:   dto.ModMetadataFieldUpdate{Mode: dto.ModMetadataFieldUpdateModeReset},
		},
		NewTags: []dto.CreateTagInput{
			{Name: "Invalid", Color: "chartreuse"},
		},
	})
	if err == nil {
		t.Fatal("UpdateModDetails() invalid error = nil")
	}

	mods, err := service.ListMods(context.Background(), gameID)
	if err != nil {
		t.Fatalf("ListMods() error = %v", err)
	}
	if len(mods) != 1 || mods[0].Name != "Aerith Dress" || len(mods[0].Tags) != 2 {
		t.Fatalf("ListMods() = %+v, want previous details preserved", mods)
	}
}

func TestModServiceImportCreatesAndMergesTags(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertServiceProfileTestGame(t, store, "Final Fantasy VII", t.TempDir())
	service := NewModService(store, testLogger())
	sourcePath := makeSourceFolder(t, map[string]string{"mod.txt": "contents"})

	first, err := service.ImportMod(context.Background(), dto.ImportModInput{
		GameID:             gameID,
		Name:               "Aerith Dress",
		SourceType:         dto.ModSourceTypeFolder,
		SourcePath:         sourcePath,
		StrategyType:       dto.StrategyTypeGenericCopy,
		TargetRelativePath: ".",
		NewTags: []dto.CreateTagInput{
			{Name: "Aerith", Color: dto.TagColorPink},
		},
	})
	if err != nil {
		t.Fatalf("ImportMod() first error = %v", err)
	}
	if len(first.Mod.Tags) != 1 || first.Mod.Tags[0].Name != "Aerith" {
		t.Fatalf("ImportMod() first tags = %+v, want Aerith", first.Mod.Tags)
	}

	second, err := service.ImportMod(context.Background(), dto.ImportModInput{
		GameID:             gameID,
		Name:               "Ignored Existing Name",
		SourceType:         dto.ModSourceTypeFolder,
		SourcePath:         sourcePath,
		StrategyType:       dto.StrategyTypeGenericCopy,
		TargetRelativePath: ".",
		NewTags: []dto.CreateTagInput{
			{Name: "Character", Color: dto.TagColorPurple},
		},
	})
	if err != nil {
		t.Fatalf("ImportMod() second error = %v", err)
	}
	if second.Mod.ID != first.Mod.ID || len(second.Mod.Tags) != 2 ||
		second.Mod.Tags[0].Name != "Aerith" || second.Mod.Tags[1].Name != "Character" {
		t.Fatalf("ImportMod() second = %+v, want existing mod with merged tags", second.Mod)
	}
}
