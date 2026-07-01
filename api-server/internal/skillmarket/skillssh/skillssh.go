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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
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
	// 2026-07-01 改:用 /api/audits/{page} 公开 JSON API(无需鉴权)做主数据源。
	// 走 2 页 = 100 条;技能排名 100 之后的不在主列表展示(用户可走搜索或本地)。
	defaultAuditsAPIPath = "/api/audits/"
	defaultAuditsPages   = 2
	// 2026-07-01 改:fallback 行格式升级为 "owner/repo@skill | author | description",
	// 用 | 分隔,前段保留 remote_id,后两段填 MarketItem.Author / Description。
	// 真实环境若 audits API 不可达,fallback 也能展示基本信息。
	// 已扩充到 30 条;门槛提到 28(见 minCatalogFallbackSize)。
	knownCatalogFallback = "vercel-labs/agent-skills@vercel-react-best-practices | Vercel Engineering | Performance optimization guidelines for React and Next.js, maintained by Vercel Engineering.\n" +
		"vercel-labs/agent-skills@vercel-composition-patterns | Vercel Engineering | Composition patterns for React Server Components and Next.js App Router.\n" +
		"vercel-labs/agent-skills@vercel-server-actions | Vercel Engineering | Server Actions best practices: form handling, revalidation, error states.\n" +
		"vercel-labs/agent-skills@vercel-async-design | Vercel Engineering | Async patterns in React: Suspense, streaming, parallel routes, loading UI.\n" +
		"vercel-labs/agent-skills@next-best-practices | Vercel Engineering | Next.js best practices: data fetching, caching, revalidation, routing.\n" +
		"ComposioHQ/awesome-claude-skills@pr-review | Composio | Pull request review checklist and inline comment guidance.\n" +
		"ComposioHQ/awesome-claude-skills@commit-message | Composio | Conventional commit message writer with type scope detection.\n" +
		"ComposioHQ/awesome-claude-skills@code-explain | Composio | Explain a code block: what it does, why, edge cases.\n" +
		"ComposioHQ/awesome-claude-skills@security-audit | Composio | Audit code for common security issues (injection, secrets, auth).\n" +
		"obra/superpowers@brainstorming | Obra | Brainstorm a feature with structured prompts before implementation.\n" +
		"obra/superpowers@writing-plans | Obra | Write an implementation plan from a brainstormed design.\n" +
		"obra/superpowers@writing-skills | Obra | Author a new skill: frontmatter, body, examples, anti-patterns.\n" +
		"obra/superpowers@test-driven-development | Obra | TDD red-green-refactor workflow with focused unit tests.\n" +
		"obra/superpowers@using-git-worktrees | Obra | Use git worktrees to isolate feature work and reviews.\n" +
		"obra/superpowers@verification-before-completion | Obra | Self-check before marking work done: tests, types, lint, smoke.\n" +
		"200ideas/dofld-skills@dofld-commit | 200ideas | Stage-aware commit messages for solo or team workflows.\n" +
		"200ideas/dofld-skills@dofld-pr | 200ideas | PR description template with rationale, screenshots, test plan.\n" +
		"200ideas/dofld-skills@dofld-test | 200ideas | Generate test scaffolding from a function signature or user story.\n" +
		"dylnuge/skillbox-claude-skills@frontend-design | dylnuge | Frontend design heuristics: typography, color, spacing, hierarchy.\n" +
		"dylnuge/skillbox-claude-skills@tailwind-patterns | dylnuge | Tailwind utility composition patterns for readable UI markup.\n" +
		"dylnuge/skillbox-claude-skills@vue-best-practices | dylnuge | Vue 3 best practices: composition API, reactivity, lifecycle.\n" +
		"dylnuge/skillbox-claude-skills@react-best-practices | dylnuge | React best practices: hooks, state, effects, performance.\n" +
		"anthropics/skills@brand-guidelines | Anthropic | Apply Anthropic brand voice, tone, and visual style to copy.\n" +
		"anthropics/skills@web-artifacts-builder | Anthropic | Build self-contained HTML/JS/CSS artifacts for the web.\n" +
		"anthropics/skills@doc-coauthoring | Anthropic | Co-author a document: outline, draft, review, polish.\n" +
		"anthropics/skills@theme-factory | Anthropic | Generate themed CSS tokens and component snippets.\n" +
		"anthropics/skills@canvas-design | Anthropic | Compose designs on a canvas with primitives: shape, text, layout.\n" +
		"anthropics/skills@pdf | Anthropic | Read, edit, and extract content from PDF documents.\n" +
		"anthropics/skills@mcp-builder | Anthropic | Author an MCP server: tools, resources, prompts, transport.\n" +
		"anthropics/skills@frontend-design | Anthropic | Frontend design heuristics aligned with Anthropic design language.\n" +
		"anthropics/skills@skill-creator | Anthropic | Scaffold a new skill: SKILL.md template + body outline."
)

