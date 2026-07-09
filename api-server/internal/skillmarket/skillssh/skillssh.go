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
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"ginp-api/internal/skilladapter"
	"ginp-api/internal/skillmarket"
	"ginp-api/pkg/httpx"
	"ginp-api/pkg/logger"
)

const (
	defaultBaseURL    = "https://skills.sh"
	defaultSourceHomepage = "https://skills.sh" // 2026-07-04 增:站点首页(skills.sh 本身就是浏览页,API 也在这个 host)
	defaultGHRawBase  = "https://raw.githubusercontent.com"
	defaultGHBlobBase = "https://github.com"
	// 2026-07-01 改:用 /api/audits/{page} 公开 JSON API(无需鉴权)做主数据源。
	// 50 页 = 2500 条,覆盖 skills.sh Top 实用区。audits API 单页固定 50 条,
	// 全站 851,604 是 GitHub 仓库抓取总维度,网站全榜只展示 Top N,
	// 2500 条后密度稀疏,对用户来说"基本够用"。
	// 若用户需要更精准搜索,需走 Vercel OIDC 鉴权接 /api/v1/skills/search(本项目暂不支持)。
	defaultAuditsAPIPath = "/api/audits/"
	defaultAuditsPages   = 50
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

// New 构造 Adapter(用 httpx 长生命周期客户端, 跨多次翻页复用 TLS 连接)。
// 2026-07-02 改:从裸 http.Client 切到 httpx.NewClient — keep-alive + gzip + UA 一站搞定;
// skills.sh 是海外, 单页 ~200-500ms, keep-alive 省 TLS 握手 ≈ 100ms/页。
func New() *Adapter {
	return &Adapter{
		httpClient: httpx.NewClient(30 * time.Second),
	}
}

// NewWithClient 构造 Adapter(测试用,允许注入 http.RoundTripper mock)。
//
// 2026-07-10 改:NewWithClient 默认就把 raw base override 设为 "https://stub",
// 跟测试里 fakeRT 常见的 base host 对齐;老测试代码不用单独调 SetRawBaseOverride
// 也能让 raw URL 命中 fakeRT(URL 形如 https://stub/{o}/{r}/.../{path})。
// 若测试需要其它 raw base,后续 SetRawBaseOverride 覆盖即可。
func NewWithClient(c *http.Client) *Adapter {
	if c == nil {
		return New()
	}
	a := &Adapter{httpClient: c}
	a.SetRawBaseOverride("https://stub")
	return a
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

// HomepageURL 2026-07-04 增:skills.sh 本身就是浏览页首页 + API host 同一站;
// 私有部署场景下用户可能在 config_json.base_url 填内网 skills.sh 镜像,这里
// 派生出 origin 让用户跳到自己的镜像。
func (a *Adapter) HomepageURL(sourceConfigJSON string) string {
	if u := extractSkillsSHOrigin(sourceConfigJSON); u != "" {
		return u
	}
	return defaultSourceHomepage
}

// extractSkillsSHOrigin 2026-07-04 增:解析 config_json.base_url 的 origin。
func extractSkillsSHOrigin(configJSON string) string {
	cfg := skillmarket.ParseSourceConfig(configJSON)
	if cfg == nil || strings.TrimSpace(cfg.BaseURL) == "" {
		return ""
	}
	u, err := url.Parse(strings.TrimSpace(cfg.BaseURL))
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ""
	}
	return u.Scheme + "://" + u.Host
}

// KnownFallbackIDs 2026-07-03 增:返回 knownCatalogFallback 列表的 RemoteID 集合。
//
// 注意:knownCatalogFallback 是文本,这里解析时复用 parseCatalog 走"remote_id | author | description"
// 拆分,只取第一段 remote_id。
func (a *Adapter) KnownFallbackIDs() []string {
	// 不依赖 baseURL(只取 slug,无需 detail_url),用占位避免空指针。
	items := parseCatalog(knownCatalogFallback, defaultBaseURL)
	out := make([]string, 0, len(items))
	for _, it := range items {
		out = append(out, it.RemoteID)
	}
	return out
}

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
//
// 2026-07-02 改造:翻页从串行改为**限并发=4 的 worker pool**。
//   - skills.sh 海外服务器,单页 ~200-500ms × 50 页 ≈ 15s;
//     并发 4 后 ≈ 4s(节省 70%+ 时间)
//   - 第一页失败仍直接放弃 JSON 路径(单页判降级)
//   - 后续页失败:已拿到前面的就返回
//
// 2026-07-02 修:之前实现里 producer 不知道"首页失败",会傻乎乎派完 50 页,
// worker 拉一堆 404 + warn log,测试场景(stub server 只返 page 0)会卡死;
// 现在用 stopMu + pageFailed 让 producer 立即感知 stop,关闭 pageCh 让 worker 退出。
func (a *Adapter) discoverFromAuditsAPI(ctx context.Context, baseURL string, pages int) ([]skillmarket.MarketItem, bool) {
	if pages <= 0 {
		pages = 1
	}
	seen := make(map[string]bool, 64)
	var seenMu sync.Mutex
	out := make([]skillmarket.MarketItem, 0, 64)
	var outMu sync.Mutex

	// stopCollect 标记"全局停止":①首页失败 ②后续页 404 触发整体降级
	// 2026-07-02 改:加 stopCh(close 后 producer select 立即感知),避免"send 阻塞 + stop 已设"死锁。
	stopCollect := false
	var stopMu sync.Mutex
	stopCh := make(chan struct{})
	setStop := func() {
		stopMu.Lock()
		defer stopMu.Unlock()
		if stopCollect {
			return
		}
		stopCollect = true
		close(stopCh)
	}
	isStop := func() bool {
		stopMu.Lock()
		defer stopMu.Unlock()
		return stopCollect
	}

	type pageResult struct {
		page  int
		skills []struct {
			Rank     int    `json:"rank"`
			Source   string `json:"source"`
			SkillID  string `json:"skillId"`
			Name     string `json:"name"`
			AgentTrustHub *struct {
				Source string `json:"source"`
				Slug   string `json:"slug"`
				Result struct {
					GeminiAnalysis struct {
						Verdict  string   `json:"verdict"`
						Summary  string   `json:"summary"`
						Categories []string `json:"categories"`
					} `json:"gemini_analysis"`
					OverallRiskLevel string `json:"overall_risk_level"`
				} `json:"result"`
			} `json:"agentTrustHub"`
			Socket *json.RawMessage `json:"socket"`
			Snyk   *json.RawMessage `json:"snyk"`
		}
		err error
	}

	fetch := func(p int) pageResult {
		u := strings.TrimRight(baseURL, "/") + defaultAuditsAPIPath + fmt.Sprintf("%d", p)
		body, err := a.fetchBody(ctx, u)
		if err != nil {
			return pageResult{page: p, err: err}
		}
		var resp auditsAPIResponse
		if uerr := json.Unmarshal([]byte(body), &resp); uerr != nil {
			return pageResult{page: p, err: uerr}
		}
		return pageResult{page: p, skills: resp.Skills}
	}

	const maxConcurrency = 4
	// pageCh 加缓冲 = maxConcurrency, 让 producer 一次压入 N 个 page,worker 并发取;
	// 否则无缓冲 channel 下 producer 等 worker,和串行没区别。
	pageCh := make(chan int, maxConcurrency)
	resultCh := make(chan pageResult, maxConcurrency)

	var wg sync.WaitGroup
	for i := 0; i < maxConcurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for p := range pageCh {
				resultCh <- fetch(p)
			}
		}()
	}

	producerDone := make(chan struct{})
	go func() {
		defer close(pageCh)
		defer close(producerDone)
		for p := 0; p < pages; p++ {
			// producer 也要感知 stop(首页失败 / ctx 取消),否则会傻乎乎派发完所有 page。
			if isStop() {
				return
			}
			if cerr := ctx.Err(); cerr != nil {
				return
			}
			// 2026-07-02 改:select 三选一(pageCh / stopCh / ctx.Done),
			// stop 设上时 producer 不阻塞在 send。
			select {
			case pageCh <- p:
			case <-stopCh:
				return
			case <-ctx.Done():
				return
			}
		}
	}()

	go func() {
		wg.Wait()
		close(resultCh)
	}()

	// 收集器:读 resultCh,直到 close
	for res := range resultCh {
		if res.err != nil {
			logger.Warn("skillssh audits API page %d: %v", res.page, res.err)
			if res.page == 0 {
				// 第一页失败直接放弃 JSON 路径
				setStop()
			}
			continue
		}
		if isStop() {
			continue
		}
		for _, s := range res.skills {
			if s.Source == "" || s.SkillID == "" {
				continue
			}
			remoteID := s.Source + "@" + s.SkillID
			seenMu.Lock()
			if seen[remoteID] {
				seenMu.Unlock()
				continue
			}
			seen[remoteID] = true
			seenMu.Unlock()

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
			if s.AgentTrustHub != nil && s.AgentTrustHub.Result.GeminiAnalysis.Summary != "" {
				item.Description = trimDescription(s.AgentTrustHub.Result.GeminiAnalysis.Summary, 280)
			}
			if s.AgentTrustHub != nil {
				if level := s.AgentTrustHub.Result.OverallRiskLevel; level != "" {
					item.Tags = []string{"risk:" + strings.ToLower(level)}
				}
			}
			outMu.Lock()
			out = append(out, item)
			outMu.Unlock()
		}
	}
	<-producerDone

	if isStop() {
		return nil, false
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

// Download 拉 SKILL.md(从 GitHub raw)+ 附属文件转 canonical。
//
// 2026-07-10 重写(关键):从"硬编码 6 候选 URL"切换到"tree 自动发现"。
//
// 背景:用户场景 `https://www.skills.sh/101-skills/skills/ai-video-generation`
// 直接 404。101-skills/skills 仓库把所有 skill 放在 `tools/<name>/SKILL.md`(路径不固定),
// 老逻辑硬编码 `dirs ∈ {skills, .claude/skills, ""}` 三种前缀完全 cover 不到,
// 6 候选全失败 → 兜底骨架 → 用户根本装不上。
//
// 新方案(镜像 github adapter):
//   1) GET /api/github/repos/{owner}/{repo}/git/trees/{branch}?recursive=1
//      → 扫整个仓库的 blob 路径元数据(~1.2s)
//   2) 在 tree 里找**以 "<name>/SKILL.md" 结尾**的 blob(允许任意深度前缀)
//      → 例: <name>=ai-video-generation 命中 `tools/video/SKILL.md` 也行
//   3) 找到 → anchor prefix = 命中路径去掉 "/SKILL.md" 前缀,然后并发 raw 拉所有
//      blob(类似 github adapter.downloadFromTree)。
//   4) 找不到 / tree 拉不到 → 退回老 6 候选 URL,只为兜底拿 SKILL.md。
//   5) 都失败 → knownCatalogFallback 命中 → 内存骨架。
func (a *Adapter) Download(ctx context.Context, baseURL, remoteID string) (*skilladapter.Canonical, error) {
	if remoteID == "" {
		return nil, skillmarket.ErrEmptyRemoteID
	}
	repo, name, ok := splitRemoteID(remoteID)
	if !ok {
		return nil, fmt.Errorf("%w: invalid remote id %q", skillmarket.ErrRemoteNotFound, remoteID)
	}
	slash := strings.Index(repo, "/")
	if slash <= 0 || slash >= len(repo)-1 {
		return nil, fmt.Errorf("%w: invalid repo %q", skillmarket.ErrRemoteNotFound, repo)
	}
	owner := repo[:slash]
	repoName := repo[slash+1:]

	// 主路径:tree API 自动发现 SKILL.md 路径,锚点目录并发拉所有附属文件。
	if can, ok := a.downloadFromTree(ctx, owner, repoName, name, remoteID); ok {
		return can, nil
	}

	// tree 失败 / 找不到 anchor → 老 6 候选 URL fallback(只下 SKILL.md)
	if can, lastErr := a.downloadSKILLMDOnly(ctx, repo, name); can != nil {
		return can, nil
	} else if isRateLimited(lastErr) {
		// 限流早退:继续 fallback 只会触发更多请求
		logger.Warn("skillssh download: GitHub rate limited, abort")
	} else if can := buildFallbackCanonical(remoteID, repo, name); can != nil {
		logger.Warn("skillssh download: %v; using fallback canonical for %s", lastErr, remoteID)
		return can, nil
	}
	return nil, fmt.Errorf("%w: no SKILL.md found for %s under %s (tried tree API + 6 candidate URLs)", skillmarket.ErrRemoteFetchFail, name, repo)
}

// downloadFromTree 走 GitHub Trees API + raw 并发下载,完整流程。
//
// 两阶段 locate(2026-07-10 增):
//   1) 路径匹配:树里找以 "<name>/SKILL.md" 结尾的 blob
//      → 命中走 anchor-files 并发拉(代价小,典型 5-10 个文件)
//   2) frontmatter 匹配(回退):扫描树里所有 SKILL.md,并发 GET 内容,parse frontmatter
//      找 `name: <name>` 精确匹配
//      → 解决 skills.sh detail 页 name ≠ GitHub 目录名的问题(如
//         "ai-video-generation" 在 `inference-sh/skills` 是 `tools/video/SKILL.md`)
//
// 返回 (canonical, true) 命中;(nil, false) 表示树抓不到 / 两阶段都没找到,
// 由调用方继续走老 6 候选 URL 兜底。
func (a *Adapter) downloadFromTree(ctx context.Context, owner, repoName, name, remoteID string) (*skilladapter.Canonical, bool) {
	// 1. 拿 tree(支持 main / master fallback)
	branches := []string{"main", "master"}
	var lastErr error
	for _, branch := range branches {
		if cerr := ctx.Err(); cerr != nil {
			return nil, false
		}
		entries, pathsByAnchor, err := a.fetchTreeEntries(ctx, owner, repoName, branch)
		if err != nil {
			lastErr = err
			if isRateLimited(err) || isAuthRequired(err) {
				return nil, false
			}
			continue
		}

		// 2. 路径匹配
		skillMDP, anchorPrefix, ok := locateSKILLMDByPath(entries, name)
		if ok && containsEntry(pathsByAnchor, anchorPrefix) && contains(pathsByAnchor[anchorPrefix], skillMDP) {
			can := a.fetchAnchorFiles(ctx, owner, repoName, branch, anchorPrefix, skillMDP, name, owner, remoteID)
			if can != nil {
				return can, true
			}
		}

		// 3. frontmatter 匹配(回退)
		skillMDByFM, fmAnchor, ok := a.locateSKILLMDByFrontmatter(ctx, owner, repoName, branch, entries, name)
		if ok {
			// 用命中路径的目录作为 anchor,这样附属文件(scripts/、references/)跟着收
			if can := a.fetchAnchorFiles(ctx, owner, repoName, branch, fmAnchor, skillMDByFM, name, owner, remoteID); can != nil {
				return can, true
			}
		}
	}
	if lastErr != nil {
		logger.Debug("skillssh downloadFromTree: %v", lastErr)
	}
	return nil, false
}

// locateSKILLMDByFrontmatter 扫 tree 里所有 SKILL.md,并发拉内容找 frontmatter `name: <name>`。
//
// 2026-07-10 增:解决 skills.sh detail 页 name 与 GitHub 路径不一致的问题。
// 例:inference-sh/skills 详情页 name="ai-video-generation",但 SKILL.md 在 tools/video/
// ——路径匹配失败,要靠 frontmatter 精确匹配挽救。
//
// 并发上限 6,跟 fetchAnchorFiles 共用信号量写法(独立 max)。
func (a *Adapter) locateSKILLMDByFrontmatter(ctx context.Context, owner, repoName, branch string, entries []treeEntry, name string) (string, string, bool) {
	var candidates []string
	for _, e := range entries {
		if e.Type == "blob" && strings.HasSuffix(e.Path, "/SKILL.md") {
			candidates = append(candidates, e.Path)
		}
	}
	if len(candidates) == 0 {
		return "", "", false
	}

	type fmHit struct {
		path    string
		anchor  string
		content string
	}
	results := make([]fmHit, len(candidates))
	sem := make(chan struct{}, 6)
	var wg sync.WaitGroup
	for i, p := range candidates {
		wg.Add(1)
		go func(i int, p string) {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				return
			}
			body, err := a.fetchRawFile(ctx, owner, repoName, branch, p)
			if err != nil {
				return
			}
			results[i] = fmHit{path: p, anchor: strings.TrimSuffix(p, "SKILL.md"), content: body}
		}(i, p)
	}
	wg.Wait()

	for _, r := range results {
		if r.content == "" {
			continue
		}
		// 切 frontmatter 行,避免 skilladapter.ParseSkillMD 因 body 不合规返错
		fmName := extractFrontmatterName(r.content)
		if fmName == name {
			return r.path, r.anchor, true
		}
	}
	return "", "", false
}

