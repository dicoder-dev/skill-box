package cskillversion

import (
	"os"

	"ginp-api/configs"
)

// 2026-07-17:configs.Skillbox.Git 是值类型,直接字段赋值就能改全局。
// 但 viper 不会同步,所以这里只改内存中的 Go struct,进程退出前不持久化。
// 持久化交给 bootstrap.StartGinLogger 路径上的 cfg.SaveYAML(本期不做)。
//
// 本期做法:这些 setter 只改运行时内存 cfg,期望用户在 Settings UI 上一次性写完整配置;
// 后端不主动 SaveYAML,避免"用户只改了 remote_url 但 token 还是旧的"这种状态漂移。

func setSkillboxGitRemoteURL(v string)    { configs.Skillbox.Git.RemoteURL = v }
func setSkillboxGitBranch(v string)       { configs.Skillbox.Git.Branch = v }
func setSkillboxGitUserName(v string)     { configs.Skillbox.Git.UserName = v }
func setSkillboxGitUserEmail(v string)    { configs.Skillbox.Git.UserEmail = v }
func setSkillboxGitTokenFile(v string)    { configs.Skillbox.Git.TokenFile = v }

// gitconfigHasToken 检查 token 文件存在且非空(给 Status / Config 接口用)。
func gitconfigHasToken(path string) bool {
	fi, err := os.Stat(path)
	if err != nil {
		return false
	}
	return fi.Size() > 0
}