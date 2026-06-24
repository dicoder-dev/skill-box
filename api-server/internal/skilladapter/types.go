// Package skilladapter 定义 canonical skill 表达 + 目标工具适配器抽象。
//
// 内部所有模块(skillstore / skillapp / skillimporter ...)只依赖 canonical 与
// Adapter interface;具体工具(Codex / Claude / OpenCode / Cursor / Trae)由
// 子包 adapter 实现,运行时通过 registry 调度。
//
// 设计要点见 docs/project/需求规划.md 第 7.1 节。
package skilladapter

// Manifest canonical skill 的元数据。
//
// 字段与 skill.yaml 1:1 对应,见 需求规划.md 第 8 节。
type Manifest struct {
	Name        string   `yaml:"name" json:"name"`
	Version     string   `yaml:"version" json:"version"`
	Description string   `yaml:"description" json:"description"`
	Triggers    []string `yaml:"triggers" json:"triggers"`
	Author      string   `yaml:"author,omitempty" json:"author,omitempty"`
	License     string   `yaml:"license,omitempty" json:"license,omitempty"`
	DependsOn   []string `yaml:"depends_on,omitempty" json:"depends_on,omitempty"`
	TargetTools []string `yaml:"target_tools,omitempty" json:"target_tools,omitempty"`
	// Source/SourceRef 用于标记"这条 skill 是从哪儿来的"(2026-06-24:从 mskill 行迁到 frontmatter)。
	// 不参与 SKILL.md 文件落盘(写盘时通过 skillstore.SkillboxSection 单独处理,避免污染 frontmatter)。
	Source    string `yaml:"-" json:"source,omitempty"`
	SourceRef string `yaml:"-" json:"source_ref,omitempty"`
}

// File canonical skill 的一个文件。
//
// Path 相对 skill 根,如 `SKILL.md` / `examples/review.sh`。
type File struct {
	Path    string `yaml:"path" json:"path"`
	Content string `yaml:"content" json:"content"`
}

// Canonical 与具体工具无关的 skill 表示。
type Canonical struct {
	Manifest Manifest `yaml:"manifest" json:"manifest"`
	Files    []File   `yaml:"files" json:"files"`
	// SourceDir 是 adapter 在本地磁盘上找到该 skill 的绝对路径(读 SKILL.md 的目录)。
	// 用于在 importer.Scan 里产出 FoundSkill.SourcePath;不参与序列化导出。
	SourceDir string `yaml:"-" json:"-"`
}

// Scope 作用域。
const (
	ScopeGlobal  = "global"
	ScopeProject = "project"
)

// ToolID 已支持的目标工具 ID 集合(与 skillbox 内部 storage.tool 字段对应)。
const (
	ToolCodex    = "codex"
	ToolClaude   = "claude"
	ToolOpenCode = "opencode"
	ToolCursor   = "cursor"
	ToolTrae     = "trae"
)

// AllTools v1 支持的全部工具 ID。
var AllTools = []string{ToolCodex, ToolClaude, ToolOpenCode, ToolCursor, ToolTrae}

// Adapter canonical 与目标工具双向转换的接口。
//
// 实现方约束:
//   - Apply 必须是原子的(失败时不应留半成品);caller 负责整组事务的 rollback。
//   - Scan 必须容错:遇到损坏文件跳过而不是整体失败,返回的 error 仅表示整体不可恢复。
type Adapter interface {
	// ToolID 工具唯一 ID,用于路由 / 落库。
	ToolID() string

	// DisplayName UI 显示名(中英混合,前端走 i18n 覆盖)。
	DisplayName() string

	// Icon 前端展示用的图标(emoji / unicode 都行,前端自由映射)。
	Icon() string

	// DiscoverPaths 返回该工具在指定 scope 下的全部 skill 目录。
	// 返回空切片表示该 scope 不支持(如某些工具没有项目级 skill 概念)。
	DiscoverPaths(scope string) ([]string, error)

	// Scan 扫描指定目录,产出 canonical。
	// 目录不存在 / 不可读时返回 ([]Canonical{}, nil),不要把不存在当 error。
	Scan(dir string) ([]Canonical, error)

	// Apply 把 canonical 落到 targetDir(覆盖式)。
	// targetDir 必须已存在;adapter 只负责写入与覆盖目标文件。
	// 实现需要处理 LocalName / 字段裁剪 / 文件名 normalize。
	Apply(c Canonical, targetDir string) error

	// LocalName canonical name 映射到目标工具的最终文件名 / 目录名。
	LocalName(c Canonical) string

	// Validate Apply 前的轻量校验(目录可写 / 字段合法)。
	// 字段合规已在 skillstore.Save 阶段校验过,这里只做工具特有的检查。
	Validate(c Canonical) error

	// IsSystemPath 判断给定扫描根是否属于该 adapter 的 system 级别。
	// system skill 是工具自带 / vendor curated / plugin 内建的那批,
	// 前端 phase2 据此把它们列为只读参考、不可勾选。
	// 不实现则默认全 user(BaseAdapter 已实现,空 SystemPaths 时返回 false)。
	IsSystemPath(p string) bool
}
