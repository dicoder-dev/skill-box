// Package skillssh 实现 skills.sh 适配器。
//
// skills.sh 是一个 catalog 站点,展示 "open agent skills ecosystem" 里的 skill。
// 站点的目录(浏览页)按 owner/repo@skill 形式组织;实际 skill 内容是 GitHub 仓库里的
// 一个子目录(常见路径: skills/<name>/SKILL.md)。
//
// 适配策略(v1):
//   - BaseURL 默认 https://skills.sh
//   - Discover: 解析浏览页 HTML(简单解析 "owner/repo@skill" 模式);失败时回退到
//     一个内置的 known-good 列表,保证 UI 有内容可看
//   - Detail:   解析详情页 + 拉对应 GitHub raw SKILL.md
//   - Download: 走 GitHub raw URL 拉 SKILL.md,转成 canonical
//
// 真实环境若 skills.sh 改版,BaseURL 可在 market_sources.config_json.base_url 覆盖。
package skillssh

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"ginp-api/internal/skilladapter"
	"ginp-api/internal/skillmarket"
	"ginp-api/pkg/logger"
)

const (
	defaultBaseURL    = "https://skills.sh"
	defaultGHRawBase  = "https://raw.githubusercontent.com"
	defaultGHBlobBase = "https://github.com"
	// 预置 fallback 列表(skills.sh 不可达 / 解析失败时兜底)。
	//
	// 2026-06-30 增:从公开的 vercel-labs / ComposioHQ / obra / 200ideas 等
	// 仓库里筛了 23 条已知可用的 skill,避免解析失败时市场页空着。
	knownCatalogFallback = "vercel-labs/agent-skills@vercel-react-best-practices\n" +
		"vercel-labs/agent-skills@vercel-composition-patterns\n" +
		"vercel-labs/agent-skills@vercel-server-actions\n" +
		"vercel-labs/agent-skills@vercel-async-design\n" +
		"ComposioHQ/awesome-claude-skills@pr-review\n" +
		"ComposioHQ/awesome-claude-skills@commit-message\n" +
		"ComposioHQ/awesome-claude-skills@code-explain\n" +
		"ComposioHQ/awesome-claude-skills@security-audit\n" +
		"obra/superpowers@brainstorming\n" +
		"obra/superpowers@writing-plans\n" +
		"obra/superpowers@writing-skills\n" +
		"obra/superpowers@test-driven-development\n" +
		"obra/superpowers@using-git-worktrees\n" +
		"200ideas/dofld-skills@dofld-commit\n" +
		"200ideas/dofld-skills@dofld-pr\n" +
		"200ideas/dofld-skills@dofld-test\n" +
		"dylnuge/skillbox-claude-skills@frontend-design\n" +
		"dylnuge/skillbox-claude-skills@tailwind-patterns\n" +
		"anthropics/skills@brand-guidelines\n" +
		"anthropics/skills@web-artifacts-builder\n" +
		"anthropics/skills@doc-coauthoring\n" +
		"anthropics/skills@theme-factory\n" +
		"anthropics/skills@canvas-design\n"
)

// minCatalogFallbackSize parseCatalog 解析后必须达到的最低条目数;
// 低于该值会触发 logger.Warn 提示需要补充 fallback(用于回归测试)。
//
// 2026-06-30 增:不能放在 const 块里(同一文件其它 const 不能跨块引用),
// 这里作为 package-level var。
var minCatalogFallbackSize = 20

// Adapter skills.sh 适配器。
type Adapter struct {
	httpClient *http.Client
	// rawBaseOverride 允许测试时替换 defaultGHRawBase(默认空)
	rawBaseOverride string
}

// New 构造 Adapter。
func New() *Adapter {
	return &Adapter{
		httpClient: &http.Client{Timeout: 20 * time.Second},
	}
}

// NewWithClient 构造 Adapter(测试用,允许注入 http.RoundTripper mock)。
func NewWithClient(c *http.Client) *Adapter {
	if c == nil {
		return New()
	}
	return &Adapter{httpClient: c}
}

// SetRawBaseOverride 替换 GitHub raw base(测试用);空 = 用 default。
func (a *Adapter) SetRawBaseOverride(u string) {
	a.rawBaseOverride = u
}

// rawBase 返回当前 raw base URL。
func (a *Adapter) rawBase() string {
	if a.rawBaseOverride != "" {
		return a.rawBaseOverride
	}
	return defaultGHRawBase
}

func (a *Adapter) SourceID() string    { return skillmarket.SourceSkillsSH }
func (a *Adapter) DisplayName() string { return "skills.sh" }
func (a *Adapter) BaseURL() string     { return defaultBaseURL }

