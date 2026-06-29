package skillstore

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ginp-api/internal/skilladapter"
)

// TestGroupTreeSmoke 2026-06-29 增:端到端验证分组 / 移动 / 删除链路。
// 覆盖:CreateGroup → ListTree → Save(分组内) → GetByPath → MoveSkill → DeleteByPath → DeleteGroup(cascade)。
func TestGroupTreeSmoke(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "skillstore-group-smoke-*")
	defer os.RemoveAll(tmpDir)
	store, err := NewAt(tmpDir)
	if err != nil {
		t.Fatalf("NewAt: %v", err)
	}

	// 1) 创建分组
	if err := store.CreateGroupDir("frontend/react"); err != nil {
		t.Fatalf("CreateGroupDir: %v", err)
	}

	// 2) 列树:应有 1 个顶层 frontend 分组,内含 react 空分组
	tree, err := store.ListTree("")
	if err != nil {
		t.Fatalf("ListTree: %v", err)
	}
	if len(tree) != 1 || !tree[0].IsGroup || tree[0].Name != "frontend" {
		t.Fatalf("ListTree: expected 1 group 'frontend', got %+v", tree)
	}
	if len(tree[0].Children) != 1 || tree[0].Children[0].Name != "react" {
		t.Fatalf("ListTree: expected react subgroup, got %+v", tree[0].Children)
	}

	// 3) 创建 skill 到分组
	err = store.Save(skilladapter.Canonical{
		Manifest: skilladapter.Manifest{
			Name:        "use-cache",
			Version:     "0.1.0",
			Description: "react hook cache pattern",
			Triggers:    []string{"cache"},
			GroupPath:   "frontend/react",
		},
		Files: []skilladapter.File{{Path: "SKILL.md", Content: "---\nname: use-cache\nversion: 0.1.0\ndescription: react hook cache pattern\ntriggers: [cache]\n---\n\nbody text\n"}},
	})
	if err != nil {
		t.Fatalf("Save (in group): %v", err)
	}

	// 4) LoadByPath 能读到
	c, err := store.LoadByPath("frontend/react", "use-cache")
	if err != nil {
		t.Fatalf("LoadByPath: %v", err)
	}
	if c.Manifest.Name != "use-cache" || c.Manifest.GroupPath != "frontend/react" {
		t.Fatalf("LoadByPath: unexpected canonical = %+v", c.Manifest)
	}

	// 5) ListTree 现在应反映 use-cache 叶子
	tree, _ = store.ListTree("")
	if got := countLeaves(tree); got != 1 {
		t.Fatalf("ListTree after create: expected 1 leaf, got %d", got)
	}

	// 6) MoveSkill 到根
	if err := store.MoveGroupPath("frontend/react", "use-cache", ""); err != nil {
		t.Fatalf("MoveGroupPath: %v", err)
	}
	// 根下应该有 use-cache;frontend/react 应为空
	if !store.ExistsByPath("", "use-cache") {
		t.Fatalf("MoveSkill: use-cache not at root after move")
	}
	if store.ExistsByPath("frontend/react", "use-cache") {
		t.Fatalf("MoveSkill: use-cache still at frontend/react after move")
	}

	// 7) DeleteByPath 删 skill
	if err := store.DeleteByPath("", "use-cache"); err != nil {
		t.Fatalf("DeleteByPath: %v", err)
	}

	// 8) DeleteGroupDir 空 + cascade=false 应成功
	if _, err := store.DeleteGroupDir("frontend/react", false); err != nil {
		t.Fatalf("DeleteGroupDir empty: %v", err)
	}

	// 9) 创建多 skill 在分组下,验证 cascade
	store.CreateGroupDir("backend/go")
	store.Save(skilladapter.Canonical{
		Manifest: skilladapter.Manifest{Name: "skill-a", Version: "0.1.0", Description: "a", Triggers: []string{"a"}, GroupPath: "backend/go"},
		Files:    []skilladapter.File{{Path: "SKILL.md", Content: "---\nname: skill-a\nversion: 0.1.0\ndescription: a\ntriggers: [a]\n---\n\nbody\n"}},
	})
	store.Save(skilladapter.Canonical{
		Manifest: skilladapter.Manifest{Name: "skill-b", Version: "0.1.0", Description: "b", Triggers: []string{"b"}, GroupPath: "backend/go"},
		Files:    []skilladapter.File{{Path: "SKILL.md", Content: "---\nname: skill-b\nversion: 0.1.0\ndescription: b\ntriggers: [b]\n---\n\nbody\n"}},
	})

	// 10a) cascade=false 非空 → 应失败 + 返回 deleted 列表
	deleted, err := store.DeleteGroupDir("backend/go", false)
	if err == nil {
		t.Fatalf("DeleteGroupDir cascade=false 非空 应失败但成功了")
	}
	if len(deleted) != 2 {
		t.Fatalf("DeleteGroupDir 非空: expected 2 deleted paths, got %v", deleted)
	}

	// 10b) cascade=true → 删成功,返回被删 skill 路径
	deleted, err = store.DeleteGroupDir("backend/go", true)
	if err != nil {
		t.Fatalf("DeleteGroupDir cascade=true: %v", err)
	}
	if len(deleted) != 2 {
		t.Fatalf("DeleteGroupDir cascade=true: expected 2 deleted paths, got %v", deleted)
	}
	for _, p := range deleted {
		if !strings.HasPrefix(p, "backend/go/") {
			t.Fatalf("DeleteGroupDir: deleted path %q should start with backend/go/", p)
		}
	}

	// 11) CreateGroupDir 拒绝 .. 路径
	if err := store.CreateGroupDir("../escape"); err == nil {
		t.Fatalf("CreateGroupDir ../escape 应失败但成功了")
	}
	// 12) 拒绝绝对路径前缀
	if err := store.CreateGroupDir("/abs/path"); err == nil {
		t.Fatalf("CreateGroupDir /abs/path 应失败但成功了")
	}
}

