package services

import (
	"context"
	"testing"
)

func TestShellServiceOpenDirectoryValidatesPath(t *testing.T) {
	t.Parallel()

	service := NewShellService(nil)

	err := service.OpenDirectory(context.Background(), "")
	if err == nil || err.Error() != "Directory path is required." {
		t.Fatalf("OpenDirectory() error = %v, want required path validation", err)
	}

	err = service.OpenDirectory(context.Background(), t.TempDir())
	if err == nil || err.Error() != "The application is not configured." {
		t.Fatalf("OpenDirectory() error = %v, want application configuration error", err)
	}
}
