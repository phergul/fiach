package theme

import (
	"strings"
	"testing"
)

func TestResolveDefaultsToAsh(t *testing.T) {
	definition := Resolve("")
	if definition.ID != "ash" {
		t.Fatalf("Resolve(\"\") id = %q, want ash", definition.ID)
	}

	definition = Resolve("unknown-theme")
	if definition.ID != "ash" {
		t.Fatalf("Resolve(unknown) id = %q, want ash", definition.ID)
	}
}

func TestResolveUsesStoredTheme(t *testing.T) {
	definition := Resolve("midnight")
	if definition.ID != "midnight" {
		t.Fatalf("Resolve(midnight) id = %q, want midnight", definition.ID)
	}
}

func TestDefinitionsLoadsAllThemes(t *testing.T) {
	definitions := Definitions()
	if len(definitions) != 6 {
		t.Fatalf("len(Definitions()) = %d, want 6", len(definitions))
	}
}

func TestUpdaterCSSMapsUpdaterVariables(t *testing.T) {
	css := UpdaterCSS("ash")

	for _, variable := range []string{
		"--bg: #222120",
		"--surface: #272624",
		"--accent: #588b8b",
		"--radius: 0",
		"color-scheme: dark",
	} {
		if !strings.Contains(css, variable) {
			t.Fatalf("updater css missing %q:\n%s", variable, css)
		}
	}
}

func TestCSSColorStripsAlpha(t *testing.T) {
	if got := CSSColor("#222120ff"); got != "#222120" {
		t.Fatalf("CSSColor() = %q, want #222120", got)
	}
}
