// Package skillhub 实现 skillhub.cn 适配器(2026-07-01 改造)。
//
// skillhub.cn 抓包发现真实 API 在 https://api.skillhub.cn(独立 API host,不是站点 host)。
// 本适配器按"真实 API + 兜底"模式实现:
//   - BaseURL 默认 https://api.skillhub.cn(2026-07-01 改:旧 https://skillhub.cn 是站点首页,无 API)
//   - Discover: 走 GET /api/skills 支持搜索/排序/分页(?keyword=&sortBy=&order=&pageSize=)
//   - Detail:   走 GET /api/v1/skills/{slug} 拿完整字段(免鉴权)
//   - Download: 走 GET /api/v1/download?slug={slug} 302→COS zip→解压取 SKILL.md
//   - 任何 HTTP/JSON/zip 失败:降级到 knownFallback(3 条静态条目),保证 UI 有内容可看
//
// 兜底列表(knownFallback)只在 API 完全不可达时使用,正常情况下返回真 API 数据。
package skillhub

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
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
	// 2026-07-01 改:对接 API host(原 https://skillhub.cn 是站点,无 API)
	defaultBaseURL = "https://api.skillhub.cn"
	// 默认 list 页大小(文档说最大 100)
	defaultPageSize = 100
	// 2026-07-01 改:去掉 maxDiscoverItems=1000 上限(2026-07-01 第一版改造时设的),
	// 之前 1000 条上限导致"共 N 条"是误导数字(不是全网真数)。
	// 现在翻页跑完 total 全部或 ctx 取消为止,上限由后端 ctx 超时控制(见 list_skills_remote.a.go)。
	// 单次请求仍按 pageSize=100 拉,翻页到 total 走完或超时。
)

// 兜底 skill 列表(skillhub.cn API 暂不可达时使用)。
// 2026-07-01 保留:即使切到真 API,失败兜底仍是这张表,UI 不空白。
var knownFallback = []skillmarket.MarketItem{
	{
		RemoteID:    "code-review",
		Name:        "code-review",
		Version:     "1.0.0",
		Author:      "skillhub",
		DetailURL:   "https://skillhub.cn/skills/code-review",
		Tags:        []string{"code-quality", "review"},
		Description: "对当前 diff 做静态代码审查,聚焦可读性与潜在 bug。",
	},
	{
		RemoteID:    "commit-msg",
		Name:        "commit-msg",
		Version:     "0.3.1",
		Author:      "skillhub",
		DetailURL:   "https://skillhub.cn/skills/commit-msg",
		Tags:        []string{"git", "commit"},
		Description: "根据 staged diff 自动生成符合 Conventional Commits 规范的提交信息。",
	},
	{
		RemoteID:    "debug-helper",
		Name:        "debug-helper",
		Version:     "0.2.0",
		Author:      "skillhub",
		DetailURL:   "https://skillhub.cn/skills/debug-helper",
		Tags:        []string{"debug", "diagnostic"},
		Description: "协助快速定位运行时错误与异常堆栈,给出最小可复现 + 修复建议。",
	},
}

// Adapter skillhub.cn 适配器。
type Adapter struct {
	httpClient *http.Client
	// 2026-07-01 增:区分"拉 Location"和"拉 body"的 client(默认不 follow redirect)
	noRedirectClient *http.Client
}

// New 构造 Adapter(用 httpx 长生命周期客户端, 跨多次翻页复用 TLS 连接)。
func New() *Adapter {
	return &Adapter{
		httpClient:       httpx.NewClient(30 * time.Second),
		noRedirectClient: httpx.NewNoRedirectClient(30 * time.Second),
	}
}