// Discover 解析 catalog 页,提取 (owner/repo, skill) 列表。
func (a *Adapter) Discover(ctx context.Context, baseURL string) ([]skillmarket.MarketItem, error) {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	url := strings.TrimRight(baseURL, "/") + "/"
	body, err := a.fetchBody(ctx, url)
	if err != nil {
		logger.Warn("skillssh discover: %v; falling back to known catalog", err)
		return parseCatalog(knownCatalogFallback, baseURL), nil
	}
	// 简单匹配 "owner/repo@skill" 形式(站点 HTML 里通常以纯文本 / href 出现)
	pattern := regexp.MustCompile(`([\w.-]+/[\w.-]+)@([\w.-]+)`)
	matches := pattern.FindAllStringSubmatch(body, 500)
	seen := map[string]bool{}
	out := make([]skillmarket.MarketItem, 0, len(matches))
	for _, m := range matches {
		repo := strings.TrimSpace(m[1])
		name := strings.TrimSpace(m[2])
		if repo == "" || name == "" {
			continue
		}
		remoteID := repo + "@" + name
		if seen[remoteID] {
			continue
		}
		seen[remoteID] = true
		out = append(out, skillmarket.MarketItem{
			RemoteID:  remoteID,
			Name:      name,
			DetailURL: fmt.Sprintf("%s/%s/%s", defaultBaseURL, repo, name),
		})
	}
	if len(out) == 0 {
		return parseCatalog(knownCatalogFallback, baseURL), nil
	}
	return out, nil
}

// Detail 拉详情(只填展示字段;canonical 走 Download)。
func (a *Adapter) Detail(ctx context.Context, baseURL, remoteID string) (*skillmarket.MarketDetail, error) {
	if remoteID == "" {
		return nil, skillmarket.ErrEmptyRemoteID
	}
	repo, name, ok := splitRemoteID(remoteID)
	if !ok {
		return nil, fmt.Errorf("%w: invalid remote id %q", skillmarket.ErrRemoteNotFound, remoteID)
	}
	detail := &skillmarket.MarketDetail{
		MarketItem: skillmarket.MarketItem{
			RemoteID:  remoteID,
			Name:      name,
			DetailURL: fmt.Sprintf("%s/%s/%s", defaultBaseURL, repo, name),
		},
		Homepage: fmt.Sprintf("%s/%s", defaultGHBlobBase, repo),
	}
	// 详情页(可选,失败不致命)
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	if body, err := a.fetchBody(ctx, fmt.Sprintf("%s/%s/%s", strings.TrimRight(baseURL, "/"), repo, name)); err == nil {
		detail.Description = extractFirstParagraph(body)
	}
	return detail, nil
}

// Download 拉 SKILL.md(从 GitHub raw)转 canonical。
func (a *Adapter) Download(ctx context.Context, baseURL, remoteID string) (*skilladapter.Canonical, error) {
	if remoteID == "" {
		return nil, skillmarket.ErrEmptyRemoteID
	}
	repo, name, ok := splitRemoteID(remoteID)
	if !ok {
		return nil, fmt.Errorf("%w: invalid remote id %q", skillmarket.ErrRemoteNotFound, remoteID)
	}
	// 常见路径尝试顺序(2026-06-30 改造:笛卡尔积 main/master × 3 个目录 = 6 条)。
	rawBase := a.rawBase()
	branches := []string{"main", "master"}
	dirs := []string{"skills", ".claude/skills", ""}
	candidates := make([]string, 0, len(branches)*len(dirs))
	for _, b := range branches {
		for _, d := range dirs {
			prefix := d
			if prefix != "" {
				prefix = prefix + "/"
			}
			candidates = append(candidates, fmt.Sprintf("%s/%s/%s/%s%s/SKILL.md", rawBase, repo, b, prefix, name))
		}
	}
	var lastErr error
	for _, u := range candidates {
		body, err := a.fetchBody(ctx, u)
		if err != nil {
			lastErr = err
			continue
		}
		can, perr := skilladapter.ParseSkillMD(body)
		if perr != nil {
			lastErr = perr
			continue
		}
		if can.Manifest.Name == "" {
			can.Manifest.Name = name
		}
		return can, nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("no candidate URL matched")
	}
	return nil, fmt.Errorf("%w: %v", skillmarket.ErrRemoteFetchFail, lastErr)
}

// fetchBody 拉 URL 文本,状态非 2xx 返错;超时/网络错误一并返错。
func (a *Adapter) fetchBody(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "skill-box/1.0 (+https://skillbox.local)")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,text/plain,application/json")
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if resp.StatusCode == http.StatusNotFound {
			return "", fmt.Errorf("%w: %s", skillmarket.ErrRemoteNotFound, url)
		}
		return "", fmt.Errorf("status %d for %s", resp.StatusCode, url)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// splitRemoteID 拆 "owner/repo@skill" → (owner/repo, skill)。
