// Package toolseed 在程序首次启动时把内置的 9 个默认 AI 编程工具 seed 到 e_tool + e_tool_path 表。
//
// 触发条件:启动期 e_tool 表 COUNT==0(全新 DB / 工具表被清空)。
// 不触发:DB 里已有任何 tool 记录(不论系统 / 用户),认作"已初始化过"。
//
// 2026-06-30 二改:此包替代 toolspecs/specs/*.yaml 的"硬编码默认工具"职责,
// 从"编译期内嵌配置"变成"运行时一次性 seed 进 DB"。
// 之后增删 / 改工具全部走 stool 服务层 + e_tool / e_tool_path 表,不再改代码。
package toolseed

import "ginp-api/internal/skilladapter"

// builtinPath 内部用的临时 path 描述,seed 完就丢。
type builtinPath struct {
	Scope     string
	Category  string
	Path      string
	PathOrder int
}

// builtinTool 内置工具 seed 描述;转 entity.Tool / entity.ToolPath 后落库。
type builtinTool struct {
	ToolID      string
	DisplayName string
	MdiIcon     string
	// IconFile seed 阶段同时写到 ~/.skill-box/tool-icons/<IconFile>,让前端能立即拿到。
	// 留空 = 仅 mdi_icon。
	IconFile    string
	Maturity    string
	Note        string
	SortOrder   int
	Paths       []builtinPath
}