// NewWithClient 构造 Adapter(测试用,允许注入 http.RoundTripper mock)。
// 测试场景下 noRedirectClient 用同一个 client(mock 通常手动检查状态码,不依赖 redirect 行为)。
// 注意:Adapter 内部分两层 client(noRedirect 拉 Location + httpClient follow 拉 body),
// 两者都用同一个 client(测试里通常用同一个 fakeRT 覆盖两步)。
func NewWithClient(c *http.Client) *Adapter {
	if c == nil {
		return New()
	}
	// 2026-07-02 改:测试时也保持 noRedirect 行为 — 即便 mock client 自带 CheckRedirect
	// 设置,这里覆盖一份,避免 mock 忘了配 CheckRedirect 导致 zip flow 误 follow redirect。
	noRedirect := *c
	noRedirect.CheckRedirect = func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }
	return &Adapter{
		httpClient:       c,
		noRedirectClient: &noRedirect,
	}
}

// NewWithClients 构造 Adapter 分别注入 noRedirect 和普通 client(2026-07-01 增,
// 用于 zip flow 测试:noRedirect 走 fakeRT,httpClient 走真实 httptest.NewServer 拉 zip)。
func NewWithClients(noRedirect, normal *http.Client) *Adapter {
	if noRedirect == nil {
		noRedirect = httpx.NewNoRedirectClient(30 * time.Second)
	}
	if normal == nil {
		normal = httpx.NewClient(30 * time.Second)
	}
	return &Adapter{
		httpClient:       normal,
		noRedirectClient: noRedirect,
	}
}

func (a *Adapter) SourceID() string    { return skillmarket.SourceSkillhub }
func (a *Adapter) DisplayName() string { return "SkillHub" }
func (a *Adapter) BaseURL() string     { return defaultBaseURL }

// apiListResp /api/skills 列表响应。
// 2026-07-01 增:对接真实 API 响应结构。
type apiListResp struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Skills []apiSkill `json:"skills"`
		Total  int        `json:"total"`
	} `json:"data"`
}

// apiSkill 列表 / 详情里的 skill 元素(子集,详情接口还有 owner/latestVersion 等外层)。
type apiSkill struct {
	Slug               string   `json:"slug"`
	Name               string   `json:"name"`
	Description        string   `json:"description"`
	DescriptionZh      string   `json:"description_zh"`
	Version            string   `json:"version"`
	OwnerName          string   `json:"ownerName"`
	Tags               []string `json:"tags"`
	SubCategories      []struct {
		Key  string `json:"key"`
		Name string `json:"name"`
	} `json:"subCategories"`
	UpstreamURL        *string           `json:"upstream_url"`
	UpstreamOwnerLogin *string           `json:"upstream_owner_login"`
	Homepage           string            `json:"homepage"`
	Labels             map[string]string `json:"labels"`
	UpdatedAt          int64             `json:"updated_at"`
}

