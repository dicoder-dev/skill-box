package skilladapter

import (
	"os"
	"path/filepath"
	"testing"
)

// writeSKILLDir 在 dir 下写一个最小 SKILL.md,让该目录可被识别为 skill 根。
func writeSKILLDir(t *testing.T, dir, name, desc string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := "---\nname: " + name + "\ndescription: " + desc + "\n---\n# " + name + "\nbody\n"
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestBaseAdapterScan_FindsHiddenDir 验证能发现 . 开头目录里的 skill(2026-06-23 修复)。
// 之前 BaseAdapter 主动跳过 .system / .curated,导致 Codex 的内置 + curated skill 全部漏掉。
func TestBaseAdapterScan_FindsHiddenDir(t *testing.T) {
	root := t.TempDir()
	writeSKILLDir(t, filepath.Join(root, ".system"), "sys-skill", "system skill")
	writeSKILLDir(t, filepath.Join(root, ".curated", "a"), "cur-a", "curated a")
	writeSKILLDir(t, filepath.Join(root, ".curated", "b"), "cur-b", "curated b")

	cs, err := (&BaseAdapter{}).Scan(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(cs) != 3 {
		names := make([]string, 0, len(cs))
		for _, c := range cs {
			names = append(names, c.Manifest.Name)
		}
		t.Fatalf("expected 3 skills, got %d: %v", len(cs), names)
	}
}

// TestBaseAdapterScan_FindsNestedDir 验证能递归发现多层目录里的 skill(2026-06-23 修复)。
// Claude marketplaces 是 4 层深:marketplaces/<m>/plugins/<p>/skills/<n>。
func TestBaseAdapterScan_FindsNestedDir(t *testing.T) {
	root := t.TempDir()
	writeSKILLDir(t, filepath.Join(root, "marketA", "plugins", "pluginA", "skills", "skill-nested"), "nested", "nested skill")
	writeSKILLDir(t, filepath.Join(root, "marketA", "plugins", "pluginA", "skills", "skill-nested2"), "nested2", "nested skill 2")

	cs, err := (&BaseAdapter{}).Scan(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(cs) != 2 {
		t.Fatalf("expected 2 nested skills, got %d", len(cs))
	}
}

// TestBaseAdapterScan_FollowsSymlink 验证能跟随 symlink 到真实 skill 目录(2026-06-23 修复)。
// Trae 的所有 skill 都是 symlink → ../../.agents/skills/xxx。
func TestBaseAdapterScan_FollowsSymlink(t *testing.T) {
	root := t.TempDir()
	realDir := filepath.Join(root, "real-store", "my-skill")
	writeSKILLDir(t, realDir, "my-skill", "my real skill")

	// 在另一处建 symlink 指向真实 skill 目录
	linkRoot := filepath.Join(root, "tool", "skills")
	if err := os.MkdirAll(linkRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(realDir, filepath.Join(linkRoot, "my-skill")); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}

	cs, err := (&BaseAdapter{ID: "x"}).Scan(linkRoot)
	if err != nil {
		t.Fatal(err)
	}
	if len(cs) != 1 || cs[0].Manifest.Name != "my-skill" {
		t.Fatalf("expected symlinked skill, got %+v", cs)
	}
}

// TestBaseAdapterScan_RespectsMaxDepth 验证超过 maxScanDepth 后停止递归,防止 symlink 环死循环。
func TestBaseAdapterScan_RespectsMaxDepth(t *testing.T) {
	root := t.TempDir()
	// 把 skill 放在 maxScanDepth 之外,确认不会发现
	deep := root
	for i := 0; i < maxScanDepth+3; i++ {
		deep = filepath.Join(deep, "lvl")
	}
	writeSKILLDir(t, deep, "too-deep", "too deep")

	cs, err := (&BaseAdapter{}).Scan(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(cs) != 0 {
		t.Fatalf("expected 0 (over maxScanDepth), got %d", len(cs))
	}
}

// TestBaseAdapterScan_SkipsMetadataFiles 验证 .DS_Store / .marker 等元数据被跳过。
func TestBaseAdapterScan_SkipsMetadataFiles(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".DS_Store"), []byte("mac"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".codex-system-skills.marker"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	writeSKILLDir(t, filepath.Join(root, "real-skill"), "real-skill", "real")

	cs, err := (&BaseAdapter{}).Scan(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(cs) != 1 || cs[0].Manifest.Name != "real-skill" {
		t.Fatalf("expected 1 real skill, got %+v", cs)
	}
}
