// 一次性脚本: 走 importer 真导入 trae 5 个 skill 到 ~/.skill-box/skills/
//
// 用法:
//   go run ./cmd/oneoff/import_trae_skills/
//
// 不依赖 HTTP / 桌面端 / DB,直接调 skillimporter 包做 scan + import。
// 幂等: 二次跑同名 skill 会覆盖。
//
// 2026-06-30 改造:不再逐一 import 5 个 adapter 子包,改用 toolspecs 一次注册。
package main

import (
	"fmt"
	"log"
	"os"

	"ginp-api/internal/skilladapter"
	_ "ginp-api/internal/skilladapter/toolspecs"
	"ginp-api/internal/skillimporter"
	"ginp-api/internal/skillstore"
)

func main() {
	store, err := skillstore.New()
	if err != nil {
		log.Fatalf("open store: %v", err)
	}
	fmt.Printf("store root: %s\n", store.Root())

	im := skillimporter.New(store)
	report, err := im.Scan(skilladapter.ScopeGlobal)
	if err != nil {
		log.Fatalf("scan: %v", err)
	}
	fmt.Printf("scan: %d tools, %d skills\n", len(report.Tools), len(report.FoundSkills))

	// 只挑 trae 的(避免覆盖其他 tool 的同名 skill)
	var traeItems []skillimporter.ImportItem
	for _, fs := range report.FoundSkills {
		if fs.ToolID == "trae" {
			traeItems = append(traeItems, skillimporter.ImportItem{
				ToolID:  fs.ToolID,
				Name:    fs.Canonical.Manifest.Name,
				Version: fs.Canonical.Manifest.Version,
			})
			fmt.Printf("  trae: %s v%s <- %s\n", fs.Canonical.Manifest.Name, fs.Canonical.Manifest.Version, fs.SourcePath)
		}
	}
	if len(traeItems) == 0 {
		log.Fatal("no trae skills found in scan")
	}

	// 这里直接传 trae items,不走空=全部,避免把 claude/codex/cursor/opencode 的都导了
	results, err := im.Import(report, traeItems)
	if err != nil {
		log.Fatalf("import: %v", err)
	}
	ok, fail := 0, 0
	for _, r := range results {
		if r.OK {
			ok++
		} else {
			fail++
			fmt.Printf("  FAIL %s/%s v%s: %s\n", r.ToolID, r.Name, r.Version, r.Error)
		}
	}
	fmt.Printf("\nimported: %d ok, %d failed\n", ok, fail)

	// 验证: ls 一下落盘目录
	entries, _ := os.ReadDir(store.Root())
	for _, e := range entries {
		if e.IsDir() {
			fmt.Printf("  store/%s/\n", e.Name())
		}
	}
}
