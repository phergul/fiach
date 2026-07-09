package theme

import (
	"strings"
	"testing"
)

func TestResolveDefaultsToDark(t *testing.T) {
	definition := Resolve("")
	if definition.ID != "dark" {
		t.Fatalf("Resolve(\"\") id = %q, want dark", definition.ID)
	}

	definition = Resolve("unknown-theme")
	if definition.ID != "dark" {
		t.Fatalf("Resolve(unknown) id = %q, want dark", definition.ID)
	}
}

func TestResolveUsesStoredTheme(t *testing.T) {
	definition := Resolve("lavender")
	if definition.ID != "lavender" {
		t.Fatalf("Resolve(lavender) id = %q, want lavender", definition.ID)
	}
}

func TestDefinitionsLoadsAllThemes(t *testing.T) {
	definitions := Definitions()
	if len(definitions) != 10 {
		t.Fatalf("len(Definitions()) = %d, want 10", len(definitions))
	}
}

func TestUpdaterCSSMapsUpdaterVariables(t *testing.T) {
	css := UpdaterCSS("dark")

	for _, variable := range []string{
		"--bg: #111315",
		"--surface: #16191c",
		"--accent: #72a17d",
		"--radius: 0",
		"color-scheme: dark",
	} {
		if !strings.Contains(css, variable) {
			t.Fatalf("updater css missing %q:\n%s", variable, css)
		}
	}
}

func TestCSSColorStripsAlpha(t *testing.T) {
	if got := CSSColor("#111315ff"); got != "#111315" {
		t.Fatalf("CSSColor() = %q, want #111315", got)
	}
}