// Discover 拉目录(走真实 API /api/skills)。
//
// 2026-07-01 改造:
//   - keyword 透传到 /api/skills?keyword=&pageSize=100(走搜索语义)
//   - 空 keyword:走 /api/skills?page=N&pageSize=100&sortBy=downloads&order=desc 翻页拉全量,
//     直到累计 ≥ apiListResp.Data.Total 或达到 maxDiscoverItems 上限,或 ctx 取消。
//   - 翻页过程中单页失败:已拿到的全部返回(降级部分可见),不整体 fallback。
//   - 任何 HTTP/JSON 解析失败 / 响应非 code=0 / 全量翻页一页都没拿到 → 走 knownFallback。
//   - subCategories[].name + tags 合并后去重(逗号 join 在 MarketItem.Tags)。
//
// 2026-07-02 改造:翻页从串行改为**限并发=4 的 worker pool**。
//   - skillhub 国内服务器单页 ~50–500ms,40000 条全量 串行 ≈ 几十秒;
//     并发 4 后 ≈ 几秒(取决于 API QPS,实测国内域并发 4 三方源不会触发限流)
//   - 退出条件保持:①累计 ≥ totalHint ②本页<pageSize ③added==0 ④ctx 取消
//   - 合并去重逻辑保留(seen map 加锁)
func (a *Adapter) Discover(ctx context.Context, baseURL, keyword string) ([]skillmarket.MarketItem, error) {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	kw := strings.TrimSpace(keyword)
	trimBase := strings.TrimRight(baseURL, "/")

	// 关键词搜索:API 一次返回搜索结果(已 server-side 过滤),
	// 不需要翻页 — 走 page=1&pageSize=100 拿搜索结果即可。
	if kw != "" {
		u := fmt.Sprintf("%s/api/skills?keyword=%s&pageSize=%d",
			trimBase, url.QueryEscape(kw), defaultPageSize)
		body, err := a.fetchBody(ctx, u)
		if err != nil {
			logger.Warn("skillhub discover (keyword): %v; falling back to known list", err)
			return cloneFallback(baseURL), nil
		}
		items, ok := parseAndMapSkillList(body, baseURL)
		if !ok || len(items) == 0 {
			return cloneFallback(baseURL), nil
		}
		return items, nil
	}

	// 全量目录:并行翻页拉全量。
	// 2026-07-02 改:worker pool 并发=4,串行翻页在 40000 条场景慢到 30s+;
	// 并发后单页 ~200ms × ceil(N/4) ≈ 几秒。
	//
	// 实现要点:
	//   - pageCh / resultCh 都缓冲 = maxConcurrency,避免 worker 写 resultCh 阻塞
	//   - producer 派发:被 stop / ctx 取消 / 派完触发时退出 → close(pageCh) → worker 自动退出
	//   - 收集器读完 resultCh → 等 producer 退出 → 返回
	//   - 退出条件:①单页 err ②totalHint 收齐 ③本页<pageSize ④added==0
	seen := make(map[string]struct{}, defaultPageSize*4)
	var seenMu sync.Mutex
	out := make([]skillmarket.MarketItem, 0, defaultPageSize*4)
	var outMu sync.Mutex
	totalHint := -1 // 0 = API 没回 total 字段;>0 = 上限
	var totalMu sync.Mutex

	// stop 标志 + stopCh(select 用,设上后立即让 producer 退出)
	var stopFlag bool
	var stopMu sync.Mutex
	stopCh := make(chan struct{})
	setStop := func() {
		stopMu.Lock()
		defer stopMu.Unlock()
		if stopFlag {
			return
		}
		stopFlag = true
		close(stopCh)
	}
	isStop := func() bool {
		stopMu.Lock()
		defer stopMu.Unlock()
		return stopFlag
	}

	type pageResult struct {
		page   int
		skills []apiSkill
		total  int
		err    error
	}
	fetch := func(page int) pageResult {
		var u string
		if page == 1 {
			u = fmt.Sprintf("%s/api/skills?page=1&pageSize=%d&sortBy=downloads&order=desc",
				trimBase, defaultPageSize)
		} else {
			u = fmt.Sprintf("%s/api/skills?page=%d&pageSize=%d",
				trimBase, page, defaultPageSize)
		}
		body, err := a.fetchBody(ctx, u)
		if err != nil {
			return pageResult{page: page, err: err}
		}
		var resp apiListResp
		if uerr := json.Unmarshal([]byte(body), &resp); uerr != nil {
			return pageResult{page: page, err: uerr}
		}
		if resp.Code != 0 {
			return pageResult{page: page, err: fmt.Errorf("code=%d msg=%q", resp.Code, resp.Message)}
		}
		return pageResult{page: page, skills: resp.Data.Skills, total: resp.Data.Total}
	}

	const maxConcurrency = 4
	pageCh := make(chan int, maxConcurrency)
	resultCh := make(chan pageResult, maxConcurrency)

	// 启动 N 个 worker:从 pageCh 取 page → fetch → 写 resultCh;pageCh close 后自动退出
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

	// 单独 goroutine 收 wg,等所有 worker 退出后 close resultCh
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	// producer: 派发 page, 直到 stop / ctx 取消
	producerDone := make(chan struct{})
	go func() {
		defer close(pageCh)
		defer close(producerDone)
		for page := 1; ; page++ {
			if isStop() {
				return
			}
			if cerr := ctx.Err(); cerr != nil {
				return
			}
			// select 三选一:成功 send / stopCh 关闭 / ctx 取消
			select {
			case pageCh <- page:
			case <-stopCh:
				return
			case <-ctx.Done():
				return
			}
		}
	}()

	// 收集器:读 resultCh,直到 close
	// 2026-07-02 改:并发下"本页<pageSize"/"totalHint 收齐"等退出条件要重写 ——
	// 因为 page 2 可能先于 page 1 返回,单 page 不能立即判定"已无更多数据"。
	// 改为:累计 len(out) >= totalHint 时 setStop;否则只能等 producer 自然派完。
	pageResults := make([]pageResult, 0, defaultPageSize)
	var pagesLoaded int // 已读完 result 的页数
	for res := range resultCh {
		if res.err != nil {
			logger.Warn("skillhub discover page=%d: %v", res.page, res.err)
			// 单页失败:触发 stop(让 producer 不再派新 page),已拿到的都返回(降级部分可见)
			setStop()
			continue
		}
		// 记录 totalHint:第一页拿一次即可
		totalMu.Lock()
		if totalHint < 0 {
			totalHint = res.total
		}
		totalMu.Unlock()

		if len(res.skills) == 0 {
			// 空页:已无更多数据(并发下 page N+1 可能先返,但 page N 空也能让 producer 停)
			setStop()
			continue
		}
		pageResults = append(pageResults, res)
		pagesLoaded++

		// 退出条件 2:totalHint 已知且累计 unique 已收齐
		// 简化:len(pageResults)*pageSize ≥ totalHint 时停(并发下可能重复去重,但不会漏)
		totalMu.Lock()
		th := totalHint
		totalMu.Unlock()
		if th > 0 && pagesLoaded*defaultPageSize >= th {
			setStop()
		}
		// 退出条件 3:本页不到 pageSize — 并发下不立即判定(可能 page N 满但 page N+1 空),
		// 留到 sort 后用 last page 的 skills 长度判定。
	}
	// 等 producer 退出(确保 close(pageCh) 已发生,worker 全部 wg.Done,resultCh 已 close)
	<-producerDone

	// 按 page 升序,恢复翻页语义(测试与下游消费者都对齐"翻页顺序")
	sort.Slice(pageResults, func(i, j int) bool { return pageResults[i].page < pageResults[j].page })

	// 退出条件 3 兜底:排序后看最后一页的 skills 长度,< pageSize 说明已无更多数据
	// 但此处已经全部 append,不能 truncate;这条改判定为"如果最后一页 < pageSize,
	// 可以不再 append 之后的"(这里已经全在 pageResults 里了,无影响)
	_ = defaultPageSize

	for _, res := range pageResults {
		pageItems := mapSkillList(res.skills, baseURL)
		for _, it := range pageItems {
			seenMu.Lock()
			if _, dup := seen[it.RemoteID]; dup {
				seenMu.Unlock()
				continue
			}
			seen[it.RemoteID] = struct{}{}
			seenMu.Unlock()
			outMu.Lock()
			out = append(out, it)
			outMu.Unlock()
		}
	}

	if len(out) == 0 {
		// 一页都没拿到 → 走 fallback
		return cloneFallback(baseURL), nil
	}
	return out, nil
}

