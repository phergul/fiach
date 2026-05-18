package installpath

import (
	"path/filepath"
	"testing"
)

func TestResolveSourceRootUsesManagedSourcePathWhenSourceSubpathIsUnset(t *testing.T) {
	t.Parallel()

	if got := ResolveSourceRoot("/managed/mod", nil); got != "/managed/mod" {
		t.Fatalf("ResolveSourceRoot() = %q, want /managed/mod", got)
	}
}

func TestResolveSourceRootJoinsSourceSubpath(t *testing.T) {
	t.Parallel()

	sourceSubpath := "plugins/core"
	want := filepath.Join("/managed/mod", "plugins", "core")
	if got := ResolveSourceRoot("/managed/mod", &sourceSubpath); got != want {
		t.Fatalf("ResolveSourceRoot() = %q, want %q", got, want)
	}
}

func TestJoinTargetRelativePath(t *testing.T) {
	t.Parallel()

	if got := JoinTargetRelativePath("BepInEx/plugins", "plugins/core.dll"); got != "BepInEx/plugins/plugins/core.dll" {
		t.Fatalf("JoinTargetRelativePath() = %q, want BepInEx/plugins/plugins/core.dll", got)
	}
	if got := JoinTargetRelativePath(".", "Data/SkyUI.esp"); got != "Data/SkyUI.esp" {
		t.Fatalf("JoinTargetRelativePath() = %q, want Data/SkyUI.esp", got)
	}
}
