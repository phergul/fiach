package updater

import (
	"testing"

	"github.com/wailsapp/wails/v3/pkg/updater"
	"github.com/wailsapp/wails/v3/pkg/updater/providers/github"
)

func TestAssetMatcherSkipsInstallerAndPackages(t *testing.T) {
	assets := []github.ReleaseAsset{
		{Name: "fiach_windows_amd64_installer.exe"},
		{Name: "fiach_windows_amd64.exe"},
		{Name: "fiach_linux_amd64.deb"},
		{Name: "fiach_linux_amd64.rpm"},
		{Name: "fiach_linux_amd64.AppImage"},
	}

	req := updater.CheckRequest{
		Platform: "windows",
		Arch:     "amd64",
	}

	got := assetMatcher(req, assets)
	if got != 1 {
		t.Fatalf("assetMatcher() = %d, want 1 (portable exe)", got)
	}
	if assets[got].Name != "fiach_windows_amd64.exe" {
		t.Fatalf("picked asset = %q, want fiach_windows_amd64.exe", assets[got].Name)
	}
}

func TestAssetMatcherPicksLinuxBinary(t *testing.T) {
	assets := []github.ReleaseAsset{
		{Name: "fiach_linux_amd64.deb"},
		{Name: "fiach_linux_amd64"},
	}

	req := updater.CheckRequest{
		Platform: "linux",
		Arch:     "amd64",
	}

	got := assetMatcher(req, assets)
	if got != 1 {
		t.Fatalf("assetMatcher() = %d, want 1 (bare binary)", got)
	}
}

func TestAssetMatcherPicksDarwinZip(t *testing.T) {
	assets := []github.ReleaseAsset{
		{Name: "fiach_darwin_amd64.zip"},
		{Name: "fiach_darwin_arm64.zip"},
	}

	req := updater.CheckRequest{
		Platform: "darwin",
		Arch:     "arm64",
	}

	got := assetMatcher(req, assets)
	if got != 1 {
		t.Fatalf("assetMatcher() = %d, want 1 (arm64 zip)", got)
	}
}