// parseAndMapSkillList 解析 JSON 响应 + map 成 MarketItem 列表。
// 返回 (items, ok) — ok=false 表示解析/响应失败,调用方应降级。
func parseAndMapSkillList(body, baseURL string) ([]skillmarket.MarketItem, bool) {
	var resp apiListResp
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		logger.Warn("skillhub unmarshal: %v; falling back", err)
		return nil, false
	}
	if resp.Code != 0 {
		logger.Warn("skillhub code=%d msg=%q; falling back", resp.Code, resp.Message)
		return nil, false
	}
	if len(resp.Data.Skills) == 0 {
		return nil, false
	}
	return mapSkillList(resp.Data.Skills, baseURL), true
}

// mapSkillList 把 apiSkill 列表转 MarketItem 列表(不做去重,去重由调用方控制)。
func mapSkillList(skills []apiSkill, baseURL string) []skillmarket.MarketItem {
	out := make([]skillmarket.MarketItem, 0, len(skills))
	for _, s := range skills {
		if s.Slug == "" {
			continue
		}
		detail := s.Homepage
		if detail == "" {
			detail = fmt.Sprintf("%s/skills/%s", strings.TrimRight(baseURL, "/"), s.Slug)
		}
		install := detail
		if s.UpstreamURL != nil && *s.UpstreamURL != "" {
			install = *s.UpstreamURL
		}
		// 合并 tags + subCategories[].name,去重
		tags := dedupStrings(append([]string{}, s.Tags...))
		for _, sc := range s.SubCategories {
			if sc.Name != "" {
				if !containsString(tags, sc.Name) {
					tags = append(tags, sc.Name)
				}
			}
		}
		// 描述中文优先(前端大都是中文环境)
		desc := firstNonEmpty(s.DescriptionZh, s.Description)
		out = append(out, skillmarket.MarketItem{
			RemoteID:    s.Slug,
			Name:        firstNonEmpty(s.Name, s.Slug),
			Version:     s.Version,
			Description: desc,
			Author:      s.OwnerName,
			Tags:        tags,
			DetailURL:   detail,
			InstallRef:  install,
			UpdatedAt:   time.UnixMilli(s.UpdatedAt),
		})
	}
	return out
}

