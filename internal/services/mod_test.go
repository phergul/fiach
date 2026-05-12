package services

import (
	"strings"
	"testing"
)

func TestModServiceListsMods(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	gameID := insertServiceProfileTestGame(t, store, "Skyrim", "/games/skyrim")
	modID := insertServiceProfileTestMod(t, store, gameID, "SkyUI", "/mods/skyui")
	service := NewModService(store)

	mods, err := service.ListMods(gameID)
	if err != nil {
		t.Fatalf("ListMods() error = %v", err)
	}
	if len(mods) != 1 || mods[0].ID != modID {
		t.Fatalf("ListMods() = %+v, want inserted mod", mods)
	}
}

func TestModServiceReturnsStorageConfigurationError(t *testing.T) {
	t.Parallel()

	service := NewModService(nil)

	_, err := service.ListMods(1)
	if err == nil {
		t.Fatal("ListMods() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "list mods") || !strings.Contains(err.Error(), "storage is not configured") {
		t.Fatalf("ListMods() error = %q, want service context", err.Error())
	}
}

func TestModServiceWrapsStorageErrors(t *testing.T) {
	t.Parallel()

	store := openMigratedStore(t)
	defer closeStore(t, store)

	if _, err := store.DB().Exec(`DROP TABLE mods`); err != nil {
		t.Fatalf("drop mods table: %v", err)
	}

	service := NewModService(store)
	_, err := service.ListMods(1)
	if err == nil {
		t.Fatal("ListMods() error = nil, want storage error")
	}
	if !strings.Contains(err.Error(), "list mods") || !strings.Contains(err.Error(), "select game mods") {
		t.Fatalf("ListMods() error = %q, want distinct service and storage context", err.Error())
	}
}
