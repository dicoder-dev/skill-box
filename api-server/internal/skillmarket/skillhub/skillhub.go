// Package skillhub 实现 skillhub.cn 适配器(草案)。
//
// skillhub.cn 暂未公开稳定的 JSON API。本适配器按 "best effort + 兜底" 模式实现:
//   - BaseURL 默认 https://skillhub.cn
//   - Discover: 优先尝试 GET /api/v1/skills(JSON 数组);失败 / 不可达时返回
//     一个 known-good fallback 列表,保证 UI 始终有内容可看
//   - Detail:   尝试 GET /api/v1/skills/<id>;失败走 fallback
//   - Download: 走 SKILL.md 路径 /api/v1/skills/<id>/skill.md;解析成 canonical
//
// 一旦 skillhub.cn 公开稳定 API,只需改 fetchBody 的 URL 拼装方式,
// 业务侧接口契约不变(skillmarket.MarketAdapter)。
package skillhub

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"ginp-api/internal/skilladapter"
	"ginp-api/internal/skillmarket"
	"ginp-api/pkg/logger"
)

const (
	defaultBaseURL = "https://skillhub.cn"
	apiPath        = "/api/v1"
)

// 兜底 skill 列表(skillhub.cn API 暂不可达时使用)。
var knownFallback = []skillmarket.MarketItem{
	{
		RemoteID:    "code-review",
		Name:        "code-review",
		Version:     "1.0.0",
		Author:      "skillhub",
		DetailURL:   defaultBaseURL + "/skills/code-review",
		Tags:        []string{"code-quality", "review"},
		Description: "对当前 diff 做静态代码审查,聚焦可读性与潜在 bug。",
	},
	{
		RemoteID:    "commit-msg",
		Name:        "commit-msg",
		Version:     "0.3.1",
		Author:      "skillhub",
		DetailURL:   defaultBaseURL + "/skills/commit-msg",
		Tags:        []string{"git", "commit"},
		Description: "根据 staged diff 自动生成符合 Conventional Commits 规范的提交信息。",
	},
	{
		RemoteID:    "debug-helper",
		Name:        "debug-helper",
		Version:     "0.2.0",
		Author:      "skillhub",
		DetailURL:   defaultBaseURL + "/skills/debug-helper",
		Tags:        []string{"debug", "diagnostic"},
		Description: "协助快速定位运行时错误与异常堆栈,给出最小可复现 + 修复建议。",
	},
}

// Adapter skillhub.cn 适配器。
type Adapter struct {
	httpClient *http.Client
}

// New 构造 Adapter(httpClient 为 nil 时用默认 20s 超时客户端)。
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

func (a *Adapter) SourceID() string    { return skillmarket.SourceSkillhub }
func (a *Adapter) DisplayName() string { return "SkillHub" }
func (a *Adapter) BaseURL() string     { return defaultBaseURL }

// Discover 拉目录。
func (a *Adapter) Discover(ctx context.Context, baseURL string) ([]skillmarket.MarketItem, error) {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	url := strings.TrimRight(baseURL, "/") + apiPath + "/skills"
	body, err := a.fetchBody(ctx, url)
	if err != nil {
		logger.Warn("skillhub discover: %v; falling back to known list", err)
		return cloneFallback(baseURL), nil
	}
	var raw []struct {
		ID          string   `json:"id"`
		Name        string   `json:"name"`
		Version     string   `json:"version"`
		Description string   `json:"description"`
		Author      string   `json:"author"`
		Tags        []string `json:"tags"`
		DetailURL   string   `json:"detail_url"`
	}
	if err := json.Unmarshal([]byte(body), &raw); err != nil {
		logger.Warn("skillhub discover unmarshal: %v; falling back", err)
		return cloneFallback(baseURL), nil
	}
	if len(raw) == 0 {
		return cloneFallback(baseURL), nil
	}
	out := make([]skillmarket.MarketItem, 0, len(raw))
	for _, it := range raw {
		if it.ID == "" {
			continue
		}
		detail := it.DetailURL
		if detail == "" {
			detail = fmt.Sprintf("%s/skills/%s", strings.TrimRight(baseURL, "/"), it.ID)
		}
		out = append(out, skillmarket.MarketItem{
			RemoteID:    it.ID,
			Name:        firstNonEmpty(it.Name, it.ID),
			Version:     it.Version,
			Description: it.Description,
			Author:      it.Author,
			Tags:        it.Tags,
			DetailURL:   detail,
		})
	}
	return out, nil
}

// Detail 拉详情。
func (a *Adapter) Detail(ctx context.Context, baseURL, remoteID string) (*skillmarket.MarketDetail, error) {
	if remoteID == "" {
		return nil, skillmarket.ErrEmptyRemoteID
	}
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	url := fmt.Sprintf("%s%s/skills/%s", strings.TrimRight(baseURL, "/"), apiPath, remoteID)
	body, err := a.fetchBody(ctx, url)
	if err != nil {
		// fallback:从 knownFallback 里找
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
	var raw struct {
		ID          string   `json:"id"`
		Name        string   `json:"name"`
		Version     string   `json:"version"`
		Description string   `json:"description"`
		Author      string   `json:"author"`
		License     string   `json:"license"`
		Tags        []string `json:"tags"`
		Homepage    string   `json:"homepage"`
		DetailURL   string   `json:"detail_url"`
	}
	if err := json.Unmarshal([]byte(body), &raw); err != nil {
		return nil, fmt.Errorf("%w: unmarshal: %v", skillmarket.ErrRemoteFetchFail, err)
	}
	if raw.ID == "" {
		return nil, fmt.Errorf("%w: %s", skillmarket.ErrRemoteNotFound, remoteID)
	}
	detail := raw.DetailURL
	if detail == "" {
		detail = fmt.Sprintf("%s/skills/%s", strings.TrimRight(baseURL, "/"), raw.ID)
	}
	return &skillmarket.MarketDetail{
		MarketItem: skillmarket.MarketItem{
			RemoteID:    raw.ID,
			Name:        firstNonEmpty(raw.Name, raw.ID),
			Version:     raw.Version,
			Description: raw.Description,
			Author:      raw.Author,
			Tags:        raw.Tags,
			DetailURL:   detail,
		},
		License:  raw.License,
		Homepage: raw.Homepage,
	}, nil
}

// Download 拉 SKILL.md 转 canonical。
func (a *Adapter) Download(ctx context.Context, baseURL, remoteID string) (*skilladapter.Canonical, error) {
	if remoteID == "" {
		return nil, skillmarket.ErrEmptyRemoteID
	}
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	url := fmt.Sprintf("%s%s/skills/%s/skill.md", strings.TrimRight(baseURL, "/"), apiPath, remoteID)
	body, err := a.fetchBody(ctx, url)
	if err != nil {
		// fallback:从 knownFallback 生成最小可用的 canonical
		for _, it := range knownFallback {
			if it.RemoteID == remoteID {
				return buildFallbackCanonical(it), nil
			}
		}
		return nil, fmt.Errorf("%w: %v", skillmarket.ErrRemoteFetchFail, err)
	}
	can, perr := skilladapter.ParseSkillMD(body)
	if perr != nil {
		return nil, fmt.Errorf("%w: %v", skillmarket.ErrRemoteFetchFail, perr)
	}
	return can, nil
}

// fetchBody 拉 URL,非 2xx 返错。
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
func buildFallbackCanonical(it skillmarket.MarketItem) *skilladapter.Canonical {
	body := "# " + it.Name + "\n\n" + it.Description + "\n"
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

// 注册到默认 registry。
func init() {
	skillmarket.Register(New())
}
