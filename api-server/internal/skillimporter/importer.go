// Package skillimporter 把已装编程工具的 skill 导入到 Skill Box 自己的 store。
//
// 设计要点(见 docs/project/需求规划.md 第 5.2 节):
//   - 只读扫描:不修改任何目标工具目录(避免破坏现有 skill)
//   - 落地策略:复制 canonical 内容到 skillstore(全局),不动原目录
//   - 幂等:同一 (tool, name, version) 二次导入会覆盖,不会留垃圾
//   - 进度可观察:Report 暴露每工具/每路径命中数与失败原因
package skillimporter

import (
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"time"

	"ginp-api/internal/skilladapter"
	"ginp-api/internal/skillstore"
)

// FoundSkill 单次扫描中发现的一个 skill。
type FoundSkill struct {
	ToolID     string                 `json:"tool_id"`
	ToolName   string                 `json:"tool_name"`
	SourcePath string                 `json:"source_path"` // 该 skill 在原工具里的绝对路径
	Canonical  skilladapter.Canonical `json:"canonical"`
}

// ScannedDir 单个扫描根目录的结果。
type ScannedDir struct {
	ToolID string   `json:"tool_id"`
	Path   string   `json:"path"`
	Exists bool     `json:"exists"`
	Found  int      `json:"found"`
	Errors []string `json:"errors,omitempty"`
}

// Report 一次完整扫描的产出。
type Report struct {
	StartedAt   time.Time      `json:"started_at"`
	FinishedAt  time.Time      `json:"finished_at"`
	Tools       []string       `json:"tools"`
	Dirs        []ScannedDir   `json:"dirs"`
	FoundSkills []FoundSkill   `json:"found_skills"`
	TotalFound  int            `json:"total_found"`
	TotalDirs   int            `json:"total_dirs"`
	ToolSummary map[string]int `json:"tool_summary"` // toolID -> count
}

// Importer 跨工具的导入器;scan / import 都通过它。
type Importer struct {
	store *skillstore.Store
	reg   *skilladapter.Registry // nil = 用默认全局
}

// New 构造 Importer;store 必传(用于 Import 阶段的物理落地)。
func New(store *skillstore.Store) *Importer {
	return &Importer{store: store}
}

// WithRegistry 让 Importer 使用自定义 registry(测试用);返回 *Importer 便于链式。
func (im *Importer) WithRegistry(reg *skilladapter.Registry) *Importer {
	im.reg = reg
	return im
}

// Scan 扫描默认 registry 中所有 adapter 在指定 scope 下的目录,产出 Report。
// scope 为空时使用 skilladapter.ScopeGlobal。
func (im *Importer) Scan(scope string) (*Report, error) {
	var adapters []skilladapter.Adapter
	if im.reg != nil {
		adapters = im.reg.All()
	} else {
		adapters = skilladapter.All()
	}
	return im.ScanWith(adapters, scope)
}

// ScanWith 用给定的 adapter 列表做扫描(测试用主入口)。
func (im *Importer) ScanWith(adapters []skilladapter.Adapter, scope string) (*Report, error) {
	if im == nil || im.store == nil {
		return nil, errors.New("skillimporter: nil importer/store")
	}
	if scope == "" {
		scope = skilladapter.ScopeGlobal
	}

	started := time.Now()
	r := &Report{
		StartedAt:   started,
		Tools:       []string{},
		Dirs:        []ScannedDir{},
		FoundSkills: []FoundSkill{},
		ToolSummary: map[string]int{},
	}

	for _, a := range adapters {
		r.Tools = append(r.Tools, a.ToolID())
		paths, err := a.DiscoverPaths(scope)
		if err != nil {
			r.Dirs = append(r.Dirs, ScannedDir{ToolID: a.ToolID(), Errors: []string{err.Error()}})
			continue
		}
		for _, p := range paths {
			entry := ScannedDir{ToolID: a.ToolID(), Path: p}
			if _, err := filepath.EvalSymlinks(p); err != nil {
				// 路径不存在或权限不足
				r.Dirs = append(r.Dirs, entry)
				continue
			}
			entry.Exists = true
			cs, scanErr := a.Scan(p)
			if scanErr != nil {
				entry.Errors = append(entry.Errors, scanErr.Error())
			}
			for _, c := range cs {
				r.FoundSkills = append(r.FoundSkills, FoundSkill{
					ToolID:     a.ToolID(),
					ToolName:   a.DisplayName(),
					SourcePath: filepath.Join(p, a.LocalName(c)),
					Canonical:  c,
				})
			}
			entry.Found = len(cs)
			r.ToolSummary[a.ToolID()] += entry.Found
			r.Dirs = append(r.Dirs, entry)
		}
	}

	r.TotalDirs = len(r.Dirs)
	r.TotalFound = len(r.FoundSkills)
	sort.Slice(r.FoundSkills, func(i, j int) bool {
		if r.FoundSkills[i].ToolID != r.FoundSkills[j].ToolID {
			return r.FoundSkills[i].ToolID < r.FoundSkills[j].ToolID
		}
		return r.FoundSkills[i].Canonical.Manifest.Name < r.FoundSkills[j].Canonical.Manifest.Name
	})
	r.FinishedAt = time.Now()
	return r, nil
}