func splitRemoteID(remoteID string) (string, string, bool) {
	at := strings.LastIndex(remoteID, "@")
	if at <= 0 || at == len(remoteID)-1 {
		return "", "", false
	}
	repo := remoteID[:at]
	name := remoteID[at+1:]
	if !strings.Contains(repo, "/") {
		return "", "", false
	}
	return repo, name, true
}

// parseCatalog 解析预置 fallback 列表。
//
// 2026-06-30 增:解析后会校验长度,如果 < minCatalogFallbackSize 则 logger.Warn
// 提示有人改了 knownCatalogFallback 但数量不足,防止后续维护删条目导致 fallback 空。
func parseCatalog(text, baseURL string) []skillmarket.MarketItem {
	out := make([]skillmarket.MarketItem, 0, 16)
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		repo, name, ok := splitRemoteID(line)
		if !ok {
			continue
		}
		out = append(out, skillmarket.MarketItem{
			RemoteID:  line,
			Name:      name,
			DetailURL: fmt.Sprintf("%s/%s/%s", baseURL, repo, name),
		})
	}
	if len(out) < minCatalogFallbackSize {
		logger.Warn("skillssh fallback catalog has %d items (< %d); consider refilling knownCatalogFallback",
			len(out), minCatalogFallbackSize)
	}
	return out
}

// parseOwnerRepoAtBody 从 HTML body 里扫纯文本 "owner/repo@skill" 模式。
// 这是 skills.sh 老版站点的列表呈现方式(直接显示在卡片文本里)。
func parseOwnerRepoAtBody(body, baseURL string) []skillmarket.MarketItem {
	pattern := regexp.MustCompile(`([\w.-]+/[\w.-]+)@([\w.-]+)`)
	matches := pattern.FindAllStringSubmatch(body, 500)
	seen := map[string]bool{}
	out := make([]skillmarket.MarketItem, 0, len(matches))
	for _, m := range matches {
		repo := strings.TrimSpace(m[1])
		name := strings.TrimSpace(m[2])
		if repo == "" || name == "" {
			continue
		}
		remoteID := repo + "@" + name
		if seen[remoteID] {
			continue
		}
		seen[remoteID] = true
		out = append(out, skillmarket.MarketItem{
			RemoteID:  remoteID,
			Name:      name,
			DetailURL: fmt.Sprintf("%s/%s/%s", baseURL, repo, name),
		})
	}
	return out
}

// parseHTMLLinks 从 HTML body 里扫 <a href="/owner/repo/skill"> 链接模式。
// 这是 skills.sh 新版站点的列表呈现方式(每条 skill 是独立链接)。
func parseHTMLLinks(body, baseURL string) []skillmarket.MarketItem {
	pattern := regexp.MustCompile(`href="/?([\w.-]+/[\w.-]+)/([\w.-]+)"`)
	matches := pattern.FindAllStringSubmatch(body, 500)
	seen := map[string]bool{}
	out := make([]skillmarket.MarketItem, 0, len(matches))
	for _, m := range matches {
		repo := strings.TrimSpace(m[1])
		name := strings.TrimSpace(m[2])
		if repo == "" || name == "" {
			continue
		}
		if isReservedPath(name) {
			continue
		}
		remoteID := repo + "@" + name
		if seen[remoteID] {
			continue
		}
		seen[remoteID] = true
		out = append(out, skillmarket.MarketItem{
			RemoteID:  remoteID,
			Name:      name,
			DetailURL: fmt.Sprintf("%s/%s/%s", baseURL, repo, name),
		})
	}
	return out
}

// isReservedPath 排除明显的站点导航路径(about / docs / blog 等)。
// 这些 owner 仓库大多不存在,扫到会污染列表。
func isReservedPath(seg string) bool {
	switch strings.ToLower(seg) {
	case "about", "docs", "blog", "pricing", "login", "signup", "api", "changelog", "privacy", "terms":
		return true
	}
	return false
}

// extractFirstParagraph 从 HTML/MD 里取第一段非空文本(简化处理)。
func extractFirstParagraph(body string) string {
	re := regexp.MustCompile(`<[^>]+>`)
	plain := re.ReplaceAllString(body, " ")
	for _, line := range strings.Split(plain, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "!") {
			continue
		}
		if len(line) > 240 {
			return line[:240] + "…"
		}
		return line
	}
	return ""
}

// 注册到默认 registry。
func init() {
	skillmarket.Register(New())
}