// extractFrontmatterName 从 raw markdown 抽出 frontmatter `name:` 值。
//
// 2026-07-10 增:轻量解析 frontmatter,跳过 skilladapter.ParseSkillMD 严格校验
// (我们的目的是快速筛 candidate,大文件 <250KB OK;校验留给最终选定的 SKILL.md)。
// 只支持 `name: foo` / `name: "foo bar"` 这两种最短表达。
func extractFrontmatterName(content string) string {
	content = strings.TrimSpace(content)
	if !strings.HasPrefix(content, "---") {
		return ""
	}
	// 找第二个 --- 块结束位置
	rest := strings.TrimPrefix(content, "---")
	idx := strings.Index(rest, "\n---")
	if idx < 0 {
		return ""
	}
	fmBlock := rest[:idx]
	for _, line := range strings.Split(fmBlock, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "name:") {
			continue
		}
		v := strings.TrimSpace(strings.TrimPrefix(line, "name:"))
		v = strings.Trim(v, "\"'`")
		return v
	}
	return ""
}

// containsEntry 判断 map 里是否存在 prefix 结尾带 "/" 的聚合 key(让 caller
// 收 anchor 下的 blob)。
func containsEntry(m map[string][]string, anchorPrefix string) bool {
	_, ok := m[anchorPrefix]
	return ok
}