// ImportItem 单条导入请求。
type ImportItem struct {
	ToolID string `json:"tool_id"`
	Name   string `json:"name"`
	// Version 可选,留空用 Report 里找到的 version
	Version string `json:"version,omitempty"`
}

// ImportResult 单条导入结果。
type ImportResult struct {
	ToolID  string `json:"tool_id"`
	Name    string `json:"name"`
	Version string `json:"version"`
	OK      bool   `json:"ok"`
	Error   string `json:"error,omitempty"`
}

// Import 把指定条目从目标工具目录导入到 skillbox 全局 store。
// 入参 items 通常来自前端在 Report 上勾选出来的子集;空 items 表示"全部导入"。
func (im *Importer) Import(report *Report, items []ImportItem) ([]ImportResult, error) {
	if im == nil || im.store == nil {
		return nil, errors.New("skillimporter: nil importer/store")
	}
	if report == nil {
		return nil, errors.New("skillimporter: nil report")
	}

	type key struct {
		ToolID string
		Name   string
	}
	index := make(map[key]FoundSkill, len(report.FoundSkills))
	for _, fs := range report.FoundSkills {
		index[key{fs.ToolID, fs.Canonical.Manifest.Name}] = fs
	}

	if len(items) == 0 {
		for _, fs := range report.FoundSkills {
			items = append(items, ImportItem{
				ToolID:  fs.ToolID,
				Name:    fs.Canonical.Manifest.Name,
				Version: fs.Canonical.Manifest.Version,
			})
		}
	}

	out := make([]ImportResult, 0, len(items))
	for _, it := range items {
		fs, ok := index[key{it.ToolID, it.Name}]
		if !ok {
			out = append(out, ImportResult{
				ToolID: it.ToolID, Name: it.Name,
				OK: false, Error: "not found in last scan",
			})
			continue
		}
		ver := it.Version
		if ver == "" {
			ver = fs.Canonical.Manifest.Version
		}
		c := fs.Canonical
		c.Manifest.Version = ver
		if err := im.store.Save(c, skilladapter.ScopeGlobal, 0); err != nil {
			out = append(out, ImportResult{
				ToolID: it.ToolID, Name: it.Name, Version: ver,
				OK: false, Error: err.Error(),
			})
			continue
		}
		out = append(out, ImportResult{
			ToolID: it.ToolID, Name: it.Name, Version: ver, OK: true,
		})
	}
	return out, nil
}

// FilterByTool 返回 Report 中指定 tool 的 FoundSkill 列表(常用于前端"按工具分组展示")。
func (r *Report) FilterByTool(toolID string) []FoundSkill {
	var out []FoundSkill
	for _, fs := range r.FoundSkills {
		if fs.ToolID == toolID {
			out = append(out, fs)
		}
	}
	return out
}

// String 把 Report 折叠成单行摘要,用于日志。
func (r *Report) String() string {
	if r == nil {
		return "<nil report>"
	}
	return fmt.Sprintf("importer: %d dirs, %d skills across %d tools in %s",
		r.TotalDirs, r.TotalFound, len(r.Tools), r.FinishedAt.Sub(r.StartedAt))
}
