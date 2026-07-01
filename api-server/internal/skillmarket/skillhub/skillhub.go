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
	"strings"
	"time"

	"ginp-api/internal/skilladapter"
	"ginp-api/internal/skillmarket"
	"ginp-api/pkg/logger"
)

const (
	// 2026-07-01 改:对接 API host(原 https://skillhub.cn 是站点,无 API)
	defaultBaseURL = "https://api.skillhub.cn"
	// 默认 list 页大小(文档说最大 100)
	defaultPageSize = 100
	// 2026-07-01 改:skillhub 单页 pageSize 最大 100,接口不直接支持更大 pageSize。
	// 原实现 page=1 一次性只拿 100 条,被用户报"全网肯定不止 100 条"。
	// 改造:Discover 走 page 翻页,直到 total 拉满或达到 maxDiscoverItems 上限。
	// 上限 1000 条 — 兼顾 99% 用户浏览需求,防 API 翻车(限流)或单次响应太慢。
	maxDiscoverItems = 1000
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

// New 构造 Adapter(httpClient 为 nil 时用默认 30s 超时客户端)。
// 2026-07-01 改:timeout 20s → 30s(skillhub pageSize=100 单页慢,真 API 需要更长)。
func New() *Adapter {
	return &Adapter{
		httpClient:       &http.Client{Timeout: 30 * time.Second},
		noRedirectClient: &http.Client{Timeout: 30 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }},
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
	return &Adapter{
		httpClient:       c,
		noRedirectClient: c,
	}
}

// NewWithClients 构造 Adapter 分别注入 noRedirect 和普通 client(2026-07-01 增,
// 用于 zip flow 测试:noRedirect 走 fakeRT,httpClient 走真实 httptest.NewServer 拉 zip)。
func NewWithClients(noRedirect, normal *http.Client) *Adapter {
	if noRedirect == nil {
		noRedirect = &http.Client{Timeout: 30 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}
	}
	if normal == nil {
		normal = &http.Client{Timeout: 30 * time.Second}
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

	// 全量目录:翻页拉全量。
	seen := make(map[string]struct{}, maxDiscoverItems)
	out := make([]skillmarket.MarketItem, 0, maxDiscoverItems)
	totalHint := -1 // 0 = API 没回 total 字段;>0 = 上限
	stop := false
	for page := 1; !stop; page++ {
		// 首页带 sortBy/order(与原契约一致,测试与下游消费者都对齐)
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
			logger.Warn("skillhub discover page=%d: %v", page, err)
			break
		}
		var resp apiListResp
		if uerr := json.Unmarshal([]byte(body), &resp); uerr != nil {
			logger.Warn("skillhub discover page=%d unmarshal: %v", page, uerr)
			break
		}
		if resp.Code != 0 {
			logger.Warn("skillhub discover page=%d code=%d msg=%q", page, resp.Code, resp.Message)
			break
		}
		// 记录 totalHint:第一页拿一次即可
		if totalHint < 0 {
			totalHint = resp.Data.Total
		}
		if len(resp.Data.Skills) == 0 {
			// 空页:已无更多数据
			break
		}
		// 把当前页 map 成 MarketItem,合并去重,累加
		pageItems := mapSkillList(resp.Data.Skills, baseURL)
		added := 0
		for _, it := range pageItems {
			if _, dup := seen[it.RemoteID]; dup {
				continue
			}
			seen[it.RemoteID] = struct{}{}
			out = append(out, it)
			added++
			if len(out) >= maxDiscoverItems {
				stop = true
				break
			}
		}
		// 翻页退出条件 1:本页没新增(全重复,可能 total 字段不可信,防死循环)
		if added == 0 {
			break
		}
		// 翻页退出条件 2:已拿到 totalHint 的全部
		if totalHint > 0 && len(out) >= totalHint {
			break
		}
		// 翻页退出条件 3:本页不到 pageSize,后面也没了
		if len(resp.Data.Skills) < defaultPageSize {
			break
		}
		// 翻页退出条件 4:ctx 取消(交给上层 45s ctx 触发)
		if cerr := ctx.Err(); cerr != nil {
			logger.Warn("skillhub discover cancelled at page=%d: %v", page, cerr)
			break
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

// fetchBody 拉 URL 文本,状态非 2xx 返错;超时/网络错误一并返错。
func (a *Adapter) fetchBody(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "skill-box/1.0 (+https://skillbox.local)")
	req.Header.Set("Accept", "application/json,text/plain,*/*")
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
