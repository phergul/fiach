package optiscaler

import (
	"context"
	"testing"
)

func TestPreparePackageReusesPreparedPackage(t *testing.T) {
	t.Parallel()

	calls := 0
	wantRelease := Release{Tag: "v1", Digest: "digest"}
	wantPackage := Package{Root: t.TempDir()}
	manager := NewManager(newMemoryStore(), ManagerOptions{
		DataDir:  t.TempDir(),
		CacheDir: t.TempDir(),
		PreparePackage: func(context.Context) (Release, Package, error) {
			calls++
			return wantRelease, wantPackage, nil
		},
	})

	for range 2 {
		release, pkg, err := manager.preparePackage(context.Background())
		if err != nil {
			t.Fatalf("preparePackage() error = %v", err)
		}
		if release != wantRelease || pkg.Root != wantPackage.Root {
			t.Fatalf("preparePackage() = %+v, %+v; want %+v, %+v", release, pkg, wantRelease, wantPackage)
		}
	}
	if calls != 1 {
		t.Fatalf("PreparePackage calls = %d, want 1", calls)
	}
}
