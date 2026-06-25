package apperror

import (
	"errors"
	"fmt"
	"testing"
)

func TestWrapPreservesCauseChain(t *testing.T) {
	t.Parallel()

	root := errors.New("duplicate profile name")
	wrapped := fmt.Errorf("insert profile row: %w", root)
	userErr := Wrap("A profile with this name already exists for this game.", wrapped)

	if userErr.Error() != "A profile with this name already exists for this game." {
		t.Fatalf("Error() = %q", userErr.Error())
	}
	if !errors.Is(userErr, root) {
		t.Fatal("errors.Is(userErr, root) = false, want true")
	}
}

func TestNewReturnsUserFacingMessage(t *testing.T) {
	t.Parallel()

	err := New("Profile name is required.")

	if err.Error() != "Profile name is required." {
		t.Fatalf("Error() = %q", err.Error())
	}
	if !IsUserError(err) {
		t.Fatal("IsUserError(err) = false, want true")
	}
}

func TestUserMessageFindsOuterFriendlyMessage(t *testing.T) {
	t.Parallel()

	root := errors.New("duplicate profile name")
	wrapped := fmt.Errorf("insert profile row: %w", root)
	err := Wrap("A profile with this name already exists for this game.", wrapped)

	if got := UserMessage(err); got != "A profile with this name already exists for this game." {
		t.Fatalf("UserMessage() = %q", got)
	}
}

func TestUserMessageReturnsEmptyForPlainError(t *testing.T) {
	t.Parallel()

	err := errors.New("insert profile row: constraint failed")

	if got := UserMessage(err); got != "" {
		t.Fatalf("UserMessage() = %q, want empty", got)
	}
}

func TestDetailWalksFullChain(t *testing.T) {
	t.Parallel()

	root := errors.New("constraint failed: UNIQUE constraint failed")
	wrapped := fmt.Errorf("insert profile row: %w", root)
	err := Wrap("A profile with this name already exists for this game.", wrapped)

	want := "A profile with this name already exists for this game.: insert profile row: constraint failed: UNIQUE constraint failed"
	if got := Detail(err); got != want {
		t.Fatalf("Detail() = %q, want %q", got, want)
	}
}

func TestDetailDeduplicatesRepeatedSegments(t *testing.T) {
	t.Parallel()

	err := fmt.Errorf("create profile: %w", fmt.Errorf("create profile: %w", errors.New("permission denied")))

	if got := Detail(err); got != "create profile: permission denied" {
		t.Fatalf("Detail() = %q", got)
	}
}