// countLeaves 统计树中所有 skill 叶子数量(递归)
func countLeaves(nodes []TreeNode) int {
	n := 0
	for _, node := range nodes {
		if !node.IsGroup {
			n++
		} else {
			n += countLeaves(node.Children)
		}
	}
	return n
}

// TestMoveGroupDir 2026-06-29 增:分组嵌套到另一分组。
func TestMoveGroupDir(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "skillstore-movegroup-*")
	defer os.RemoveAll(tmpDir)
	store, _ := NewAt(tmpDir)

	store.CreateGroupDir("a")
	store.CreateGroupDir("b")
	store.Save(skilladapter.Canonical{
		Manifest: skilladapter.Manifest{Name: "x", Version: "0.1.0", GroupPath: "a"},
		Files:    []skilladapter.File{{Path: "SKILL.md", Content: "---\nname: x\nversion: 0.1.0\n---\n\nbody\n"}},
	})

	// 把分组 a 挪到 b 下 → b/a
	if err := store.MoveGroupDir("a", "b"); err != nil {
		t.Fatalf("MoveGroupDir: %v", err)
	}
	// a 下原本有 skill x(写盘位置是 a/x/SKILL.md);挪到 b/a 后应变成 b/a/x/SKILL.md
	if _, err := os.Stat(filepath.Join(tmpDir, "b", "a", "x", "SKILL.md")); err != nil {
		t.Fatalf("MoveGroupDir: b/a/x/SKILL.md should exist after move: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "a")); !os.IsNotExist(err) {
		t.Fatalf("MoveGroupDir: old 'a' dir should be gone after move")
	}

	// LoadByPath 在新位置仍能读到
	c, err := store.LoadByPath("b/a", "x")
	if err != nil {
		t.Fatalf("LoadByPath after move: %v", err)
	}
	if c.Manifest.Name != "x" {
		t.Fatalf("LoadByPath after move: name mismatch = %s", c.Manifest.Name)
	}

	// 把 b/a 挪到根下(dst = "")
	if err := store.MoveGroupDir("b/a", ""); err != nil {
		t.Fatalf("MoveGroupDir to root: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "a", "x", "SKILL.md")); err != nil {
		t.Fatalf("MoveGroupDir to root: a/x/SKILL.md should exist after move-to-root: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "b")); !os.IsNotExist(err) {
		t.Fatalf("MoveGroupDir to root: old 'b/a' path should be gone")
	}
}

// TestRenameGroupDir 2026-06-29 增:分组重命名(只改最后一段,父路径不变)。
func TestRenameGroupDir(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "skillstore-renamegroup-*")
	defer os.RemoveAll(tmpDir)
	store, _ := NewAt(tmpDir)

	// 准备:frontend/react/use-cache + frontend/react/other
	store.CreateGroupDir("frontend/react")
	store.Save(skilladapter.Canonical{
		Manifest: skilladapter.Manifest{Name: "use-cache", Version: "0.1.0", GroupPath: "frontend/react"},
		Files:    []skilladapter.File{{Path: "SKILL.md", Content: "---\nname: use-cache\nversion: 0.1.0\n---\n\nbody\n"}},
	})
	store.Save(skilladapter.Canonical{
		Manifest: skilladapter.Manifest{Name: "other", Version: "0.1.0", GroupPath: "frontend/react"},
		Files:    []skilladapter.File{{Path: "SKILL.md", Content: "---\nname: other\nversion: 0.1.0\n---\n\nbody\n"}},
	})

	// 1) 改"react"为"react-x" → 整个子树挪过去
	newPath, err := store.RenameGroupDir("frontend/react", "react-x")
	if err != nil {
		t.Fatalf("RenameGroupDir: %v", err)
	}
	if newPath != "frontend/react-x" {
		t.Fatalf("RenameGroupDir: new path = %q, want %q", newPath, "frontend/react-x")
	}
	// 旧位置没了
	if _, err := os.Stat(filepath.Join(tmpDir, "frontend", "react")); !os.IsNotExist(err) {
		t.Fatalf("RenameGroupDir: old path frontend/react should be gone")
	}
	// 子树整组搬过去
	if _, err := os.Stat(filepath.Join(tmpDir, "frontend", "react-x", "use-cache", "SKILL.md")); err != nil {
		t.Fatalf("RenameGroupDir: use-cache should be at new path: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "frontend", "react-x", "other", "SKILL.md")); err != nil {
		t.Fatalf("RenameGroupDir: other should be at new path: %v", err)
	}
	// LoadByPath 走新 path
	if _, err := store.LoadByPath("frontend/react-x", "use-cache"); err != nil {
		t.Fatalf("LoadByPath after rename: %v", err)
	}

	// 2) 同名冲突 → 报错
	store.CreateGroupDir("frontend/existing")
	if _, err := store.RenameGroupDir("frontend/react-x", "existing"); err == nil {
		t.Fatalf("RenameGroupDir: should fail when target name exists")
	}

	// 3) 同名(无变化)→ 幂等返 OK
	idemPath, err := store.RenameGroupDir("frontend/react-x", "react-x")
	if err != nil {
		t.Fatalf("RenameGroupDir: same name should be idempotent: %v", err)
	}
	if idemPath != "frontend/react-x" {
		t.Fatalf("RenameGroupDir: idempotent path mismatch = %q", idemPath)
	}

	// 4) 空 src → 错
	if _, err := store.RenameGroupDir("", "x"); err == nil {
		t.Fatalf("RenameGroupDir: empty src should fail")
	}
	// 5) newName 含 '/' → 错
	if _, err := store.RenameGroupDir("frontend/react-x", "a/b"); err == nil {
		t.Fatalf("RenameGroupDir: newName with / should fail")
	}
	// 6) 源不存在 → ErrNotFound
	if _, err := store.RenameGroupDir("frontend/nonexistent", "x"); err == nil {
		t.Fatalf("RenameGroupDir: nonexistent src should fail")
	}
}

// TestMoveGroupDir_AncestorCheck 2026-06-29 增:防止分组挪到自己的子目录下(死循环防御)。
// 回归测试:之前 copyDirRecursive 漏做"src 在 dst 父级下"循环检测,导致
// 把 aa 挪到 aa/yy/aa 时,os.Rename 失败 → 降级 copy,创建 aa/yy/aa 后
// 立刻发现它落在 src 内,无限递归到 macOS 文件名长度上限才崩。
func TestMoveGroupDir_AncestorCheck(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "skillstore-movegroup-ancestor-*")
	defer os.RemoveAll(tmpDir)
	store, _ := NewAt(tmpDir)

	store.CreateGroupDir("aa")
	store.CreateGroupDir("aa/yy")
	store.Save(skilladapter.Canonical{
		Manifest: skilladapter.Manifest{Name: "x", Version: "0.1.0", GroupPath: "aa"},
		Files:    []skilladapter.File{{Path: "SKILL.md", Content: "---\nname: x\nversion: 0.1.0\n---\n\nbody\n"}},
	})

	// 把 aa 挪到 aa/yy 下 → 目标 = aa/yy/aa,正好在 src(aa) 内,必须拒
	if err := store.MoveGroupDir("aa", "aa/yy"); err == nil {
		t.Fatalf("MoveGroupDir: moving aa into its own descendant aa/yy/aa should be rejected")
	}
	// 源必须原封不动
	if _, err := os.Stat(filepath.Join(tmpDir, "aa", "x", "SKILL.md")); err != nil {
		t.Fatalf("MoveGroupDir: src should be untouched after rejection, got %v", err)
	}
	// 关键防御:不能误创建 aa/yy/aa(死循环的中间产物)
	if _, err := os.Stat(filepath.Join(tmpDir, "aa", "yy", "aa")); err == nil {
		t.Fatalf("MoveGroupDir: target aa/yy/aa should NOT be created (would have been the loop's victim)")
	}
	// 根除:扫一遍 tmpDir,断言树深度不超过初始深度 + 1(否则就是死循环的痕迹)
	if deep := maxDepthUnder(tmpDir); deep > 4 {
		t.Fatalf("MoveGroupDir: tree depth %d after rejection — looks like a runaway loop created deep paths", deep)
	}
}

// TestMoveGroupPath_AncestorCheck 2026-06-29 增:防止把 skill 挪到包含自己父级的位置。
// 同 MoveGroupDir_AncestorCheck,加严检查:之前只测 SKILL.md 是否还在,会假 pass
// (死循环产生了大量中间目录,但源未被覆盖)。现在断言目标父目录 aa/x/inner 不能
// 被创建 + 整树深度不超过 4 层。
func TestMoveGroupPath_AncestorCheck(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "skillstore-moveskill-ancestor-*")
	defer os.RemoveAll(tmpDir)
	store, _ := NewAt(tmpDir)

	store.CreateGroupDir("aa")
	store.Save(skilladapter.Canonical{
		Manifest: skilladapter.Manifest{Name: "x", Version: "0.1.0", GroupPath: "aa"},
		Files:    []skilladapter.File{{Path: "SKILL.md", Content: "---\nname: x\nversion: 0.1.0\n---\n\nbody\n"}},
	})

	// 把 aa/x 挪到 aa/x/inner 下 → 目标 = aa/x/inner/x,正好在 src(aa/x) 内,必须拒
	if err := store.MoveGroupPath("aa", "x", "aa/x/inner"); err == nil {
		t.Fatalf("MoveGroupPath: moving x into aa/x/inner/x should be rejected")
	}
	// 源 SKILL.md 必须还在
	if _, err := os.Stat(filepath.Join(tmpDir, "aa", "x", "SKILL.md")); err != nil {
		t.Fatalf("MoveGroupPath: src should be untouched after rejection, got %v", err)
	}
	// 关键防御:不能误创建 aa/x/inner(死循环的中间产物)
	if _, err := os.Stat(filepath.Join(tmpDir, "aa", "x", "inner")); err == nil {
		t.Fatalf("MoveGroupPath: target parent aa/x/inner should NOT be created (would have been the loop's victim)")
	}
	// 根除:扫一遍 tmpDir,断言树深度不超过 4(否则就是死循环的痕迹)
	if deep := maxDepthUnder(tmpDir); deep > 4 {
		t.Fatalf("MoveGroupPath: tree depth %d after rejection — looks like a runaway loop created deep paths", deep)
	}
}

// maxDepthUnder 算 root 下最深路径的段数(用于检测死循环产生的深层嵌套)。
// 初始 tmpDir 下结构是 <tmp>/aa/x/SKILL.md,最深 3 段;死循环产物至少 6 段+。
func maxDepthUnder(root string) int {
	maxD := 0
	_ = filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(root, p)
		if rel == "." {
			return nil
		}
		d := strings.Count(rel, string(filepath.Separator)) + 1
		if d > maxD {
			maxD = d
		}
		return nil
	})
	return maxD
}
