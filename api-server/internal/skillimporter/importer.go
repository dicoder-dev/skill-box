// Package skillimporter 把已装编程工具的 skill 导入到 Skill Box 自己的 store。
//
// 设计要点(见 docs/project/需求规划.md 第 5.2 节):
//   - 只读扫描:不修改任何目标工具目录(避免破坏现有 skill)
//   - 落地策略:复制 canonical 内容到 skillstore(全局),不动原目录
//   - 幂等:同一 (tool, name, version) 二次导入会覆盖,不会留垃圾
//   - 进度可观察:Report 暴露每工具/每路径命中数与失败原因
package skillimporter

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"ginp-api/internal/gapi/entity"
	mskill "ginp-api/internal/gapi/model/skillbox/mskill"
	"ginp-api/internal/skilladapter"
	"ginp-api/internal/skillstore"

	"gorm.io/gorm"
)

// CategoryFound skill 在原工具里的归类(user = 用户自有 / system = 工具自带或 vendor)。
//
// 前端 Onboarding phase2 据此决定是否可勾选 —— user 默认勾选可取消,
// system 只读列出不可勾(避免把工具内建 skill 误导入覆盖本地 store)。
type Category string

const (
	CategoryUser   Category = "user"
	CategorySystem Category = "system"
)

// FoundSkill 单次扫描中发现的一个 skill。
type FoundSkill struct {
	ToolID     string                 `json:"tool_id"`
	ToolName   string                 `json:"tool_name"`
	SourcePath string                 `json:"source_path"` // 该 skill 在原工具里的绝对路径
	Category   Category               `json:"category"`    // user | system
	Canonical  skilladapter.Canonical `json:"canonical"`
}

