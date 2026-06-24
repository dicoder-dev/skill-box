package conboarding_test

import (
	"os"
	"path/filepath"
	"testing"

	"ginp-api/internal/skilladapter"
	"ginp-api/internal/skillimporter"
	"ginp-api/internal/skillstore"
)

// TestImportFlow_PreservesSkillContent 模拟 PostOnboardingImport 修复后的
// 实现路径:scan 阶段缓存 SourcePath,import 阶段从 SourcePath 重新 ReadSkillDir
// 拿完整 Canonical(含 SKILL.md + 全部附属文件),而不是只塞 Name/Version。
//
// 用 trae 的真实 find-skills 做 fixture(已确认大小 ≈ 4635 字节),验证:
//   1. import 后 store 里的 SKILL.md 字节数 ≈ 源字节数
//   2. 不退化成 ~126 字节的占位货
func TestImportFlow_PreservesSkillContent(t *testing.T) {
	srcPath := "/Users/brody/.trae/skills/find-skills"
	if _, err := os.Stat(srcPath); err != nil {
		t.Skipf("trae 源不可用: %v", err)
	}
	srcBytes, _ := os.ReadFile(filepath.Join(srcPath, "SKILL.md"))
	srcSize := len(srcBytes)
	t.Logf("源 SKILL.md: %d 字节", srcSize)
	if srcSize < 1000 {
		t.Fatalf("源文件太小,跳过测试: %d 字节", srcSize)
	}

	store, err := skillstore.NewAt(filepath.Join(t.TempDir(), "store"))
	if err != nil {
		t.Fatal(err)
	}
	im := skillimporter.New(store)

	// scan 阶段:adapter 读出完整 Canonical
	realPath, _ := filepath.EvalSymlinks(srcPath)
	scannedCanonical, err := skilladapter.ReadSkillDir(realPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(scannedCanonical.Files) < 1 {
		t.Fatal("scan 阶段 Files 为空,无法继续")
	}

	// 模拟 scan 缓存只留轻量字段
	cached := struct {
		ToolID     string
		ToolName   string
		Name       string
		Version    string
		SourcePath string
		Category   string
	}{
		ToolID:     "trae",
		ToolName:   "Trae",
		Name:       scannedCanonical.Manifest.Name,
		Version:    scannedCanonical.Manifest.Version,
		SourcePath: realPath,
		Category:   "user",
	}

	// 修复后的 import 路径:从 SourcePath 重新读
	full, err := skilladapter.ReadSkillDir(cached.SourcePath)
	if err != nil {
		t.Fatal(err)
	}
	if len(full.Files) < 1 {
		t.Fatal("重读后 Files 为空")
	}
	if len(full.Files[0].Content) != srcSize {
		t.Errorf("重读后内容字节数与源不一致: 源=%d, 重读=%d", srcSize, len(full.Files[0].Content))
	}

	// 走 importer.Import
	report := &skillimporter.Report{
		FoundSkills: []skillimporter.FoundSkill{{
			ToolID:     cached.ToolID,
			ToolName:   cached.ToolName,
			SourcePath: cached.SourcePath,
			Canonical:  full,
		}},
	}
	results, err := im.Import(report, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || !results[0].OK {
		t.Fatalf("import failed: %+v", results)
	}

	// 验证落盘内容
	dst := filepath.Join(store.Root(), "find-skills", "SKILL.md")
	dstBytes, _ := os.ReadFile(dst)
	dstSize := len(dstBytes)
	t.Logf("落盘后 SKILL.md: %d 字节", dstSize)

	// 关键断言:落盘后字节数不应被截断到 ~126 字节占位货
	if dstSize < srcSize/2 {
		t.Errorf("落盘后被严重截断: 源=%d, 落盘=%d(可能走了 NormalizeForStore 兜底分支)", srcSize, dstSize)
	}
	if dstSize < 1000 {
		t.Errorf("落盘后 < 1000 字节,符合 bug 现象: 实际 %d 字节", dstSize)
	}
}