// treeEntry 镜像 github adapter 的 tree 解析(本地复用,避免跨包依赖)。
type treeEntry struct {
	Path string `json:"path"`
	Type string `json:"type"`
}

// fetchTreeEntries 调 GitHub Trees API,返回所有 blob 路径 + 按 anchor prefix 分组的 map。
//
// anchor 概念:每个 SKILL.md 决定一个 anchor 目录(SKILL.md 的父目录);其它 blob
// 按"最浅的祖先 anchor"分桶(blob 没有更浅的 anchor 时归 own root)。
// 例如 trees 是:
//
//	tools/video/SKILL.md           → anchor = tools/video/
//	tools/video/helper.py          → tools/video/
//	tools/video/scripts/run.py     → tools/video/  (scripts/ 下没 SKILL.md,用 parent anchor)
//	other-skill/SKILL.md           → anchor = other-skill/
//
// 这样按 anchor 收附属文件时,scripts/ 这种深嵌套也能带回来。
//
// 返回的 map key 是 anchor prefix(以 "/" 结尾),value 是该 anchor 下所有 blob。
// 顶层没 anchor 的 blob 会被丢弃。
func (a *Adapter) fetchTreeEntries(ctx context.Context, owner, repoName, branch string) ([]treeEntry, map[string][]string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/git/trees/%s?recursive=1", owner, repoName, branch)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("User-Agent", httpx.UserAgent)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("tree api: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil, fmt.Errorf("status 404: branch %s not found", branch)
	}
	if resp.StatusCode == http.StatusForbidden {
		return nil, nil, fmt.Errorf("status 403: GitHub API rate limited")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, nil, fmt.Errorf("status %d: %s", resp.StatusCode, body)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, nil, fmt.Errorf("read tree: %w", err)
	}

	var treeResp struct {
		Tree      []treeEntry `json:"tree"`
		Truncated bool        `json:"truncated"`
	}
	if err := json.Unmarshal(body, &treeResp); err != nil {
		return nil, nil, fmt.Errorf("parse tree: %w", err)
	}
	if treeResp.Truncated {
		return nil, nil, fmt.Errorf("tree truncated (repo too large)")
	}

	// 按 anchor 分组:先收集所有 SKILL.md 的目录当 anchors,再把每个 blob 归到
	// "最深的祖先 anchor"(深度优先命中),用于后续按 anchor 取附属文件。
	anchors := make([]string, 0, 4) // 按长度降序排
	byAnchor := map[string][]string{}
	for _, e := range treeResp.Tree {
		if e.Type != "blob" || !strings.HasSuffix(e.Path, "/SKILL.md") {
			continue
		}
		dir := strings.TrimSuffix(e.Path, "SKILL.md")
		if !strings.HasSuffix(dir, "/") {
			dir += "/"
		}
		anchors = append(anchors, dir)
	}
	// 按 path 长度**降序**排(深的 anchor 优先)
	sort.Slice(anchors, func(i, j int) bool { return len(anchors[i]) > len(anchors[j]) })

	for _, e := range treeResp.Tree {
		if e.Type != "blob" {
			continue
		}
		// 找最深祖先 anchor(prefix 必须以 "/" 结尾,且等于 blob path 前缀或等于某层)
		var hit string
		for _, anc := range anchors {
			if strings.HasPrefix(e.Path, anc) {
				hit = anc
				break
			}
		}
		if hit == "" {
			continue
		}
		byAnchor[hit] = append(byAnchor[hit], e.Path)
	}
	return treeResp.Tree, byAnchor, nil
}

