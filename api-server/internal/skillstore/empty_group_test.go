package skillstore

import (
	"os"
	"path/filepath"
	"testing"

	"ginp-api/internal/skilladapter"
)

// TestEmptyGroupVisibleAfterMove 2026-07-08 增:回归测试 — 移走 skill 后空
// group 必须保留,不能被隐式清理。
// 根因:MoveGroupPath 在 451-452 行调 removeIfEmpty(srcParent),把变空的源
// group 目录删了,ListTree 看不到 → 首页树把空 group 隐藏。修法:移除该调用。
func TestEmptyGroupVisibleAfterMove(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "skillstore-empty-group-*")
	defer os.RemoveAll(tmpDir)
	store, _ := NewAt(tmpDir)

	store.CreateGroupDir("a")
	store.Save(skilladapter.Canonical{
		Manifest: skilladapter.Manifest{Name: "x", Version: "0.1.0", GroupPath: "a"},
		Files:    []skilladapter.File{{Path: "SKILL.md", Content: "---\nname: x\nversion: 0.1.0\n---\n\nbody\n"}},
	})

	// 移出唯一 skill 到根
	if err := store.MoveGroupPath("a", "x", ""); err != nil {
		t.Fatalf("MoveGroupPath: %v", err)
	}

	// 1) 磁盘上空 group 目录必须保留
	if _, err := os.Stat(filepath.Join(tmpDir, "a")); err != nil {
		t.Fatalf("空 group 目录 a/ 应保留,实际: %v", err)
	}
	// 2) 移出的 skill 也在根
	if _, err := os.Stat(filepath.Join(tmpDir, "x", "SKILL.md")); err != nil {
		t.Fatalf("移出后的 x/SKILL.md 应存在: %v", err)
	}

	// 3) ListTree 应返回 2 个根节点: 空 group a + skill x
	tree, _ := store.ListTree("")
	if len(tree) != 2 {
		t.Fatalf("ListTree 应返回 2 个根节点,实际 %d: %+v", len(tree), tree)
	}
	var groupA *TreeNode
	for i := range tree {
		if tree[i].Name == "a" {
			groupA = &tree[i]
		}
	}
	if groupA == nil {
		t.Fatalf("ListTree 缺少 group a: %+v", tree)
	}
	if !groupA.IsGroup {
		t.Fatalf("a 节点应是 group,实际 isGroup=false")
	}
	if len(groupA.Children) != 0 {
		t.Fatalf("a 应为空 group,实际 children=%+v", groupA.Children)
	}

	// 4) 把 x 移回 a,a 应"复活"成非空 group
	if err := store.MoveGroupPath("", "x", "a"); err != nil {
		t.Fatalf("MoveGroupPath back: %v", err)
	}
	tree2, _ := store.ListTree("")
	if len(tree2) != 1 || !tree2[0].IsGroup || tree2[0].Name != "a" {
		t.Fatalf("x 移回 a 后,ListTree 应只含 group a,实际 %+v", tree2)
	}
	if len(tree2[0].Children) != 1 {
		t.Fatalf("a 应含 1 个 child x,实际 %+v", tree2[0].Children)
	}
}
