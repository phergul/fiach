package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/phergul/fiach/internal/storage"
	"github.com/phergul/fiach/internal/storage/dbtypes"
)

type testAppliedFileSeed struct {
	GameRelativePath   string
	BaselineExists     bool
	BaselineSHA256     *string
	BaselineSizeBytes  *int64
	BaselineBackupPath *string
	AppliedSHA256      string
	AppliedSizeBytes   int64
}

func saveTestAppliedProfileState(
	t *testing.T,
	store *storage.Store,
	gameID int64,
	profileID int64,
	fileStates []testAppliedFileSeed,
	createdDirectories []dbtypes.AppliedCreatedDirectoryRow,
) {
	t.Helper()

	rows := make([]dbtypes.AppliedFileStateRow, len(fileStates))
	for index, seed := range fileStates {
		appliedSHA256 := seed.AppliedSHA256
		appliedSizeBytes := seed.AppliedSizeBytes
		rows[index] = dbtypes.AppliedFileStateRow{
			GameID:             gameID,
			GameRelativePath:   seed.GameRelativePath,
			ProfileID:          profileID,
			BaselineExists:     seed.BaselineExists,
			BaselineSHA256:     seed.BaselineSHA256,
			BaselineSizeBytes:  seed.BaselineSizeBytes,
			BaselineBackupPath: seed.BaselineBackupPath,
			AppliedExists:      true,
			AppliedSHA256:      &appliedSHA256,
			AppliedSizeBytes:   &appliedSizeBytes,
			OutputKind:         "copied",
			LastAppliedAt:      "2026-06-27T00:00:00Z",
		}
	}

	if _, err := store.SaveAppliedProfileState(context.Background(), dbtypes.SaveAppliedProfileStateInput{
		GameID:                    gameID,
		ProfileID:                 profileID,
		FileStates:                rows,
		ReplaceFileStates:         true,
		CreatedDirectories:        createdDirectories,
		ReplaceCreatedDirectories: len(createdDirectories) > 0,
	}); err != nil {
		t.Fatalf("SaveAppliedProfileState() error = %v", err)
	}
}

func testContentHash(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}

func testStringPtr(value string) *string {
	return &value
}

func testInt64Ptr(value int64) *int64 {
	return &value
}