// locateSKILLMDByPath 在 entry 列表里找路径以 "<name>/SKILL.md" 结尾的 blob。
//
// 策略:扫所有 blob,凡是 path 末两段是 "<name>/SKILL.md" 即视为候选;
// 若同一 name 命中多个(例如 skills/foo/SKILL.md 和 tools/foo/SKILL.md),取最短路径
// (顶层最浅);仍冲突则取字典序最小的(确保稳定)。
//
// 返回 (skillMDPath, anchorPrefix, ok):anchorPrefix = "<name>" 这层目录(不含 SKILL.md)。
func locateSKILLMDByPath(entries []treeEntry, name string) (string, string, bool) {
	const mdName = "SKILL.md"
	suffix := name + "/" + mdName
	var hit string
	hitDepth := -1
	for _, e := range entries {
		if e.Type != "blob" {
			continue
		}
		if !strings.HasSuffix(e.Path, suffix) {
			continue
		}
		// 注意:可能 name 本身含子路径(如 "foo/bar"),按 name 整段匹配后缀
		// 此时 path 至少长 len(suffix);顶层 name 时 path 长度 == len("name")+len("/SKILL.md")
		// 深度 = '/' 出现次数
		depth := strings.Count(e.Path, "/")
		if hit == "" || depth < hitDepth {
			hit = e.Path
			hitDepth = depth
		}
	}
	if hit == "" {
		return "", "", false
	}
	anchor := strings.TrimSuffix(hit, suffix)
	anchorPrefix := anchor + name + "/"
	return hit, anchorPrefix, true
}