// ScannedDir 单个扫描根目录的结果。
type ScannedDir struct {
	ToolID   string   `json:"tool_id"`
	Path     string   `json:"path"`
	Category Category `json:"category"` // user | system
	Exists   bool     `json:"exists"`
	Found    int      `json:"found"`
	Errors   []string `json:"errors,omitempty"`
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
	store   *skillstore.Store
	reg     *skilladapter.Registry // nil = 用默认全局
	dbWrite *gorm.DB               // nil = 跳过 DB 写库(测试 / 纯盘导入场景)
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

// WithDB 让 Importer 在 Import 阶段同步落库。
// 不传则只写盘,跟旧行为一致(便于测试,以及只跑盘导入的脚本场景)。
func (im *Importer) WithDB(db *gorm.DB) *Importer {
	im.dbWrite = db
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
		if len(paths) == 0 {
			// adapter 未声明任何路径(可能没装)。依然在 Tools 列表中占位,
			// 让前端 phase1 的 adapter 状态表能展示;但 phase2(扫描结果)用
			// 后处理过的 foundTools 过滤掉,避免出现"空名字 + 数量 0"的幽灵 tab。
			continue
		}
		for _, p := range paths {
			// 按 root 路径判定 category:adapter 在 BaseAdapter 上声明的 SystemPaths
			// 覆盖该根(或其子路径)则视为 system,否则 user。
			cat := CategoryUser
			if sys, ok := a.(interface{ IsSystemPath(string) bool }); ok && sys.IsSystemPath(p) {
				cat = CategorySystem
			}
			entry := ScannedDir{ToolID: a.ToolID(), Path: p, Category: cat}
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
				// 优先用 adapter 给的真实 skill 目录(SourceDir 由 readSkillDir 写入),
				// 对多层嵌套(claude marketplaces 6 层)更准确;空时回退到
				// scan 根 + LocalName,兼容未来不带 SourceDir 的 adapter。
				src := c.SourceDir
				if src == "" {
					src = filepath.Join(p, a.LocalName(c))
				}
				// ToolName 兜底为 ToolID:极少数 adapter 可能没设 DisplayName,
				// 前端 phase2 渲染 tool tab 时不允许出现空名字。
				tn := a.DisplayName()
				if tn == "" {
					tn = a.ToolID()
				}
				r.FoundSkills = append(r.FoundSkills, FoundSkill{
					ToolID:     a.ToolID(),
					ToolName:   tn,
					SourcePath: src,
					Category:   cat,
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
	// 过滤 Tools:只保留有 found 命中的 toolID,避免 phase2 渲染"空名字 + 数量 0"
	// 的幽灵 tab(典型场景:adapter 目录未找到 / 0 命中)。保持原相对顺序。
	{
		hit := make(map[string]bool, len(r.FoundSkills))
		for _, fs := range r.FoundSkills {
			hit[fs.ToolID] = true
		}
		filtered := r.Tools[:0]
		for _, tid := range r.Tools {
			if hit[tid] {
				filtered = append(filtered, tid)
			}
		}
		r.Tools = filtered
	}
	// 排序规则:user 在前 system 在后;同档内按 toolID + name 字典序。
	// 排序是给前端"按档位分组"的兜底;前端 Onboarding phase2 自己也会再分组渲染。
	sort.Slice(r.FoundSkills, func(i, j int) bool {
		if r.FoundSkills[i].Category != r.FoundSkills[j].Category {
			return r.FoundSkills[i].Category == CategoryUser
		}
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

// NormalizeForStore 在 Save 前修补 Manifest,适配上游工具 SKILL.md frontmatter
// 不全的情况(典型:外部 skill 只有 name + description,没有 triggers;
// 或某些 community skill description 偏短 < 10 字)。
//
// 规则:
//   - Triggers 空:从 Description 头几个有意义的词提取,仍空就放一个 name。
//   - Description 长度 < 10:用 body 第一段非空非标题文字兜底;
//     仍空就用 Name + " skill" 兜底(避免 store 拒收)。
//   - 修补后仍然只补不删,不会动 caller 已经填好的字段。
func NormalizeForStore(c *skilladapter.Canonical) {
	if c == nil {
		return
	}
	body := ""
	if len(c.Files) > 0 {
		body = c.Files[0].Content
	}

	// description 兜底
	if len(c.Manifest.Description) < 10 {
		if s := FirstBodyParagraph(body, 480); len(s) >= 10 {
			c.Manifest.Description = s
		} else {
			c.Manifest.Description = c.Manifest.Name + " skill"
		}
	}

	// triggers 兜底:从 description 抽前几个英文/中文词;仍空就放 name
	if len(c.Manifest.Triggers) == 0 {
		ts := ExtractTriggers(c.Manifest.Description, c.Manifest.Name)
		if len(ts) > 0 {
			c.Manifest.Triggers = ts
		}
	}
}

// FirstBodyParagraph 跳过 frontmatter 与 # 标题,返回 body 第一段非空文字。
// 长度超过 max 自动截断在最近的句号/换行。
func FirstBodyParagraph(md string, max int) string {
	if md == "" {
		return ""
	}
	// 跳过 frontmatter
	if strings.HasPrefix(md, "---") {
		if end := strings.Index(md, "\n---"); end > 0 {
			md = md[end+4:]
		}
	}
	for _, line := range strings.Split(md, "\n") {
		t := strings.TrimSpace(line)
		if t == "" || strings.HasPrefix(t, "#") {
			continue
		}
		// 去掉行内 markdown 标记(链接/粗体/代码)
		t = strings.TrimSpace(strings.TrimLeft(t, "-*>`"))
		if t == "" {
			continue
		}
		if len(t) > max {
			t = t[:max]
			if i := strings.LastIndexAny(t, "。.!?\n"); i > max/2 {
				t = t[:i+1]
			}
		}
		return t
	}
	return ""
}

// extractTriggers 从一段文字里提取 1~10 个 trigger 词。
// 策略:按空格分词,过滤短词(<2)与停用词,去重,补一个 name 兜底。
func ExtractTriggers(desc, name string) []string {
	stops := map[string]bool{
		"the": true, "a": true, "an": true, "is": true, "are": true,
		"and": true, "or": true, "of": true, "to": true, "for": true,
		"in": true, "on": true, "with": true, "this": true, "that": true,
		"be": true, "by": true, "as": true, "at": true, "from": true,
		"的": true, "了": true, "是": true, "在": true, "和": true,
		"与": true, "或": true, "把": true, "用": true, "对": true,
	}
	seen := map[string]bool{}
	var out []string
	add := func(w string) {
		w = strings.ToLower(strings.TrimSpace(w))
		w = strings.Trim(w, ",.;:!?\"'`()[]{}<>")
		if len(w) < 2 || len(w) > 32 {
			return
		}
		if stops[w] {
			return
		}
		if seen[w] {
			return
		}
		seen[w] = true
		out = append(out, w)
	}
	// 用空白 + 非字母数字 拆
	for _, f := range strings.FieldsFunc(desc, func(r rune) bool {
		return !(r == '-' || r == '_' ||
			(r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') ||
			(0x4E00 <= r && r <= 0x9FFF)) // CJK
	}) {
		add(f)
		if len(out) >= 9 {
			break
		}
	}
	if len(out) == 0 && name != "" {
		out = append(out, name)
	}
	if len(out) > 10 {
		out = out[:10]
	}
	return out
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
		// 兜底:外部工具的 SKILL.md frontmatter 可能只有 name+description
		// 没有 triggers / description 过短。store 校验要求严格,直接 Save 会失败。
		// normalizeForStore 自动补全,不影响已经合法的 manifest。
		NormalizeForStore(&c)
		if err := im.store.Save(c, skilladapter.ScopeGlobal, 0); err != nil {
			out = append(out, ImportResult{
				ToolID: it.ToolID, Name: it.Name, Version: ver,
				OK: false, Error: err.Error(),
			})
			continue
		}
		// 双写 DB:store 是 source of truth,DB 落库失败也要让用户看见。
		// 若没传 dbWrite(测试/脚本场景),跳过 —— 物理盘已经成功,不会影响功能。
		// 重复导入同 (scope, name, version) 走 Update 覆盖,跟 store 的覆盖语义一致;
		// 避免触发 idx_skill_scope_proj_name_ver 唯一约束。
		if im.dbWrite != nil {
			if dbErr := im.upsertDBRow(c, fs.ToolID); dbErr != nil {
				out = append(out, ImportResult{
					ToolID: it.ToolID, Name: it.Name, Version: ver,
					OK: false, Error: "db upsert failed: " + dbErr.Error(),
				})
				continue
			}
		}
		out = append(out, ImportResult{
			ToolID: it.ToolID, Name: it.Name, Version: ver, OK: true,
		})
	}
	return out, nil
}

// upsertDBRow 落库一条 mskill 行。
// 已存在则 Update ManifestJSON/Source/SourceRef,不存在则 Create。
// 不抛 panic,所有错误回传;store 已经是 source of truth,DB 失败不影响盘上数据。
func (im *Importer) upsertDBRow(c skilladapter.Canonical, toolID string) error {
	scope := skilladapter.ScopeGlobal
	mj, err := json.Marshal(c.Manifest)
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}
	row := &entity.Skill{
		Scope:        scope,
		ProjectID:    0,
		Name:         c.Manifest.Name,
		Version:      c.Manifest.Version,
		Source:       "imported",
		SourceRef:    toolID,
		ManifestJSON: string(mj),
	}
	// 先按唯一键查,存在就 Update,不存在就 Create。
	var existing entity.Skill
	tx := im.dbWrite.Where(map[string]interface{}{
		mskill.FieldScope:     scope,
		mskill.FieldName:      c.Manifest.Name,
		mskill.FieldProjectID: 0,
		mskill.FieldVersion:   c.Manifest.Version,
	}).First(&existing)
	if tx.Error == nil && existing.ID > 0 {
		// 已存在:刷 manifest + source 标记,不动 id/created_at
		updates := &entity.Skill{
			ManifestJSON: string(mj),
			Source:       "imported",
			SourceRef:    toolID,
		}
		return im.dbWrite.Model(&entity.Skill{}).
			Where(mskill.FieldID+" = ?", existing.ID).
			Updates(updates).Error
	}
	// 兜底:用 model.Create 走标准路径(里面 dbops.Create 会设置 ID)
	return im.dbWrite.Create(row).Error
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
