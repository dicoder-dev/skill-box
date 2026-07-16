package skillversion

import (
	"ginp-api/internal/skillversion/gitconfig"
)

// gitconfigSnapshot 是 skillversion 包对外的唯一配置入口 — 通过桥接函数
// 从 gitconfig 子包拿 SkillVersionGitConfig,转成本地结构返回。
//
// 2026-07-17:为什么不在 skillversion 包直接 import gitconfig 然后用 gitconfig.SkillVersionGitConfig?
//   - 避免循环依赖(虽然这里没循环,但模式上把"配置读"隔离到独立子包更干净)
//   - 单元测试时可以 mock 这个桥接
func gitconfigSnapshot() gitconfig.SkillVersionGitConfig {
	return gitconfig.GetGitRemoteConfig()
}