// builtins 9 个默认 AI 编程工具(顺序:稳定老工具在前,新实验工具在后)。
//
// 来源:2026-06-30 第一波 toolspecs/specs/*.yaml 9 个文件同等内容,迁移到
// Go 常量。每条注释保留原 yaml 文件头里的"为什么这样配"信息,方便后续
// 维护时直接对照理解。
//
// 2026-07-02:加 IconFile 字段,指向 builtin-icons/ 下的真实图标文件;
// seed 时同步写到 ~/.skill-box/tool-icons/ 让前端能立即显示真实 logo。
var builtins = []builtinTool{
	// ── 5 个老工具(2026-06 之前就有,稳定)────────────────────────────
	{
		ToolID:      "claude",
		DisplayName: "Claude Code",
		MdiIcon:     "mdi:robot-outline",
		IconFile:    "claude.ico",
		Maturity:    "stable",
		SortOrder:   10,
		Note:        "Anthropic 推行的 Agent Skills 标准;写盘根 ~/.agents/skills(共享标准),项目级 .claude/skills。",
		Paths: []builtinPath{
			{skilladapter.ScopeGlobal, "user", "~/.agents/skills", 0},
			{skilladapter.ScopeGlobal, "system", "~/.claude/plugins/marketplaces", 0},
			{skilladapter.ScopeProject, "user", ".claude/skills", 0},
		},
	},
	{
		ToolID:      "codex",
		DisplayName: "Codex",
		MdiIcon:     "mdi:console",
		IconFile:    "codex.png",
		Maturity:    "stable",
		SortOrder:   20,
		Note:        "OpenAI Codex;写盘根 ~/.agents/skills(共享标准),系统级含 .system + vendor_imports/.curated。",
		Paths: []builtinPath{
			{skilladapter.ScopeGlobal, "user", "~/.agents/skills", 0},
			{skilladapter.ScopeGlobal, "system", "~/.codex/skills/.system", 0},
			{skilladapter.ScopeGlobal, "system", "~/.codex/vendor_imports/skills/skills/.curated", 1},
			{skilladapter.ScopeProject, "user", ".agents/skills", 0},
		},
	},
	{
		ToolID:      "cursor",
		DisplayName: "Cursor",
		MdiIcon:     "mdi:cursor-default-click-outline",
		IconFile:    "cursor.png",
		Maturity:    "stable",
		SortOrder:   30,
		Note:        "Cursor 走自己的 ~/.cursor/skills,不走 Agent Skills 标准。",
		Paths: []builtinPath{
			{skilladapter.ScopeGlobal, "user", "~/.cursor/skills", 0},
			{skilladapter.ScopeProject, "user", ".cursor/skills", 0},
		},
	},
	{
		ToolID:      "opencode",
		DisplayName: "OpenCode",
		MdiIcon:     "mdi:code-tags",
		IconFile:    "opencode.png",
		Maturity:    "stable",
		SortOrder:   40,
		Note:        "OpenCode 走自己的 ~/.config/opencode/skills。",
		Paths: []builtinPath{
			{skilladapter.ScopeGlobal, "user", "~/.config/opencode/skills", 0},
			{skilladapter.ScopeProject, "user", ".opencode/skills", 0},
		},
	},
	{
		ToolID:      "trae",
		DisplayName: "Trae",
		MdiIcon:     "mdi:leaf",
		IconFile:    "trae.png",
		Maturity:    "stable",
		SortOrder:   50,
		Note:        "Trae 全局入口 ~/.trae/skills 实际是 symlink,写盘走 ~/.agents/skills(共享标准);项目级 .trae/skills。",
		Paths: []builtinPath{
			{skilladapter.ScopeGlobal, "user", "~/.agents/skills", 0},
			{skilladapter.ScopeProject, "user", ".trae/skills", 0},
		},
	},

	// ── 4 个新工具(2026-06-30 新增)──────────────────────────────────
	{
		ToolID:      "antigravity",
		DisplayName: "Antigravity",
		MdiIcon:     "mdi:rocket-launch-outline",
		IconFile:    "antigravity.png",
		Maturity:    "stable",
		SortOrder:   60,
		Note:        "Google Antigravity IDE(Gemini 3 一同发布);官方标准路径 ~/.gemini/antigravity/skills。",
		Paths: []builtinPath{
			{skilladapter.ScopeGlobal, "user", "~/.gemini/antigravity/skills", 0},
			{skilladapter.ScopeProject, "user", ".gemini/antigravity/skills", 0},
		},
	},
	{
		ToolID:      "cline",
		DisplayName: "Cline",
		MdiIcon:     "mdi:file-document-outline",
		IconFile:    "cline.png",
		Maturity:    "stable",
		SortOrder:   70,
		Note:        "Cline VSCode 插件;同时支持 ~/.agents/skills(共享标准) + ~/.cline/skills(自用目录)。",
		Paths: []builtinPath{
			{skilladapter.ScopeGlobal, "user", "~/.agents/skills", 0},
			{skilladapter.ScopeGlobal, "user", "~/.cline/skills", 1},
			{skilladapter.ScopeProject, "user", ".cline/skills", 0},
		},
	},
	{
		ToolID:      "codebuddy",
		DisplayName: "CodeBuddy",
		MdiIcon:     "mdi:buddy",
		IconFile:    "codebuddy.svg",
		Maturity:    "experimental",
		SortOrder:   80,
		Note:        "腾讯云 CodeBuddy;官方 SKILL.md 规范未公开,路径为占位 ~/.codebuddy/skills,用户实测后可改。",
		Paths: []builtinPath{
			{skilladapter.ScopeGlobal, "user", "~/.codebuddy/skills", 0},
			{skilladapter.ScopeProject, "user", ".codebuddy/skills", 0},
		},
	},
	{
		ToolID:      "jetbrains",
		DisplayName: "JetBrains AI",
		MdiIcon:     "mdi:language-java",
		IconFile:    "jetbrains.ico",
		Maturity:    "experimental",
		SortOrder:   90,
		Note:        "JetBrains AI Assistant;官方 SKILL.md 规范未公开,路径为占位 ~/.jetbrains/skills,用户实测后可改。",
		Paths: []builtinPath{
			{skilladapter.ScopeGlobal, "user", "~/.jetbrains/skills", 0},
			{skilladapter.ScopeProject, "user", ".jetbrains/skills", 0},
		},
	},
}