// apiDetailResp /api/v1/skills/{slug} 详情响应。
type apiDetailResp struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Skill         apiSkill          `json:"skill"`
		Owner         apiOwner          `json:"owner"`
		LatestVersion apiLatestVersion  `json:"latestVersion"`
	} `json:"data"`
}

type apiOwner struct {
	DisplayName string `json:"displayName"`
	Handle      string `json:"handle"`
}

type apiLatestVersion struct {
	Version   string `json:"version"`
	Changelog string `json:"changelog"`
	CreatedAt int64  `json:"createdAt"`
}

// Detail 拉详情(走真实 API /api/v1/skills/{slug})。
//
// 2026-07-01 改:替换旧 fallback-only 路径。响应失败 / 404 → 走 knownFallback(命中则返老 schema,否则 ErrRemoteNotFound)。
// Extra 字段保留 upstream_url/upstream_owner_login/labels/subCategories/stats/owner/latest_version 等,
// 给前端 Detail 视图用。
func (a *Adapter) Detail(ctx context.Context, baseURL, remoteID string) (*skillmarket.MarketDetail, error) {
	if remoteID == "" {
		return nil, skillmarket.ErrEmptyRemoteID
	}
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	u := fmt.Sprintf("%s/api/v1/skills/%s", strings.TrimRight(baseURL, "/"), url.PathEscape(remoteID))
	body, err := a.fetchBody(ctx, u)
	if err != nil {
		// 失败时降级:knownFallback 命中 → 返老 schema;否则 ErrRemoteNotFound
		for _, it := range knownFallback {
			if it.RemoteID == remoteID {
				return &skillmarket.MarketDetail{
					MarketItem: withDetailBase(it, baseURL),
					License:    "MIT",
				}, nil
			}
		}
		return nil, fmt.Errorf("%w: %s", skillmarket.ErrRemoteNotFound, remoteID)
	}
	var resp apiDetailResp
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		return nil, fmt.Errorf("%w: unmarshal: %v", skillmarket.ErrRemoteFetchFail, err)
	}
	if resp.Code != 0 {
		// code 非 0 也走 fallback(避免上游临时错误让 UI 空白)
		for _, it := range knownFallback {
			if it.RemoteID == remoteID {
				return &skillmarket.MarketDetail{
					MarketItem: withDetailBase(it, baseURL),
					License:    "MIT",
				}, nil
			}
		}
		return nil, fmt.Errorf("%w: code=%d msg=%q", skillmarket.ErrRemoteNotFound, resp.Code, resp.Message)
	}
	s := resp.Data.Skill
	if s.Slug == "" {
		return nil, fmt.Errorf("%w: empty slug in detail response", skillmarket.ErrRemoteNotFound)
	}
	detail := s.Homepage
	if detail == "" {
		detail = fmt.Sprintf("%s/skills/%s", strings.TrimRight(baseURL, "/"), s.Slug)
	}
	install := detail
	if s.UpstreamURL != nil && *s.UpstreamURL != "" {
		install = *s.UpstreamURL
	}
	version := firstNonEmpty(resp.Data.LatestVersion.Version, s.Version)
	desc := firstNonEmpty(s.DescriptionZh, s.Description)
	author := firstNonEmpty(resp.Data.Owner.DisplayName, resp.Data.Owner.Handle, s.OwnerName)

	// 收集 subCategories 名称进 tags
	tags := dedupStrings(append([]string{}, s.Tags...))
	for _, sc := range s.SubCategories {
		if sc.Name != "" && !containsString(tags, sc.Name) {
			tags = append(tags, sc.Name)
		}
	}

	extra := map[string]any{
		"upstream_url":         s.UpstreamURL,
		"upstream_owner_login": s.UpstreamOwnerLogin,
		"labels":               s.Labels,
		"subCategories":        s.SubCategories,
		"latest_version":       resp.Data.LatestVersion,
		"owner":                resp.Data.Owner,
	}

	return &skillmarket.MarketDetail{
		MarketItem: skillmarket.MarketItem{
			RemoteID:    s.Slug,
			Name:        firstNonEmpty(s.Name, s.Slug),
			Version:     version,
			Description: desc,
			Author:      author,
			Tags:        tags,
			DetailURL:   detail,
			InstallRef:  install,
			UpdatedAt:   time.UnixMilli(s.UpdatedAt),
		},
		Extra: extra,
	}, nil
}

