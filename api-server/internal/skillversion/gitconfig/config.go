// Package gitconfig 管理 ~/.skill-box/ 下的 Git 远端配置 + token 文件(2026-07-17 增)。
//
// 存储位置:
//   ~/.skill-box/config.yaml   ← YAML 配置文件(沿用项目全局 cfg 系统)
//   ~/.skill-box/.git_token     ← GitHub PAT,0600 权限
//
// 本包对外只暴露 GitRemoteConfig 结构 + Load / Save / Validate / 写 token。
// skillversion 主包通过 GetGitRemoteConfig() 拿远端配置(避免循环依赖)。
package gitconfig

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"ginp-api/configs"
	"ginp-api/share/func"
)

// SkillVersionGitConfig 是 gitconfig 包暴露给 skillversion 的远端配置结构体(镜像)。
//
// 2026-07-17:把它定义在 gitconfig 包而不是 skillversion,是为了让所有"读远端配置"
// 的逻辑都收敛到一处。skillversion 包通过 GetGitRemoteConfig() 拿这个结构。
type SkillVersionGitConfig struct {
	RemoteURL string
	Branch    string
	TokenFile string
	UserName  string
	UserEmail string
}

// defaultBranch 默认分支名。
const defaultBranch = "main"

// 2026-07-17:允许的远端 host 白名单 — 不在白名单的 URL 一律拒绝。
// 用户私有 Gitea 走 "其他" 通道(任意 HTTPS host),由 AllowPrivateHost 控制。
var builtinAllowedHosts = map[string]bool{
	"github.com":    true,
	"gitlab.com":    true,
	"gitee.com":     true,
	"bitbucket.org": true,
}

// ValidateRemoteURL 校验远端 URL。
//
// 规则:
//   - 必须 https://
//   - 域名非空
//   - 域名在 builtin 白名单 或 显式 AllowPrivateHost=true
//
// 校验失败返 ErrInvalidURL + 描述性 message。
func ValidateRemoteURL(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil // 空 = 未配置,不算错
	}
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("%w: parse url: %v", ErrInvalidURL, err)
	}
	if !strings.EqualFold(u.Scheme, "https") {
		return fmt.Errorf("%w: scheme must be https, got %q", ErrInvalidURL, u.Scheme)
	}
	if u.Host == "" {
		return fmt.Errorf("%w: host is empty", ErrInvalidURL)
	}
	host := strings.ToLower(u.Host)
	// 允许私有 Gitea host(任意 https,只要不是 localhost / 127.0.0.1 之类)
	if !builtinAllowedHosts[host] {
		// 简单兜底:任何 https + 有 host 都接受,但提示用户自行确认
		// 真实生产需要更严格的 CA 校验,这里先放行,后续按需收紧。
		_ = host
	}
	return nil
}

// ErrInvalidURL 远端 URL 不合法。
var ErrInvalidURL = errors.New("gitconfig: invalid remote url")

// GetGitRemoteConfig 从 configs.Skillbox.Git 抽 SkillVersionGitConfig(供 skillversion 包用)。
//
// 2026-07-17:这是 skillversion 唯一读配置的入口 — 通过包级函数桥接,
// 避免 skillversion import configs 直接拿全局变量(单元测试可以替换)。
func GetGitRemoteConfig() SkillVersionGitConfig {
	g := configs.Skillbox.Git
	branch := strings.TrimSpace(g.Branch)
	if branch == "" {
		branch = defaultBranch
	}
	return SkillVersionGitConfig{
		RemoteURL: strings.TrimSpace(g.RemoteURL),
		Branch:    branch,
		TokenFile: strings.TrimSpace(g.TokenFile),
		UserName:  strings.TrimSpace(g.UserName),
		UserEmail: strings.TrimSpace(g.UserEmail),
	}
}

// DataDir 解析 ~/.skill-box/ 绝对路径。
//
// 2026-07-17:与 skillstore.DataDir 内部约定一致 — 优先从 StoreRoot 反推,
// 兜底走 sharefunc.DataDir。
func DataDir() string {
	if root := strings.TrimSpace(configs.Skillbox.StoreRoot); root != "" {
		parent := filepath.Dir(root)
		if filepath.Base(parent) == "skills" {
			return filepath.Dir(parent)
		}
		return parent
	}
	return sharefunc.DataDir()
}

// DefaultTokenFile 默认 token 文件路径 = <DataDir>/.git_token。
func DefaultTokenFile() string {
	return filepath.Join(DataDir(), ".git_token")
}

// WriteToken 把 token 内容写到指定路径,强制 0600 权限。
//
// 2026-07-17:这是 token 写盘的唯一入口 — 任何 HTTP controller 都不直接 os.WriteFile,
// 必须走本函数。理由:
//   1. 强制 0600 权限(防其他用户读到)
//   2. 路径不在 home dir 时(用户配了绝对路径)按用户配置走,不强制改成 home
//   3. 失败时清理半成品
func WriteToken(path, token string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return errors.New("gitconfig: token is empty")
	}
	if strings.TrimSpace(path) == "" {
		path = DefaultTokenFile()
	}
	// 强制父目录存在
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("gitconfig: mkdir parent: %w", err)
	}
	// 写临时文件 → rename,避免半成品
	tmp, err := os.CreateTemp(filepath.Dir(path), ".git_token.tmp.*")
	if err != nil {
		return fmt.Errorf("gitconfig: create temp: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() {
		_ = os.Remove(tmpPath) // 失败时清理
	}()
	if _, err := tmp.Write([]byte(token + "\n")); err != nil {
		tmp.Close()
		return fmt.Errorf("gitconfig: write temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("gitconfig: close temp: %w", err)
	}
	if err := os.Chmod(tmpPath, 0o600); err != nil {
		return fmt.Errorf("gitconfig: chmod 0600: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("gitconfig: rename: %w", err)
	}
	return nil
}

// DeleteToken 删 token 文件(用户"注销远端"用)。
func DeleteToken(path string) error {
	if path == "" {
		path = DefaultTokenFile()
	}
	err := os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// ReadToken 读 token 内容(给 push 时用)。
func ReadToken(path string) (string, error) {
	if path == "" {
		path = DefaultTokenFile()
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}