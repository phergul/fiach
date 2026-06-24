package reshade

import (
	"fmt"
	"strings"
	"testing"
)

func TestInstallerDownloadURLRejectsRelativeBase(t *testing.T) {
	t.Parallel()

	_, err := installerDownloadURL("/downloads", "6.7.3", InstallerVariantStandard)
	if err == nil {
		t.Fatal("installerDownloadURL() error = nil, want error")
	}
	if !strings.Contains(fmt.Sprint(err), "not absolute") {
		t.Fatalf("installerDownloadURL() error = %q, want not absolute", err)
	}
}
