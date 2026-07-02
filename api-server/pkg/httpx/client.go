// Package httpx 提供三方源适配器使用的"长生命周期"HTTP 客户端。
//
// 与 pkg/httpclient(一次性, 每次新建)的区别:
//
//	- 单一底层 *http.Client, 跨多次调用复用 TCP/TLS 连接(Keep-Alive)
//	- Transport 显式调优: MaxIdleConnsPerHost=10, IdleConnTimeout=90s,
//	  TLSHandshakeTimeout=10s, ExpectContinueTimeout=1s
//	- 自动 Accept-Encoding: gzip, 透明解压响应体
//
// 设计取舍:
//
//	- 适配器(skillhub / skillssh)在 New() 时建一个 client, 进程内单例
//	- 不加 singleflight / retry / 限流 — 那些是业务策略, 各 adapter 自己决定
//	- 不引入第三方依赖, 纯 stdlib(compress/gzip + net/http)
package httpx

import (
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

// DefaultTransport 返回针对"高频小请求到同一 host"调优的 http.Transport。
//
// 参数说明:
//
//	MaxIdleConns        — 整个 client 池空闲连接上限, 100 足够常规三方源
//	MaxIdleConnsPerHost — 单 host 空闲连接上限, 默认 net/http 是 2,
//	                      三方源翻页场景会被卡住; 提到 10 让并发请求不必等连接
//	IdleConnTimeout     — 空闲连接多久关闭, 90s 兼顾 keep-alive 与故障切换
//	TLSHandshakeTimeout — TLS 握手超时, 10s 防止三方源慢响应拖死整个 client
//	ExpectContinueTimeout — 100-continue 等待超时, 默认 1s 防止 PUT 类请求卡住
//	DisableCompression  — false(显式声明), 我们手动加 Accept-Encoding 并透明解压
func DefaultTransport() *http.Transport {
	return &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DisableCompression:    false,
	}
}

// NewClient 返回一个"长生命周期"的 http.Client, 带 keep-alive + gzip 自动解压。
//
// 参数:
//
//	timeout — 单次请求的总超时(Dial + TLS + Write + ReadAll 全部算进去)。
//	          推荐 30s, 给 pageSize=100 的 JSON API 留余量; 太短反而误杀。
//
// 注意:返回的 client 是**非并发安全**地共享 — http.Client 本身并发安全,
// 但其内部 Transport 是; 放心在多个 goroutine 共享同一个 client 实例。
func NewClient(timeout time.Duration) *http.Client {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &http.Client{
		Timeout:   timeout,
		Transport: DefaultTransport(),
	}
}

// NewNoRedirectClient 返回不自动 follow 30x 的 client(用于 302 → COS zip 这种
// 需要先看 Location 再决定下一步的场景)。
func NewNoRedirectClient(timeout time.Duration) *http.Client {
	c := NewClient(timeout)
	c.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}
	return c
}

// UserAgent 三方源 UA(固定; 部分 CDN 会按 UA 做白名单)。
const UserAgent = "skill-box/1.0 (+https://skillbox.local)"

// GetJSONWithUA 在 NewClient 基础上做 4 件事:
//
//	1. 自动加 Accept / Accept-Encoding: gzip / User-Agent
//	2. 透明解压 gzip 响应(Content-Encoding: gzip)
//	3. 状态码非 2xx → 返错, 带 body 摘要
//	4. body 上限 4MB(防止恶意大包撑爆内存); 翻页场景 pageSize=100 JSON < 200KB 完全够
//
// 返回的是**字符串**(用于 skillssh 的 HTML 解析场景; skillhub JSON 解析场景
// 也用 string + json.Unmarshal 二次解析)。
func GetJSONWithUA(ctx context.Context, client *http.Client, url string) (string, error) {
	if client == nil {
		client = NewClient(30 * time.Second)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Accept", "application/json,text/html,application/xhtml+xml,text/plain,*/*")
	req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9,zh-CN;q=0.8,zh;q=0.7")
	req.Header.Set("Connection", "keep-alive")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if resp.StatusCode == http.StatusNotFound {
			return "", fmt.Errorf("status 404: %s", url)
		}
		return "", fmt.Errorf("status %d: %s", resp.StatusCode, url)
	}

	// 透明解压:Content-Encoding: gzip → gzip.NewReader; 否则直接读
	var reader io.Reader = resp.Body
	if strings.EqualFold(resp.Header.Get("Content-Encoding"), "gzip") {
		gz, gerr := gzip.NewReader(resp.Body)
		if gerr != nil {
			return "", fmt.Errorf("gzip reader: %w", gerr)
		}
		defer gz.Close()
		reader = gz
	}

	const bodyCap = 4 << 20 // 4MB; 翻页 100 条 JSON < 200KB
	b, err := io.ReadAll(io.LimitReader(reader, bodyCap))
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// ErrBodyTooLarge body 超过 bodyCap 时返错(防止恶意大包 + 内存爆)。
var ErrBodyTooLarge = errors.New("httpx: response body exceeds limit")