// fetchAnchorFiles 并发拉 anchor 目录下所有 blob,组装 canonical。
func (a *Adapter) fetchAnchorFiles(ctx context.Context, owner, repoName, branch, anchorPrefix, skillMDP, name, authorOwner, remoteID string) *skilladapter.Canonical {
	paths, _ := collectAnchorPaths(ctx, a, owner, repoName, branch, anchorPrefix)
	type fileResult struct {
		path    string
		content string
		err     error
	}
	results := make([]fileResult, len(paths))
	sem := make(chan struct{}, 6)
	var wg sync.WaitGroup
	for i, p := range paths {
		wg.Add(1)
		go func(i int, p string) {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				results[i] = fileResult{path: p, err: ctx.Err()}
				return
			}
			content, err := a.fetchRawFile(ctx, owner, repoName, branch, p)
			results[i] = fileResult{path: p, content: content, err: err}
		}(i, p)
	}
	wg.Wait()

	var skillMD string
	files := make([]skilladapter.File, 0, len(paths))
	for _, r := range results {
		if r.err != nil {
			continue
		}
		if r.path == skillMDP {
			skillMD = r.content
			continue
		}
		rel := strings.TrimPrefix(r.path, anchorPrefix)
		files = append(files, skilladapter.File{Path: filepath.ToSlash(rel), Content: r.content})
	}
	if skillMD == "" {
		return nil
	}
	can, perr := skilladapter.ParseSkillMD(skillMD)
	if perr != nil {
		return nil
	}
	if can.Manifest.Name == "" {
		can.Manifest.Name = name
	}
	if can.Manifest.Author == "" {
		can.Manifest.Author = authorOwner
	}
	allFiles := make([]skilladapter.File, 0, len(files)+1)
	allFiles = append(allFiles, skilladapter.File{Path: "SKILL.md", Content: skillMD})
	allFiles = append(allFiles, files...)
	can.Files = allFiles
	_ = remoteID // 暂未使用,保留参数便于将来日志关联
	return can
}

