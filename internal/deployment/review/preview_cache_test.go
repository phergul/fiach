package review_test

import (
	"testing"
	"time"

	"github.com/phergul/fiach/internal/deployment"
	"github.com/phergul/fiach/internal/deployment/planner"
	"github.com/phergul/fiach/internal/deployment/review"
)

func TestPreviewCacheEvictsPreviousProfilePreview(t *testing.T) {
	t.Parallel()

	cache := review.NewPreviewCache()
	first := review.CachedPreview{
		PreviewHash: "hash-a",
		ProfileID:   1,
		GameID:      10,
		BuiltAt:     time.Now(),
	}
	second := review.CachedPreview{
		PreviewHash: "hash-b",
		ProfileID:   1,
		GameID:      10,
		BuiltAt:     time.Now(),
	}

	cache.Store(first)
	cache.Store(second)

	if _, found := cache.Get("hash-a"); found {
		t.Fatal("cache.Get(hash-a) found = true, want eviction after profile replacement")
	}

	got, found := cache.Get("hash-b")
	if !found || got.ProfileID != 1 {
		t.Fatalf("cache.Get(hash-b) = %+v found=%t, want second preview", got, found)
	}
}

func TestBuildTreeChildrenLazyLoadsDirectChildren(t *testing.T) {
	t.Parallel()

	plan := planner.FirstApplyPlan{
		Paths: map[string]planner.PathPlan{
			"data/skyui.esp": {
				GameRelativePath: "Data/SkyUI.esp",
				PlannedAction:    planner.ReapplyCreate,
				FileStatus:       deployment.FileStatusAdded,
				RiskLevel:        deployment.RiskNone,
			},
			"data/sub/nested.txt": {
				GameRelativePath: "Data/Sub/nested.txt",
				PlannedAction:    planner.ReapplyReplace,
				FileStatus:       deployment.FileStatusReplaced,
				RiskLevel:        deployment.RiskInfo,
			},
			"root.txt": {
				GameRelativePath: "root.txt",
				PlannedAction:    planner.ReapplyCreate,
				FileStatus:       deployment.FileStatusAdded,
				RiskLevel:        deployment.RiskNone,
			},
		},
	}

	rootNodes := review.BuildTreeChildren(plan, "")
	if len(rootNodes) != 2 {
		t.Fatalf("BuildTreeChildren(root) len = %d, want 2", len(rootNodes))
	}

	dataNode := findTreeNode(rootNodes, "Data")
	if dataNode == nil || !dataNode.IsDirectory || !dataNode.HasChildren || dataNode.ChildCount != 2 {
		t.Fatalf("Data node = %+v, want directory with two direct children", dataNode)
	}
	if dataNode.Status != deployment.FileStatusReplaced {
		t.Fatalf("Data node status = %q, want replaced roll-up", dataNode.Status)
	}

	dataChildren := review.BuildTreeChildren(plan, "Data")
	if len(dataChildren) != 2 {
		t.Fatalf("BuildTreeChildren(Data) len = %d, want 2", len(dataChildren))
	}
}

func TestPreviewHashIsDeterministic(t *testing.T) {
	t.Parallel()

	entry := review.CachedPreview{
		ProfileID: 1,
		GameID:    10,
		Plan: planner.FirstApplyPlan{
			Paths: map[string]planner.PathPlan{
				"data/a.esp": {
					GameRelativePath: "Data/a.esp",
					PlannedAction:    planner.ReapplyCreate,
					FileStatus:       deployment.FileStatusAdded,
				},
			},
		},
		Desired: deployment.DesiredState{
			Files: map[string]deployment.DesiredFile{
				"data/a.esp": {
					GameRelativePath: "Data/a.esp",
					SHA256:           "abc",
				},
			},
		},
	}

	firstHash, err := review.PreviewHash(entry)
	if err != nil {
		t.Fatalf("PreviewHash() error = %v", err)
	}

	secondHash, err := review.PreviewHash(entry)
	if err != nil {
		t.Fatalf("PreviewHash() second error = %v", err)
	}

	if firstHash == "" || firstHash != secondHash {
		t.Fatalf("PreviewHash() = %q and %q, want stable non-empty hash", firstHash, secondHash)
	}
}

func findTreeNode(nodes []review.TreeNode, name string) *review.TreeNode {
	for index := range nodes {
		if nodes[index].Name == name {
			return &nodes[index]
		}
	}
	return nil
}
