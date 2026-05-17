package installconfig

import "testing"

func TestNormalizeTargetRelativePathAcceptsSafePaths(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"Data":               "Data",
		"BepInEx/plugins":    "BepInEx/plugins",
		"Content/Paks/~mods": "Content/Paks/~mods",
		".":                  ".",
		`Data\SKSE`:          "Data/SKSE",
	}

	for input, want := range tests {
		input := input
		want := want
		t.Run(input, func(t *testing.T) {
			t.Parallel()

			got, err := NormalizeTargetRelativePath(input)
			if err != nil {
				t.Fatalf("NormalizeTargetRelativePath(%q) error = %v", input, err)
			}
			if got != want {
				t.Fatalf("NormalizeTargetRelativePath(%q) = %q, want %q", input, got, want)
			}
		})
	}
}

func TestNormalizeTargetRelativePathRejectsUnsafePaths(t *testing.T) {
	t.Parallel()

	tests := []string{
		"",
		" ",
		"/Data",
		`C:\Games\Skyrim`,
		"../Data",
		"Data/../../Other",
	}

	for _, input := range tests {
		input := input
		t.Run(input, func(t *testing.T) {
			t.Parallel()

			if _, err := NormalizeTargetRelativePath(input); err == nil {
				t.Fatalf("NormalizeTargetRelativePath(%q) error = nil, want error", input)
			}
		})
	}
}
