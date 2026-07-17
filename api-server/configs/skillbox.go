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

	// Git 远端同步配置(2026-07-17 增:go-git 版本管理改造)。
	// 全部字段可空;为空时 skillversion.InitIfNotExists 仍会 PlainInit 本地仓库,
	// 但不创建 remote,所有 push/pull 走 no-op + 返回明确错误。
	//
	// 示例 yaml:
	//   skillbox:
	//     git:
	//       remote_url: https://github.com/alice/my-skills.git
	//       branch: main
	//       token_file: ~/.skill-box/.git_token   # 0600 权限,存 GitHub PAT
	//       user_name: alice                       # 可选;留空用环境变量 / 占位
	//       user_email: alice@example.com          # 可选
	Git GitConfig `default:"" configkey:"skillbox.git"`

	// AutoCommit 2026-07-18 增:auto-commit message 策略。LLMEnabled 必须是
	// "至少一个 ai_providers 行 enabled 且 api key 非空" — 不满足时本字段
	// 仍可写,实际生成时降级到模板路径(message 仍落盘)并把 Source 标 llm-failed。
	// 前端设置面板根据 GetLLMTest 返回的 Available 决定是否允许勾选。
	AutoCommit AutoCommitConfig `default:"" configkey:"skillbox.auto_commit"`
}

// AutoCommitConfig skillbox.auto_commit.* 配置块。
type AutoCommitConfig struct {
	// LLMEnabled 是否用 LLM 自动识别 commit 信息。
	LLMEnabled bool `default:"false" configkey:"llm_enabled"`

	// Template LLMEnabled=false 或 LLM 降级时使用的固定模板风格。
	// 支持: timestamp / filename / op_files(默认 filename)。
	Template string `default:"filename" configkey:"template"`
}

// GitConfig skillbox.git.* 配置块。
//
// 2026-07-17 增:go-git 远端同步配置。本地仓库 ~/.skill-box/skills/ 不需要
// 任何配置就能 PlainInit,这套只是远端 + 凭证的元数据。
type GitConfig struct {
	// RemoteURL HTTPS 远端 URL(必填项若要 push/pull);不允许 http:// / ssh://。
	RemoteURL string `default:"" configkey:"remote_url"`

	// Branch 远端分支;留空默认 "main"。
	Branch string `default:"main" configkey:"branch"`

	// TokenFile token 文件绝对路径,内容是 GitHub PAT(github_pat_xxx)。
	// 写入时强制 0600 权限,本字段不出现在 HTTP response。
	TokenFile string `default:"" configkey:"token_file"`

	// UserName commit 作者名;留空时尝试 env GIT_AUTHOR_NAME,否则 "skill-box"。
	UserName string `default:"" configkey:"user_name"`

	// UserEmail commit 作者邮箱;留空时尝试 env GIT_AUTHOR_EMAIL,否则 "skill-box@local"。
	UserEmail string `default:"" configkey:"user_email"`
}

func init() {
	cfg.ParseConfigStruct(Skillbox)
}