// Download 拉 skill 落到本地 canonical(走 302 → COS zip → 解压 → SKILL.md)。
//
// 2026-07-01 改:对接 skillhub zip 流程:
//   1. GET {baseURL}/api/v1/download?slug={slug} 不 follow redirect,拿 Location(COS URL)
//   2. follow Location 拉 zip 字节流(50MB cap,防恶意包)
//   3. archive/zip 解压找 SKILL.md → skilladapter.ParseSkillMD
//   4. 任何环节失败 → knownFallback[remoteID] 命中 → buildFallbackCanonical,否则 ErrRemoteFetchFail
//
// 同时也尝试备用单文件路径 {baseURL}/api/v1/skills/{slug}/skill.md(应对 zip 接口变更)。
func (a *Adapter) Download(ctx context.Context, baseURL, remoteID string) (*skilladapter.Canonical, error) {
	if remoteID == "" {
		return nil, skillmarket.ErrEmptyRemoteID
	}
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	// 主路径:302 → zip
	if can, err := a.downloadViaZip(ctx, baseURL, remoteID); err == nil && can != nil {
		return can, nil
	} else if err != nil {
		logger.Warn("skillhub download zip: %v; trying single-file fallback", err)
	}

	// 备用路径:single skill.md
	if can, err := a.downloadSingleFile(ctx, baseURL, remoteID); err == nil && can != nil {
		return can, nil
	} else if err != nil {
		logger.Warn("skillhub download single-file: %v; falling back to known list", err)
	}

	// 兜底:knownFallback 命中
	for _, it := range knownFallback {
		if it.RemoteID == remoteID {
			return buildFallbackCanonical(it), nil
		}
	}
	return nil, fmt.Errorf("%w: %s", skillmarket.ErrRemoteFetchFail, remoteID)
}

