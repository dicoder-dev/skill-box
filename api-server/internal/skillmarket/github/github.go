// Package github 提供 GitHub raw 三方源适配器(2026-07-09 增)。
//
// 与 skillssh 的区别:
//   - skillssh 解析 catalog 站点,RemoteID 是 owner/repo@skill;
//   - github 直接走 GitHub raw URL,RemoteID 是 owner/repo@skill-path(skill-path 含子目录)。
//
// 支持 URL 形态:
//   - https://github.com/{owner}/{repo}/blob/{branch}/{path}/SKILL.md
//   - https://raw.githubusercontent.com/{owner}/{repo}/{branch}/{path}/SKILL.md
//
// 不支持 catalog 浏览 / 搜索(那是 skillssh 的事)。
// 附属文件:GitHub raw 只能一次下一个文件 — 因此只下 SKILL.md;附属文件
// 视为 v1 不支持(后续可扩展 zipball API)。
package github

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"ginp-api/internal/skilladapter"
	"ginp-api/internal/skillmarket"
	"ginp-api/pkg/httpx"
)

const (
	// 2026-07-09:GitHub raw content host,跟 skillssh adapter 共用
	defaultGHRawBase = "https://raw.githubusercontent.com"
	// 2026-07-09:源官方站点(github.com),用于「前往官网」按钮
	defaultSourceHomepage = "https://github.com"
	// 2026-07-09:从已知 skill 仓库列表里能反查 owner/repo → 仓库名
	// 用作 canonical.Manifest.Author 兜底
	defaultAuthor = "GitHub"
)

// Adapter github 适配器。
type Adapter struct {
	httpClient *http.Client
}

// New 构造 Adapter。
func New() *Adapter {
	return &Adapter{httpClient: httpx.NewClient(15 * time.Second)}
}

// NewWithClient 测试用,注入 http.RoundTripper。
func NewWithClient(c *http.Client) *Adapter {
	if c == nil {
		return New()
	}
	return &Adapter{httpClient: c}
}

func (a *Adapter) SourceID() string    { return skillmarket.SourceGitHub }
func (a *Adapter) DisplayName() string { return "GitHub" }
func (a *Adapter) BaseURL() string     { return defaultSourceHomepage }

// HomepageURL 返回源官方首页(github.com)。
func (a *Adapter) HomepageURL(sourceConfigJSON string) string {
	cfg := skillmarket.ParseSourceConfig(sourceConfigJSON)
	if cfg == nil || strings.TrimSpace(cfg.BaseURL) == "" {
		return defaultSourceHomepage
	}
	u, err := url.Parse(strings.TrimSpace(cfg.BaseURL))
	if err != nil || u.Scheme == "" || u.Host == "" {
		return defaultSourceHomepage
	}
	return u.Scheme + "://" + u.Host
}

// KnownFallbackIDs github 无 catalog 兜底列表,直接返空。
func (a *Adapter) KnownFallbackIDs() []string { return nil }

// Discover 返空:github 适配器不参与 catalog 浏览,只支持用户主动粘贴 URL。
// 这样前端如果在 github tab 调用 Discover 不会拿一堆噪音数据。
func (a *Adapter) Discover(ctx context.Context, baseURL, keyword string) ([]skillmarket.MarketItem, error) {
	return nil, nil
}

// Detail 同 Discover:无列表,详情走「直接下载」单文件路径。
func (a *Adapter) Detail(ctx context.Context, baseURL, remoteID string) (*skillmarket.MarketDetail, error) {
	// remoteID = owner/repo@skill-path(路径含子目录,如 "skills/pdf")
	parts := strings.SplitN(remoteID, "@", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("%w: github remoteID must be owner/repo@skill-path, got %q", skillmarket.ErrRemoteNotFound, remoteID)
	}
	repo := parts[0]
	skillPath := parts[1]
	return &skillmarket.MarketDetail{
		MarketItem: skillmarket.MarketItem{
			RemoteID:    remoteID,
			Name:        lastSegment(skillPath),
			Author:      defaultAuthor,
			DetailURL:   fmt.Sprintf("https://github.com/%s/tree/main/%s", repo, skillPath),
			Description: fmt.Sprintf("GitHub raw skill: %s (branch 默认 main)", remoteID),
		},
	}, nil
}

// Download 走 GitHub raw URL 拉 SKILL.md。
//
// 2026-07-09 v1:只下一个 SKILL.md 文件(已知限制,跟 skillssh 行为一致)。
// 后续要支持附属文件,需要调 GitHub zipball API(/repos/{owner}/{repo}/zipball/{branch})。
func (a *Adapter) Download(ctx context.Context, baseURL, remoteID string) (*skilladapter.Canonical, error) {
	repo, skillPath, ok := splitRemoteID(remoteID)
	if !ok {
		return nil, fmt.Errorf("%w: invalid github remote id %q", skillmarket.ErrRemoteNotFound, remoteID)
	}
	// 1) 拼 raw URL — 先试 main,失败试 master
	branches := []string{"main", "master"}
	var lastErr error
	rawBase := defaultGHRawBase
	for _, b := range branches {
		u := fmt.Sprintf("%s/%s/%s/%s/SKILL.md", rawBase, repo, b, skillPath)
		body, err := a.fetchBody(ctx, u)
		if err != nil {
			lastErr = err
			if isRateLimited(err) {
				return nil, fmt.Errorf("%w: GitHub rate limited on %s", skillmarket.ErrRemoteFetchFail, u)
			}
			continue
		}
		can, perr := skilladapter.ParseSkillMD(body)
		if perr != nil {
			lastErr = perr
			continue
		}
		if can.Manifest.Name == "" {
			can.Manifest.Name = lastSegment(skillPath)
		}
		if can.Manifest.Author == "" {
			can.Manifest.Author = defaultAuthor
		}
		return can, nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("no candidate URL matched")
	}
	return nil, fmt.Errorf("%w: %v", skillmarket.ErrRemoteFetchFail, lastErr)
}

// splitRemoteID 拆 "owner/repo@skill-path" → (owner/repo, skill-path)。
func splitRemoteID(s string) (string, string, bool) {
	at := strings.LastIndex(s, "@")
	if at <= 0 || at == len(s)-1 {
		return "", "", false
	}
	repo := s[:at]
	skillPath := s[at+1:]
	if !strings.Contains(repo, "/") || strings.TrimSpace(skillPath) == "" {
		return "", "", false
	}
	return repo, skillPath, true
}

// lastSegment 取路径末段(用作 skill name 兜底)。
func lastSegment(p string) string {
	p = strings.Trim(p, "/")
	if i := strings.LastIndex(p, "/"); i >= 0 {
		return p[i+1:]
	}
	return p
}

// isRateLimited 2026-07-09 增:同 skillssh adapter 的识别规则。
func isRateLimited(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	if strings.Contains(msg, "status 429") || strings.Contains(msg, "status 403") {
		return true
	}
	lower := strings.ToLower(msg)
	return strings.Contains(lower, "rate limit") ||
		strings.Contains(lower, "too many requests") ||
		strings.Contains(lower, "api rate limit exceeded")
}

func (a *Adapter) fetchBody(ctx context.Context, u string) (string, error) {
	return httpx.GetJSONWithUA(ctx, a.httpClient, u)
}

func init() {
	skillmarket.Register(New())
}