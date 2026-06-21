package configs

import "ginp-api/pkg/cfg"

// Skillbox 全局配置变量
//
// 命名空间统一在 yaml 里挂 `skillbox.*`,避免与现有 system/server/db 字段冲突。
// 见 docs/project/需求规划.md 第 6 节。
var Skillbox = new(SkillboxConfig)

// SkillboxConfig Skill Box 自身的运行时配置。
type SkillboxConfig struct {
	// StoreRoot canonical skill 物理存储根目录。
	//   global: <StoreRoot>/global/<name>/<version>/
	//   project: <StoreRoot>/project/<project_id>/<name>/<version>/ (在 skillstore 内部拼装)
	// 留空时由 skillstore 在首次启动时根据 OS 用户目录兜底:
	//   macOS / Linux: ~/.skillbox/store
	//   Windows:      %USERPROFILE%\.skillbox\store
	StoreRoot string `default:"" configkey:"skillbox.store_root"`

	// ToolPaths 各目标工具的 skill 目录覆盖,key = tool id,value = 绝对路径。
	// 留空时由对应 adapter 的 DiscoverPaths() 给出默认值。
	// 例:
	//   tool_paths:
	//     codex: /Users/xxx/.codex/skills
	//     claude: /Users/xxx/.claude/skills
	ToolPaths map[string]string `default:"" configkey:"skillbox.tool_paths"`

	// DefaultScope 新建 skill 时默认落到的作用域,`global` 或 `project`。
	DefaultScope string `default:"global" configkey:"skillbox.default_scope"`

	// AutoBackup 在打 tag / 回滚等操作前是否自动打一个隐式 tag。
	AutoBackup bool `default:"true" configkey:"skillbox.auto_backup"`

	// PresetSkillsDir 预置 skill 库目录(首次安装时扫描导入)。
	// 留空表示不预置。
	PresetSkillsDir string `default:"" configkey:"skillbox.preset_skills_dir"`
}

func init() {
	cfg.ParseConfigStruct(Skillbox)
}
