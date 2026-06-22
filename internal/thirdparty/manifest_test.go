package thirdparty

import (
	"strings"
	"testing"
)

func TestParseAcceptsValidManifest(t *testing.T) {
	t.Parallel()

	if _, err := Parse(validManifest()); err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
}

func TestParseRejectsUnsupportedSchemaVersion(t *testing.T) {
	t.Parallel()

	contents := strings.Replace(string(validManifest()), `"version": 1`, `"version": 2`, 1)
	_, err := Parse([]byte(contents))
	if err == nil || !strings.Contains(err.Error(), "unsupported schema version") {
		t.Fatalf("Parse() error = %v, want unsupported version", err)
	}
}

func TestParseRejectsInvalidHashLength(t *testing.T) {
	t.Parallel()

	contents := strings.Replace(string(validManifest()), strings.Repeat("a", 64), "abcd", 1)
	_, err := Parse([]byte(contents))
	if err == nil || !strings.Contains(err.Error(), "SHA-256") {
		t.Fatalf("Parse() error = %v, want SHA-256 error", err)
	}
}

func TestParseRejectsNonPositiveSize(t *testing.T) {
	t.Parallel()

	contents := strings.Replace(string(validManifest()), `"sizeBytes": 100`, `"sizeBytes": 0`, 1)
	_, err := Parse([]byte(contents))
	if err == nil || !strings.Contains(err.Error(), "size must be positive") {
		t.Fatalf("Parse() error = %v, want size error", err)
	}
}

func TestParseRejectsUntrustedURL(t *testing.T) {
	t.Parallel()

	contents := strings.Replace(
		string(validManifest()),
		"https://reshade.me/downloads/ReShade_Setup_6.7.3.exe",
		"http://example.invalid/ReShade_Setup_6.7.3.exe",
		1,
	)
	_, err := Parse([]byte(contents))
	if err == nil || !strings.Contains(err.Error(), "must use HTTPS") {
		t.Fatalf("Parse() error = %v, want HTTPS error", err)
	}
}

func validManifest() []byte {
	return []byte(`{
  "version": 1,
  "updatedAt": "2026-06-22T00:00:00Z",
  "tools": {
    "optiscaler": {
      "tag": "v0.9.3",
      "version": "OptiScaler v0.9.3",
      "assetName": "Optiscaler_0.9.3-final.20260618.7z",
      "url": "https://github.com/optiscaler/OptiScaler/releases/download/v0.9.3/Optiscaler_0.9.3-final.20260618.7z",
      "sha256": "` + strings.Repeat("a", 64) + `",
      "sizeBytes": 100
    },
    "reshade": {
      "standard": {
        "version": "6.7.3",
        "assetName": "ReShade_Setup_6.7.3.exe",
        "url": "https://reshade.me/downloads/ReShade_Setup_6.7.3.exe",
        "sha256": "` + strings.Repeat("b", 64) + `",
        "sizeBytes": 200
      },
      "addon": {
        "version": "6.7.3",
        "assetName": "ReShade_Setup_6.7.3_Addon.exe",
        "url": "https://reshade.me/downloads/ReShade_Setup_6.7.3_Addon.exe",
        "sha256": "` + strings.Repeat("c", 64) + `",
        "sizeBytes": 300
      }
    }
  }
}`)
}