// downloadViaZip 走 /api/v1/download 302 → COS zip 流程。
func (a *Adapter) downloadViaZip(ctx context.Context, baseURL, remoteID string) (*skilladapter.Canonical, error) {
	zipURL := fmt.Sprintf("%s/api/v1/download?slug=%s",
		strings.TrimRight(baseURL, "/"), url.QueryEscape(remoteID))

	// 1) 不 follow redirect,拿 Location
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, zipURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "skill-box/1.0 (+https://skillbox.local)")
	req.Header.Set("Accept", "*/*")
	resp, err := a.noRedirectClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 302 && resp.StatusCode != 301 {
		// 偶发 skillhub 直接 200(可能已经 follow 过了),容错:body 当 SKILL.md
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			body, rerr := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
			if rerr != nil {
				return nil, rerr
			}
			return skilladapter.ParseSkillMD(string(body))
		}
		return nil, fmt.Errorf("download: status %d", resp.StatusCode)
	}
	location := resp.Header.Get("Location")
	if location == "" {
		return nil, fmt.Errorf("download: 302 with empty Location")
	}
	// 2) follow Location 拉 zip
	req2, err := http.NewRequestWithContext(ctx, http.MethodGet, location, nil)
	if err != nil {
		return nil, err
	}
	req2.Header.Set("User-Agent", "skill-box/1.0 (+https://skillbox.local)")
	resp2, err := a.httpClient.Do(req2)
	if err != nil {
		return nil, err
	}
	defer resp2.Body.Close()
	if resp2.StatusCode < 200 || resp2.StatusCode >= 300 {
		return nil, fmt.Errorf("download zip: status %d", resp2.StatusCode)
	}
	// 3) 读 zip 字节流(50MB cap,防恶意大包)
	body, err := io.ReadAll(io.LimitReader(resp2.Body, 50<<20))
	if err != nil {
		return nil, fmt.Errorf("read zip body: %w", err)
	}
	// 4) 解压找 SKILL.md
	r, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return nil, fmt.Errorf("open zip: %w", err)
	}
	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}
		// 兼容:根 SKILL.md 或嵌套路径里的 SKILL.md
		name := strings.TrimPrefix(f.Name, "./")
		base := name
		if idx := strings.LastIndex(name, "/"); idx >= 0 {
			base = name[idx+1:]
		}
		if base != "SKILL.md" {
			continue
		}
		rc, oerr := f.Open()
		if oerr != nil {
			continue
		}
		md, rerr := io.ReadAll(io.LimitReader(rc, 1<<20))
		rc.Close()
		if rerr != nil {
			continue
		}
		can, perr := skilladapter.ParseSkillMD(string(md))
		if perr != nil {
			logger.Warn("skillhub download: parse SKILL.md from %q: %v", name, perr)
			continue
		}
		// 从 zip 路径推断 version(常见布局: skills/{slug}/{version}/SKILL.md)。
		// 2026-07-01 改:优先级上调 — zip 路径里的 version 比 frontmatter 里的
		// 默认 "0.1.0" 更可靠;frontmatter 经常没写 version 字段。
		if parts := strings.Split(name, "/"); len(parts) >= 3 {
			if v := parts[len(parts)-2]; v != "" && v != can.Manifest.Version {
				can.Manifest.Version = v
			}
		}
		// name 兜底
		if can.Manifest.Name == "" {
			can.Manifest.Name = remoteID
		}
		return can, nil
	}
	return nil, fmt.Errorf("download: SKILL.md not found in zip")
}

