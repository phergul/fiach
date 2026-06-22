package services

import (
	"context"
	"strings"
	"testing"

	"github.com/phergul/fiach/internal/optiscaler"
)

func TestOptiScalerServiceRejectsUnsupportedPlatformWithServiceContext(t *testing.T) {
	t.Parallel()
	service := NewOptiScalerService(nil, testLogger(), nil)
	service.operatingSystem = "linux"
	_, err := service.PreviewOptiScalerAction(context.Background(), optiscaler.Request{GameID: 1})
	if err == nil || !strings.Contains(err.Error(), "preview game OptiScaler action") ||
		!strings.Contains(err.Error(), "only supported on Windows") {
		t.Fatalf("PreviewOptiScalerAction() error = %v", err)
	}
}

func TestOptiScalerReleaseStatusReturnsLookupErrorAsStatus(t *testing.T) {
	t.Parallel()

	service := NewOptiScalerService(nil, testLogger(), nil)
	service.manager = optiscaler.NewManager(nil, optiscaler.ManagerOptions{
		ReleaseManifest: []byte(`{`),
	})

	status, err := service.GetOptiScalerReleaseStatus(context.Background(), false)
	if err != nil {
		t.Fatalf("GetOptiScalerReleaseStatus() error = %v, want nil", err)
	}
	if !strings.Contains(status.Error, "parse third-party release manifest") {
		t.Fatalf("GetOptiScalerReleaseStatus() error status = %q, want manifest parse message", status.Error)
	}
}
