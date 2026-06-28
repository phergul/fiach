package mappers_test

import (
	"testing"

	"github.com/phergul/fiach/internal/services/dto/mappers"
	"github.com/phergul/fiach/internal/storage/dbtypes"
)

func TestToDTOProfileModSetsDisplayLoadOrder(t *testing.T) {
	t.Parallel()

	profileMod := mappers.ToDTOProfileMod(dbtypes.ProfileMod{
		ProfileID: 1,
		ModID:     2,
		LoadOrder: 0,
	})

	if profileMod.DisplayLoadOrder != 1 {
		t.Fatalf("DisplayLoadOrder = %d, want 1", profileMod.DisplayLoadOrder)
	}
}