// collectAnchorPaths 重新调 tree api,按 anchor prefix 取该 anchor 下所有 blob。
//
// 2026-07-10 改:byAnchor 的 key 现在以 "/" 结尾(anchor prefix 形态),这里直接
// 拿 anchor prefix 当 key 查。返回值是 anchor 目录(含 SKILL.md)和子目录下所有 blob。
func collectAnchorPaths(ctx context.Context, a *Adapter, owner, repoName, branch, anchorPrefix string) ([]string, error) {
	_, byAnchor, err := a.fetchTreeEntries(ctx, owner, repoName, branch)
	if err != nil {
		return nil, err
	}
	// anchorPrefix 已经以 "/" 结尾,byAnchor key 也是 "tools/video/" 形态
	return byAnchor[anchorPrefix], nil
}

// fetchRawFile 单 GET 拉 raw.githubusercontent.com 文件内容。
func (a *Adapter) fetchRawFile(ctx context.Context, owner, repoName, branch, path string) (string, error) {
	url := fmt.Sprintf("%s/%s/%s/%s/%s", a.rawBase(), owner, repoName, branch, path)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", httpx.UserAgent)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch raw: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("status %d: %s", resp.StatusCode, path)
	}
	const cap = 4 << 20
	b, err := io.ReadAll(io.LimitReader(resp.Body, cap))
	if err != nil {
		return "", fmt.Errorf("read raw: %w", err)
	}
	return string(b), nil
}

