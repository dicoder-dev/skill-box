package skilladapter

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestReadSkillDir_SetsSourceDir 验证 ReadSkillDir 成功后 SourceDir 已被赋值为
// 绝对路径(2026-07-03 修复:此前 ReadSkillDir 与 base.go readSkillDir 行为不一致,
// 漏设 SourceDir 导致走 post_onboarding_import 还原流程时 symlink apply 抛
// "empty source_dir" 错误)。
func TestReadSkillDir_SetsSourceDir(t *testing.T) {
	root := t.TempDir()
	skillDir := filepath.Join(root, "unit-test-gen")
	writeSKILLDir(t, skillDir, "unit-test-gen", "测试 SKILL")

	c, err := ReadSkillDir(skillDir)
	if err != nil {
		t.Fatalf("ReadSkillDir: %v", err)
	}
	if c.SourceDir == "" {
		t.Fatalf("SourceDir should not be empty after ReadSkillDir, got %+v", c)
	}
	if !filepath.IsAbs(c.SourceDir) {
		t.Fatalf("SourceDir should be absolute, got %q", c.SourceDir)
	}
	// EvalSymlinks 在真实目录上应等于该目录(无链式 symlink)
	if real, err := filepath.EvalSymlinks(skillDir); err == nil {
		if c.SourceDir != real {
			t.Fatalf("SourceDir = %q, want %q", c.SourceDir, real)
		}
	}
}

// TestReadSkillDir_FollowsSymlink 验证 ReadSkillDir 在传入 symlink 时把 SourceDir
// 解析为真实路径(而不是 symlink 自身),与 base.go readSkillDir 行为一致。
func TestReadSkillDir_FollowsSymlink(t *testing.T) {
	root := t.TempDir()
	realDir := filepath.Join(root, "real", "unit-test-gen")
	writeSKILLDir(t, realDir, "unit-test-gen", "real dir")

	linkDir := filepath.Join(root, "link-root")
	if err := os.MkdirAll(linkDir, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(linkDir, "unit-test-gen")
	if err := os.Symlink(realDir, link); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}

	c, err := ReadSkillDir(link)
	if err != nil {
		t.Fatalf("ReadSkillDir: %v", err)
	}
	if c.SourceDir == link {
		t.Fatalf("SourceDir should be resolved (real path), got symlink itself: %q", c.SourceDir)
	}
	if want, err := filepath.EvalSymlinks(realDir); err == nil {
		if c.SourceDir != want {
			t.Fatalf("SourceDir = %q, want %q", c.SourceDir, want)
		}
	}
}

// TestBaseAdapterApplyLink_EmptySourceDir 验证当 SourceDir 为空时,ApplyLink
// 返回包含 skill 名的明确错误,便于上层定位。
func TestBaseAdapterApplyLink_EmptySourceDir(t *testing.T) {
	ad := &BaseAdapter{ID: "antigravity"}
	targetDir := filepath.Join(t.TempDir(), "unit-test-gen")

	err := ad.ApplyLink(Canonical{
		Manifest: Manifest{Name: "unit-test-gen"},
	}, targetDir)
	if err == nil {
		t.Fatalf("expected error for empty source_dir, got nil")
	}
	if !strings.Contains(err.Error(), "empty source_dir") {
		t.Fatalf("error message should mention 'empty source_dir', got: %v", err)
	}
	if !strings.Contains(err.Error(), "unit-test-gen") {
		t.Fatalf("error message should include skill name, got: %v", err)
	}
	if !strings.Contains(err.Error(), "antigravity") {
		t.Fatalf("error message should include tool id, got: %v", err)
	}
}