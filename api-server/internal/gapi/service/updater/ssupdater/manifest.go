// Package ssupdater 提供 skill-box 应用自身的检测升级 / 下载 / SHA256 校验逻辑。
//
// 不依赖 gin / wails 任何专属概念,纯业务:
//   - FetchManifest    拉远端 manifest(多源超时降级)
//   - Compare          与本地版本对比(用 golang.org/x/mod/semver)
//   - Download         HTTP Range 下载,进度回调到状态对象
//   - Verify           SHA256 校验
//   - Helper           替身脚本入口字符串拼接(运行时由 desktop 子模块 fork)
//
// controller(skillbox/cdesktop/update.a.go)负责收 gin 请求、调用这里。
package ssupdater

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"ginp-api/pkg/logger"
)

// 一个新版发布的所有字段。UI / controller 都基于此结构传输。
// schema 详见 docs/agent/project/updater-manifest.md。
type Manifest struct {
	Channel     string             `json:"channel"`
	Version     string             `json:"version"`
	ReleasedAt  string             `json:"released_at"`
	MinSupport  string             `json:"min_supported"`
	Notes       map[string]string  `json:"notes"`
	Assets      []Asset            `json:"assets"`
}

type Asset struct {
	OS     string   `json:"os"`
	Arch   string   `json:"arch"`
	Kind   string   `json:"kind"`
	Size   int64    `json:"size"`
	SHA256 string   `json:"sha256"`
	URLs   []string `json:"urls"`
}

// 选择匹配当前 OS/Arch 的 asset;找不到返 nil。
func (m *Manifest) PickAsset(osName, arch string) *Asset {
	if m == nil {
		return nil
	}
	for i := range m.Assets {
		a := &m.Assets[i]
		if a.OS == osName && a.Arch == arch {
			return a
		}
	}
	// 兜底:同 OS 下不挑 Arch,留给前端二次匹配(后续可改严格)。
	for i := range m.Assets {
		a := &m.Assets[i]
		if a.OS == osName && (a.Arch == "" || a.Arch == arch) {
			return a
		}
	}
	return nil
}

// 客户端(全局单例即可)。
type Client struct {
	httpClient *http.Client
	mu         sync.Mutex
	cache      map[string]*cacheEntry
}

type cacheEntry struct {
	manifest  *Manifest
	expiresAt time.Time
}

func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 8 * time.Second, // 单源超时,内部多源顺序降级
		},
		cache: map[string]*cacheEntry{},
	}
}

// FetchManifest 从多源 urls 数组里按顺序拉,**严格按顺序试**(HEAD 探测太重,
// 直接 GET 一次 + 8s 超时,失败切下一个)。所有源都失败返 error。
//
// channel 参数本版本未使用,作为 cache key;MVP 阶段 manifest 不带 ?channel=。
func (c *Client) FetchManifest(ctx context.Context, urls []string, channel string) (*Manifest, error) {
	c.mu.Lock()
	if e, ok := c.cache[channel]; ok && e.manifest != nil && time.Now().Before(e.expiresAt) {
		c.mu.Unlock()
		return e.manifest, nil
	}
	c.mu.Unlock()
	if len(urls) == 0 {
		return nil, fmt.Errorf("updater: empty urls")
	}
	var lastErr error
	for _, raw := range urls {
		if raw == "" {
			continue
		}
		url := strings.TrimSpace(raw)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			lastErr = err
			continue
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("User-Agent", "skill-box-updater/1.0")
		resp, err := c.httpClient.Do(req)
		if err != nil {
			logger.Warn("updater: fetch manifest failed url=%s err=%v", url, err)
			lastErr = err
			continue
		}
		body, rerr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if rerr != nil || resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("status %d: %v", resp.StatusCode, rerr)
			continue
		}
		var m Manifest
		if err := json.Unmarshal(body, &m); err != nil {
			lastErr = fmt.Errorf("decode: %w", err)
			continue
		}
		if m.Version == "" {
			lastErr = fmt.Errorf("manifest missing version")
			continue
		}
		// 5 分钟缓存。MVP 期间避免反复拉,生产可降级到 1 分钟。
		c.mu.Lock()
		c.cache[channel] = &cacheEntry{manifest: &m, expiresAt: time.Now().Add(5 * time.Minute)}
		c.mu.Unlock()
		return &m, nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("updater: no valid url")
	}
	return nil, fmt.Errorf("updater: all urls failed: %w", lastErr)
}
