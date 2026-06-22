package services

import (
	"context"
	"net/http"
	"net/http/httptest"
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

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "rate limited", http.StatusForbidden)
	}))
	defer server.Close()

	service := NewOptiScalerService(nil, testLogger(), nil)
	service.manager = optiscaler.NewManager(nil, optiscaler.ManagerOptions{
		ReleasesURL: server.URL,
		HTTPClient:  server.Client(),
	})

	status, err := service.GetOptiScalerReleaseStatus(context.Background())
	if err != nil {
		t.Fatalf("GetOptiScalerReleaseStatus() error = %v, want nil", err)
	}
	if status.Error != "GitHub returned 403 Forbidden while checking the latest OptiScaler release." {
		t.Fatalf("GetOptiScalerReleaseStatus() error status = %q, want 403 message", status.Error)
	}
}