// minCatalogFallbackSize parseCatalog 解析后必须达到的最低条目数;
// 低于该值会触发 logger.Warn 提示需要补充 fallback(用于回归测试)。
//
// 2026-07-01 增:fallback 从 23 → 30,门槛提到 28。
var minCatalogFallbackSize = 28

// Adapter skills.sh 适配器。
type Adapter struct {
	httpClient *http.Client
	// rawBaseOverride 允许测试时替换 defaultGHRawBase(默认空)
	rawBaseOverride string
}

// New 构造 Adapter。
// 2026-07-01 改:timeout 20s → 30s(与 skillhub 保持一致;真实页面 + search 页面解析更慢)。
func New() *Adapter {
	return &Adapter{
		httpClient: &http.Client{Timeout: 30 * time.Second},
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
//
// 2026-07-01 改:三段式 —
//   1) 优先走 /api/audits/{page} 公开 JSON API(无需鉴权,含 author/description/tags)
//   2) JSON 解析失败时回退 HTML 解析(老版 @ 文本 + 新版 href 链接),合并去重
//   3) HTML 也为空时走 knownCatalogFallback
//
// 2026-07-01 增:keyword 参数处理。
//   - 空 keyword:走 /api/audits/{0..N-1} 全量目录
//   - 非空 keyword:走 /api/audits/0 全量 + substring 过滤(API 不直接支持关键字);
//     也可走 GET /search?q=<encoded> HTML 解析,失败时降级到 knownCatalogFallback
func (a *Adapter) Discover(ctx context.Context, baseURL, keyword string) ([]skillmarket.MarketItem, error) {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	kw := strings.TrimSpace(keyword)

	// 1) 优先:JSON API(/api/audits/{page})
	// 空 keyword 时拉多页(默认 2 页 = 100 条);非空 keyword 时只拉首页,再 substring 过滤
	pages := defaultAuditsPages
	if kw != "" {
		pages = 1
	}
	if items, ok := a.discoverFromAuditsAPI(ctx, baseURL, pages); ok && len(items) > 0 {
		if kw != "" {
			items = filterItemsByKeyword(items, kw)
		}
		if len(items) > 0 {
			return items, nil
		}
	}

	// 2) 回退:HTML 解析(首页 / 搜索页)
	var targetURL string
	if kw == "" {
		targetURL = strings.TrimRight(baseURL, "/") + "/"
	} else {
		targetURL = strings.TrimRight(baseURL, "/") + "/search?q=" + url.QueryEscape(kw)
	}
	body, err := a.fetchBody(ctx, targetURL)
	if err != nil {
		logger.Warn("skillssh discover: %v; falling back to known catalog", err)
		return filterCatalogByKeyword(knownCatalogFallback, baseURL, kw), nil
	}

	// 合并两个解析器:老版纯文本 owner/repo@skill + 新版 href 链接
	seen := map[string]bool{}
	out := make([]skillmarket.MarketItem, 0, 64)
	add := func(items []skillmarket.MarketItem) {
		for _, it := range items {
			if seen[it.RemoteID] {
				continue
			}
			seen[it.RemoteID] = true
			out = append(out, it)
		}
	}
	add(parseOwnerRepoAtBody(body, baseURL))
	add(parseHTMLLinks(body, baseURL))

	// 关键词二次过滤
	if kw != "" {
		out = filterItemsByKeyword(out, kw)
	}

	// 3) HTML 解析为空 → 走 knownCatalogFallback + substring 过滤
	if len(out) == 0 {
		return filterCatalogByKeyword(knownCatalogFallback, baseURL, kw), nil
	}
	return out, nil
}

// discoverFromAuditsAPI 走 /api/audits/{0..pages-1} 拉 JSON,合并去重转 MarketItem。
//
// 字段映射:
//   - RemoteID    = "{source}@{skillId}"  (与 HTML 路径一致)
//   - Name        = skillId(URL slug)
//   - Author      = source 的 owner 部分("vercel-labs/skills" → "vercel-labs")
//   - Description = agentTrustHub.result.gemini_analysis.summary(裁剪到 280 字)
//   - Tags        = [overall_risk_level](SAFE/LOW/MEDIUM/HIGH)
//   - DetailURL   = "{baseURL}/{source}/{skillId}"
//   - UpdatedAt   = 暂不填(API 没暴露更新时间)
//
// 失败/解析异常 → 返回 (nil, false),调用方降级到 HTML 解析。
func (a *Adapter) discoverFromAuditsAPI(ctx context.Context, baseURL string, pages int) ([]skillmarket.MarketItem, bool) {
	if pages <= 0 {
		pages = 1
	}
	seen := map[string]bool{}
	out := make([]skillmarket.MarketItem, 0, 64)

	for p := 0; p < pages; p++ {
		u := strings.TrimRight(baseURL, "/") + defaultAuditsAPIPath + fmt.Sprintf("%d", p)
		body, err := a.fetchBody(ctx, u)
		if err != nil {
			logger.Warn("skillssh audits API page %d: %v", p, err)
			// 第一页失败直接放弃 JSON 路径
			if p == 0 {
				return nil, false
			}
			// 后续页失败:已拿到前面的就返回
			break
		}
		var resp auditsAPIResponse
		if err := json.Unmarshal([]byte(body), &resp); err != nil {
			logger.Warn("skillssh audits API page %d: unmarshal: %v", p, err)
			if p == 0 {
				return nil, false
			}
			break
		}
		for _, s := range resp.Skills {
			if s.Source == "" || s.SkillID == "" {
				continue
			}
			remoteID := s.Source + "@" + s.SkillID
			if seen[remoteID] {
				continue
			}
			seen[remoteID] = true
			author := s.Source
			if idx := strings.Index(s.Source, "/"); idx > 0 {
				author = s.Source[:idx]
			}
			item := skillmarket.MarketItem{
				RemoteID:  remoteID,
				Name:      s.SkillID,
				Author:    author,
				DetailURL: fmt.Sprintf("%s/%s/%s", strings.TrimRight(baseURL, "/"), s.Source, s.SkillID),
			}
			// description: 优先用 gemini summary,裁剪到 280 字符
			if s.AgentTrustHub != nil && s.AgentTrustHub.Result.GeminiAnalysis.Summary != "" {
				item.Description = trimDescription(s.AgentTrustHub.Result.GeminiAnalysis.Summary, 280)
			}
			// tags: 安全等级作为可见标签
			if level := s.AgentTrustHub.Result.OverallRiskLevel; level != "" {
				item.Tags = []string{"risk:" + strings.ToLower(level)}
			}
			out = append(out, item)
		}
	}

	if len(out) == 0 {
		return nil, false
	}
	return out, true
}

// auditsAPIResponse 对应 /api/audits/{page} 的响应(只取需要的字段)。
type auditsAPIResponse struct {
	Skills []struct {
		Rank     int    `json:"rank"`
		Source   string `json:"source"`
		SkillID  string `json:"skillId"`
		Name     string `json:"name"`
		AgentTrustHub *struct {
			Source string `json:"source"`
			Slug   string `json:"slug"`
			Result struct {
				GeminiAnalysis struct {
					Verdict  string `json:"verdict"`
					Summary  string `json:"summary"`
					Categories []string `json:"categories"`
				} `json:"gemini_analysis"`
				OverallRiskLevel string `json:"overall_risk_level"`
			} `json:"result"`
		} `json:"agentTrustHub"`
		Socket *json.RawMessage `json:"socket"`
		Snyk   *json.RawMessage `json:"snyk"`
	} `json:"skills"`
}

// trimDescription 把 description 文本裁剪到 max 字符(避免长文本撑爆卡片布局)。
// 在最近的句号/逗号/空格处断行更友好,避免把英文单词从中间切。
func trimDescription(s string, max int) string {
	s = strings.TrimSpace(s)
	if max <= 0 || len(s) <= max {
		return s
	}
	cut := s[:max]
	// 找最近的句号/逗号/分号断行(从 cut 末尾往回找,最多回溯 50 字符)。
	start := len(cut) - 1
	if start > max-1 {
		start = max - 1
	}
	limit := start - 50
	if limit < 0 {
		limit = 0
	}
	for i := start; i >= limit; i-- {
		if cut[i] == '.' || cut[i] == ',' || cut[i] == ';' {
			return strings.TrimSpace(cut[:i+1])
		}
	}
	return strings.TrimSpace(cut) + "…"
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
// 2026-07-01 改:行格式 "owner/repo@skill | author | description"(用 | 分隔,前段必填,
// 后两段可选;不填则为空字符串),让 fallback 在审计 API 不可达时也能展示 author/description。
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
		// 拆分 "remote_id | author | description"(后两段可空)
		parts := strings.SplitN(line, "|", 3)
		head := strings.TrimSpace(parts[0])
		repo, name, ok := splitRemoteID(head)
		if !ok {
			continue
		}
		item := skillmarket.MarketItem{
			RemoteID:  head,
			Name:      name,
			DetailURL: fmt.Sprintf("%s/%s/%s", baseURL, repo, name),
		}
		if len(parts) >= 2 {
			item.Author = strings.TrimSpace(parts[1])
		}
		if len(parts) >= 3 {
			item.Description = strings.TrimSpace(parts[2])
		}
		out = append(out, item)
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

// filterCatalogByKeyword 对 fallback 列表做 substring 匹配(case-insensitive)。
//
// 2026-07-01 增:keyword 透传到 fallback 时,按 name / remote_id 子串命中过滤;
// 空 keyword = 全量。匹配为空时仍返回空切片(调用方已知道这是 fallback 状态)。
func filterCatalogByKeyword(text, baseURL, kw string) []skillmarket.MarketItem {
	base := parseCatalog(text, baseURL)
	if kw == "" {
		return base
	}
	out := make([]skillmarket.MarketItem, 0, len(base))
	lk := strings.ToLower(kw)
	for _, it := range base {
		if strings.Contains(strings.ToLower(it.RemoteID), lk) ||
			strings.Contains(strings.ToLower(it.Name), lk) {
			out = append(out, it)
		}
	}
	return out
}

// filterItemsByKeyword 对真实 HTML 解析后的 items 做 substring 二次过滤。
//
// 2026-07-01 增:防御性 — 即使 HTML 解析器匹配到条目,业务上仍按 keyword 收敛,
// 避免用户输入"react"却看到首页全部 30 条。
func filterItemsByKeyword(items []skillmarket.MarketItem, kw string) []skillmarket.MarketItem {
	lk := strings.ToLower(kw)
	out := make([]skillmarket.MarketItem, 0, len(items))
	for _, it := range items {
		if strings.Contains(strings.ToLower(it.RemoteID), lk) ||
			strings.Contains(strings.ToLower(it.Name), lk) {
			out = append(out, it)
		}
	}
	return out
}

// 注册到默认 registry。
func init() {
	skillmarket.Register(New())
}