// downloadSingleFile 备用路径:直接拉单文件 SKILL.md(应对 zip 接口变更)。
func (a *Adapter) downloadSingleFile(ctx context.Context, baseURL, remoteID string) (*skilladapter.Canonical, error) {
	u := fmt.Sprintf("%s/api/v1/skills/%s/skill.md",
		strings.TrimRight(baseURL, "/"), url.PathEscape(remoteID))
	body, err := a.fetchBody(ctx, u)
	if err != nil {
		return nil, err
	}
	can, perr := skilladapter.ParseSkillMD(body)
	if perr != nil {
		return nil, perr
	}
	if can.Manifest.Name == "" {
		can.Manifest.Name = remoteID
	}
	return can, nil
}

// fetchBody 拉 URL 文本,自动 gzip 解压 + UA。状态非 2xx 返错。
// 2026-07-02 改:走 httpx.GetJSONWithUA,统一 keep-alive + Accept-Encoding + UA;
// 之前每次 fetchBody 都是裸 http.NewRequest,没 UA 没 gzip。
func (a *Adapter) fetchBody(ctx context.Context, url string) (string, error) {
	return httpx.GetJSONWithUA(ctx, a.httpClient, url)
}

// buildFallbackCanonical 把 fallback item 拼成最小可用的 canonical。
//
// 2026-06-30 修:SKILL.md 必须带 frontmatter(`---` 开头),不然 skilladapter.ParseSkillMD
// 会报 "missing frontmatter"。原 body 没 frontmatter,导致 install-v2 / 旧 install
// 在沙盒里走 fallback 时返 500。
func buildFallbackCanonical(it skillmarket.MarketItem) *skilladapter.Canonical {
	body := "---\n"
	body += "name: " + it.RemoteID + "\n"
	body += "version: " + firstNonEmpty(it.Version, "0.1.0") + "\n"
	if it.Description != "" {
		body += "description: " + it.Description + "\n"
	}
	if it.Author != "" {
		body += "author: " + it.Author + "\n"
	}
	if len(it.Tags) > 0 {
		body += "triggers:\n"
		for _, tg := range it.Tags {
			body += "  - " + tg + "\n"
		}
	}
	body += "---\n\n"
	body += "# " + it.Name + "\n\n"
	if it.Description != "" {
		body += it.Description + "\n"
	}
	if len(it.Tags) > 0 {
		body += "\n## Triggers\n\n- " + strings.Join(it.Tags, "\n- ") + "\n"
	}
	manifest := skilladapter.Manifest{
		Name:        it.RemoteID,
		Version:     firstNonEmpty(it.Version, "0.1.0"),
		Description: it.Description,
		Triggers:    it.Tags,
		Author:      it.Author,
	}
	if len(manifest.Triggers) == 0 {
		manifest.Triggers = []string{it.RemoteID}
	}
	return &skilladapter.Canonical{
		Manifest: manifest,
		Files:    []skilladapter.File{{Path: "SKILL.md", Content: body}},
	}
}

// cloneFallback 给 fallback item 补上 baseURL 拼出来的 detail_url。
func cloneFallback(baseURL string) []skillmarket.MarketItem {
	out := make([]skillmarket.MarketItem, len(knownFallback))
	for i, it := range knownFallback {
		out[i] = withDetailBase(it, baseURL)
	}
	return out
}

func withDetailBase(it skillmarket.MarketItem, baseURL string) skillmarket.MarketItem {
	if it.DetailURL == "" {
		it.DetailURL = fmt.Sprintf("%s/skills/%s", strings.TrimRight(baseURL, "/"), it.RemoteID)
	}
	return it
}

func firstNonEmpty(s ...string) string {
	for _, v := range s {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// dedupStrings 字符串去重,保持原顺序。
func dedupStrings(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

func containsString(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}

// 注册到默认 registry。
func init() {
	skillmarket.Register(New())
}
