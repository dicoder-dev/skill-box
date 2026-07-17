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

// builtins 默认内置 AI 工具(2026-07-18 扩到 17 个)。
//
// 顺序约定:SortOrder 数字越小越靠前,首页按"用户最常用"排在前:
//   - 编程 IDE / 终端 Agent(stable 主流)         10..90
//   - 通用 Agent / 个人助理(experimental 较新)   100..190
//   - 较新或小众实验工具                         200..290
//
// 2026-07-18 大扩:
//   - 重排已有 9 个,SortOrder 拉开间距(10/20/30/... 跳到 30 间隔)便于后续插入
//   - 新增 8 个:openclaw / hermes / copilot / windsurf / aider / roo / continue / goose
//   - 新增项 IconFile 暂留空(走 mdi 兜底),后续单独迭代补官方图标
var builtins = []builtinTool{
	// ── 编程 IDE / 终端 Agent(stable)────────────────────────
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
		SortOrder:   30,
		Note:        "OpenAI Codex;写盘根 ~/.agents/skills(共享标准),系统级仅 ~/.codex/skills/.system 一个。",
		Paths: []builtinPath{
			{skilladapter.ScopeGlobal, "user", "~/.agents/skills", 0},
			{skilladapter.ScopeGlobal, "system", "~/.codex/skills/.system", 0},
			{skilladapter.ScopeProject, "user", ".agents/skills", 0},
		},
	},
	{
		ToolID:      "cursor",
		DisplayName: "Cursor",
		MdiIcon:     "mdi:cursor-default-click-outline",
		IconFile:    "cursor.ico",
		Maturity:    "stable",
		SortOrder:   50,
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
		IconFile:    "opencode.svg",
		Maturity:    "stable",
		SortOrder:   70,
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
		IconFile:    "trae.ico",
		Maturity:    "stable",
		SortOrder:   90,
		Note:        "Trae 全局入口 ~/.trae/skills 实际是 symlink,写盘走 ~/.agents/skills(共享标准);项目级 .trae/skills。",
		Paths: []builtinPath{
			{skilladapter.ScopeGlobal, "user", "~/.agents/skills", 0},
			{skilladapter.ScopeProject, "user", ".trae/skills", 0},
		},
	},
	{
		ToolID:      "antigravity",
		DisplayName: "Antigravity",
		MdiIcon:     "mdi:rocket-launch-outline",
		IconFile:    "antigravity.ico",
		Maturity:    "stable",
		SortOrder:   110,
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
		SortOrder:   130,
		Note:        "Cline VSCode 插件;走 ~/.agents/skills 共享标准(单一全局路径)。",
		Paths: []builtinPath{
			{skilladapter.ScopeGlobal, "user", "~/.agents/skills", 0},
			{skilladapter.ScopeProject, "user", ".cline/skills", 0},
		},
	},

	// ── 通用 Agent / 个人助理(experimental,2026-07-18 新增)────────
	// OpenClaw:2026 年爆火的开源 AI Agent 框架(红色龙虾 logo),
	// GitHub 24.8 万 star,登顶星标榜首;支持多通道接入(WhatsApp/Telegram/飞书/钉钉),
	// 通过 ~/.openclaw/skills 持久化记忆与技能。
	{
		ToolID:      "openclaw",
		DisplayName: "OpenClaw",
		MdiIcon:     "mdi:rabbit-variant",
		IconFile:    "",
		Maturity:    "experimental",
		SortOrder:   200,
		Note:        "2026 爆火开源 Agent 框架(github.com/openclaw/openclaw),多通道接入 + 本地记忆 + 技能生态;官方 SKILL.md 路径未公开,占位 ~/.openclaw/skills。",
		Paths: []builtinPath{
			{skilladapter.ScopeGlobal, "user", "~/.openclaw/skills", 0},
			{skilladapter.ScopeProject, "user", ".openclaw/skills", 0},
		},
	},
	// Hermes Agent:Nous Research 出品的自进化 AI Agent,
	// 核心特性"学习闭环 + 持久记忆",配置目录 ~/.hermes/。
	{
		ToolID:      "hermes",
		DisplayName: "Hermes Agent",
		MdiIcon:     "mdi:brain",
		IconFile:    "",
		Maturity:    "experimental",
		SortOrder:   220,
		Note:        "Nous Research 开源自进化 AI Agent(github.com/NousResearch/hermes-agent),内置学习闭环 + 持久记忆;官方 SKILL.md 路径未公开,占位 ~/.hermes/skills。",
		Paths: []builtinPath{
			{skilladapter.ScopeGlobal, "user", "~/.hermes/skills", 0},
			{skilladapter.ScopeProject, "user", ".hermes/skills", 0},
		},
	},
	// GitHub Copilot:VSCode 老牌 AI 补全 + Chat;Agent Skills 路径暂未官方公开,
	// 推测走 ~/.copilot/skills(参考 VSCode 其他 Copilot 扩展)。
	{
		ToolID:      "copilot",
		DisplayName: "GitHub Copilot",
		MdiIcon:     "mdi:github",
		IconFile:    "copilot.ico",
		Maturity:    "experimental",
		SortOrder:   240,
		Note:        "GitHub Copilot(VSCode/JetBrains);官方 SKILL.md 规范未公开,占位 ~/.copilot/skills,用户实测后可改。",
		Paths: []builtinPath{
			{skilladapter.ScopeGlobal, "user", "~/.copilot/skills", 0},
			{skilladapter.ScopeProject, "user", ".copilot/skills", 0},
		},
	},
	// Windsurf:Codeium 推出的 AI IDE,原 Cascade 引擎;社区讨论支持 Agent Skills 后,
	// 推测走 ~/.codeium/skills(Windsurf 改名前的 codeium 配置根)。
	{
		ToolID:      "windsurf",
		DisplayName: "Windsurf",
		MdiIcon:     "mdi:waves-arrow-up",
		IconFile:    "windsurf.ico",
		Maturity:    "experimental",
		SortOrder:   260,
		Note:        "Codeium 出品的 AI 原生 IDE(原 Cascade);官方 SKILL.md 规范未公开,占位 ~/.codeium/skills,用户实测后可改。",
		Paths: []builtinPath{
			{skilladapter.ScopeGlobal, "user", "~/.codeium/skills", 0},
			{skilladapter.ScopeProject, "user", ".codeium/skills", 0},
		},
	},

	// ── 较新或小众实验工具(experimental,2026-07-18 新增)──────────
	// Aider:终端成对的 AI 编程 pair,git-aware;走 ~/.aider/。
	{
		ToolID:      "aider",
		DisplayName: "Aider",
		MdiIcon:     "mdi:account-multiple-plus-outline",
		IconFile:    "",
		Maturity:    "experimental",
		SortOrder:   280,
		Note:        "终端 AI pair 编程工具(aider.chat),git-aware diff;官方 SKILL.md 规范未公开,占位 ~/.aider/skills。",
		Paths: []builtinPath{
			{skilladapter.ScopeGlobal, "user", "~/.aider/skills", 0},
			{skilladapter.ScopeProject, "user", ".aider/skills", 0},
		},
	},
	// Roo Code:Cline fork,VSCode 插件;多模式 + 自定义角色,~/.roo/skills。
	{
		ToolID:      "roo",
		DisplayName: "Roo Code",
		MdiIcon:     "mdi:rocket",
		IconFile:    "roo.ico",
		Maturity:    "experimental",
		SortOrder:   300,
		Note:        "Cline fork 出的多模式 VSCode AI 插件(RooCodeInc/Roo-Code);支持自定义角色 + 多模式;占位 ~/.roo/skills。",
		Paths: []builtinPath{
			{skilladapter.ScopeGlobal, "user", "~/.roo/skills", 0},
			{skilladapter.ScopeProject, "user", ".roo/skills", 0},
		},
	},
	// Continue:开源 AI 编程助手,VSCode + JetBrains 插件;配置 ~/.continue/config.yaml。
	{
		ToolID:      "continue",
		DisplayName: "Continue",
		MdiIcon:     "mdi:play-box-outline",
		IconFile:    "continue.png",
		Maturity:    "experimental",
		SortOrder:   320,
		Note:        "开源 AI 编程助手(continuedev/continue),VSCode + JetBrains 插件;配置 ~/.continue/;占位 ~/.continue/skills。",
		Paths: []builtinPath{
			{skilladapter.ScopeGlobal, "user", "~/.continue/skills", 0},
			{skilladapter.ScopeProject, "user", ".continue/skills", 0},
		},
	},
	// Goose:Block 公司开源的 AI Agent(2026 已迁到 Linux Foundation AAIF),
	// Rust 写的桌面 + CLI + API;配置 ~/.config/goose/。
	{
		ToolID:      "goose",
		DisplayName: "Goose",
		MdiIcon:     "mdi:duck",
		IconFile:    "goose.ico",
		Maturity:    "experimental",
		SortOrder:   340,
		Note:        "Block 开源 AI Agent(block/goose,已迁 Linux Foundation AAIF),Rust 桌面 + CLI + API;占位 ~/.config/goose/skills。",
		Paths: []builtinPath{
			{skilladapter.ScopeGlobal, "user", "~/.config/goose/skills", 0},
			{skilladapter.ScopeProject, "user", ".goose/skills", 0},
		},
	},

	// ── 早期内置但目前 SKILL.md 规范未公开的两个,继续保留 experimental ──
	{
		ToolID:      "codebuddy",
		DisplayName: "CodeBuddy",
		MdiIcon:     "mdi:buddy",
		IconFile:    "codebuddy.svg",
		Maturity:    "experimental",
		SortOrder:   360,
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
		SortOrder:   380,
		Note:        "JetBrains AI Assistant;官方 SKILL.md 规范未公开,路径为占位 ~/.jetbrains/skills,用户实测后可改。",
		Paths: []builtinPath{
			{skilladapter.ScopeGlobal, "user", "~/.jetbrains/skills", 0},
			{skilladapter.ScopeProject, "user", ".jetbrains/skills", 0},
		},
	},
}