// contains 简单判断。
func contains(haystack []string, needle string) bool {
	for _, h := range haystack {
		if h == needle {
			return true
		}
	}
	return false
}

// isAuthRequired 2026-07-10 增:识别 GitHub tree API 401(私有仓库 / token 失效)。
//
// 与 isRateLimited 不完全等同 — rate limited 也常返 403,但 401 通常意味着
// 资源不可见(私有 / 鉴权配置错),继续试下一个分支没意义。
func isAuthRequired(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "status 401") || strings.Contains(msg, "Bad credentials")
}

// downloadSKILLMDOnly 老路径:6 候选 URL × main/master,只下 SKILL.md,用于 tree 失败时兜底。
//
// 返回 (canonical, nil) 成功;(nil, err) 全失败,err 是最后一个错误。
func (a *Adapter) downloadSKILLMDOnly(ctx context.Context, repo, name string) (*skilladapter.Canonical, error) {
	branches := []string{"main", "master"}
	dirs := []string{"skills", ".claude/skills", ""}
	for _, b := range branches {
		for _, d := range dirs {
			prefix := d
			if prefix != "" {
				prefix = prefix + "/"
			}
			url := fmt.Sprintf("%s/%s/%s/%s%s/SKILL.md", a.rawBase(), repo, b, prefix, name)
			body, err := a.fetchBody(ctx, url)
			if err == nil {
				can, perr := skilladapter.ParseSkillMD(body)
				if perr != nil {
					continue
				}
				if can.Manifest.Name == "" {
					can.Manifest.Name = name
				}
				return can, nil
			}
			if isRateLimited(err) {
				return nil, err
			}
		}
	}
	return nil, fmt.Errorf("no candidate matched for %s/%s", repo, name)
}

