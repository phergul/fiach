package services

import (
	"context"
	"strings"
	"testing"
)

func TestShellServiceOpenDirectoryValidatesPath(t *testing.T) {
	t.Parallel()

	service := NewShellService(nil)

	err := service.OpenDirectory(context.Background(), "")
	if err == nil || !strings.Contains(err.Error(), "directory path is required") {
		t.Fatalf("OpenDirectory() error = %v, want required path validation", err)
	}

	err = service.OpenDirectory(context.Background(), t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "application is not configured") {
		t.Fatalf("OpenDirectory() error = %v, want application configuration error", err)
	}
}
