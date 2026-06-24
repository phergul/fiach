package reshade

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/phergul/fiach/internal/thirdparty"
	"github.com/wailsapp/wails/v3/pkg/application"
)

const (
	DefaultTagsURL         = thirdparty.DefaultManifestURL
	DefaultDownloadBaseURL = "https://reshade.me/downloads"
)

type InstallerVariant string

const (
	InstallerVariantStandard InstallerVariant = ""
	InstallerVariantAddon    InstallerVariant = "addon"
)

func DefaultInstallerCacheDir() string {
	return filepath.Join(application.Path(application.PathCacheHome), "fiach", "reshade", "installers")
}

func installerDownloadURL(downloadBaseURL string, installerVersion string, variant InstallerVariant) (string, error) {
	base, err := url.Parse(downloadBaseURL)
	if err != nil {
		return "", fmt.Errorf("parse ReShade download base URL: %w", err)
	}
	if base.Scheme == "" || base.Host == "" {
		return "", fmt.Errorf("parse ReShade download base URL: %q is not absolute", downloadBaseURL)
	}

	base.Path = strings.TrimRight(base.Path, "/") + "/" + installerFileName(installerVersion, variant)
	return base.String(), nil
}

func installerFileName(installerVersion string, variant InstallerVariant) string {
	if variant == InstallerVariantAddon {
		return fmt.Sprintf("ReShade_Setup_%s_Addon.exe", installerVersion)
	}

	return fmt.Sprintf("ReShade_Setup_%s.exe", installerVersion)
}

func installerDescription(variant InstallerVariant) string {
	if variant == InstallerVariantAddon {
		return "ReShade add-on installer"
	}

	return "ReShade installer"
}