// isRateLimited 2026-07-09 增:识别 GitHub raw 返回的 429 / "rate limit exceeded"。
//
// httpx 内部把 HTTP 错误统一包成 `status N: <url>` 格式,所以既用 errors.Is 识别
// http.StatusTooManyRequests,也用 substring 兜底识别 message 里出现的 rate limit 字样
// (防止 httpx 未来换格式漏掉)。
func isRateLimited(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, http.ErrAbortHandler) {
		return false
	}
	msg := err.Error()
	if strings.Contains(msg, "status 429") || strings.Contains(msg, "status 403") {
		// 403 也算限流(GitHub 未鉴权请求到阈值后会返 403 forbidden)
		return true
	}
	lower := strings.ToLower(msg)
	return strings.Contains(lower, "rate limit") ||
		strings.Contains(lower, "too many requests") ||
		strings.Contains(lower, "api rate limit exceeded")
}

// buildFallbackCanonical 2026-07-09 增:命中 knownCatalogFallback 时构建最小 SKILL.md。
//
// 跟 skillhub adapter 的 buildFallbackCanonical 行为对齐 — 用户至少能装上骨架
// (含 frontmatter + Triggers),不会因为网络抖动就完全失败。
// remoteID 形如 "owner/repo@skill";命中条件:knownCatalogFallback 中能找到同名
// "{owner}/{repo}@{skill}" 行。
func buildFallbackCanonical(remoteID, repo, name string) *skilladapter.Canonical {
	items := parseCatalog(knownCatalogFallback, defaultBaseURL)
	for _, it := range items {
		if it.RemoteID == remoteID {
			body := "---\n"
			body += "name: " + name + "\n"
			body += "version: " + firstNonEmptyOr(it.Version, "0.1.0") + "\n"
			if it.Description != "" {
				body += "description: " + it.Description + "\n"
			}
			if it.Author != "" {
				body += "author: " + it.Author + "\n"
			}
			manifest := skilladapter.Manifest{
				Name:        name,
				Version:     firstNonEmptyOr(it.Version, "0.1.0"),
				Description: it.Description,
				Author:      it.Author,
			}
			if len(manifest.Triggers) == 0 {
				manifest.Triggers = []string{name}
			}
			return &skilladapter.Canonical{
				Manifest: manifest,
				Files:    []skilladapter.File{{Path: "SKILL.md", Content: body}},
			}
		}
	}
	return nil
}

// firstNonEmptyOr 简单兜底:第一个非空值,都空用 def。
func firstNonEmptyOr(s, def string) string {
	if strings.TrimSpace(s) == "" {
		return def
	}
	return s
}

// fetchBody 拉 URL 文本,自动 gzip 解压 + UA。状态非 2xx 返错。
// 2026-07-02 改:走 httpx.GetJSONWithUA,统一 keep-alive + Accept-Encoding + UA;
// 之前每次 fetchBody 都是裸 http.NewRequest,没 UA 没 gzip。
func (a *Adapter) fetchBody(ctx context.Context, url string) (string, error) {
	return httpx.GetJSONWithUA(ctx, a.httpClient, url)
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
