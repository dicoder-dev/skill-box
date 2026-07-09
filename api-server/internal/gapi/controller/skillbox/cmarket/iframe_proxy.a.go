// 市场 iframe 代理 — 反向代理三方站点页面资源,响应里抹掉 X-Frame-Options 与
// Content-Security-Policy 头,让前端 iframe 能正常加载(2026-07-09 增)。
//
// 为什么需要这个端点:
//   skillhub.cn 返回 `X-Frame-Options: SAMEORIGIN`,浏览器会拒绝在跨源 iframe
//   里展示。通过本端点反代后,响应头由我们控制,可以彻底抹掉这个限制。
//   CSP frame-ancestors 同理。
//
// 安全与局限:
//   - 仅代理 GET/HEAD,不代理 POST/PUT(避免误写三方源,虽然 skillhub/skills.sh
//     都是只读站点,但保险起见)
//   - 不修改 cookie / 认证头(用户未登录三方站 = 不登录,与匿名访问行为一致)
//   - 不缓存(每次都拉三方源,避免 stale)
//   - 不对返回的 HTML 做 deep rewrite(避免漏改 / 引入漏洞);相对路径资源
//     让浏览器自然走相对解析,代理自动捕获(因为请求打到同源,相对 URL 拼到
//     当前 URL 上 = /api/skillbox/market-iframe-proxy/<site>/...,正好被本 handler 再次代理)
//   - 不修改 HTML body(不注入 <base>),保持三方页面原始行为
//
// 路由:
//   GET /api/skillbox/market-iframe-proxy/:site/*path
//     :site = "skillhub" | "skillssh"
//     *path = 目标站点的 URL 路径(必须以 / 开头)
//
// 已知限制:
//   - 部分三方页面的 JS 会 fetch 绝对 URL(同源策略自动禁止),导致页面功能
//     不完整 — 这是浏览器安全策略,代理无法绕过。属于 iframe 内嵌通病,
//     skillhub 的列表浏览 / 搜索 / 详情跳转基本可用。

package cmarket

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"ginp-api/pkg/ginp"
)

// 允许代理的站点白名单(防 SSRF:用户不能拿这个端点当 open proxy 扫内网)。
//
// 每个站点一个固定的上游 origin,不允许 query/path 里动态指定。site 段是
// 路由参数,后端做严格匹配,前端硬编码两个值,任何其它值直接 400。
var marketIframeProxyUpstreams = map[string]string{
	"skillhub": "https://skillhub.cn",
	"skillssh": "https://www.skills.sh",
}

// iframeProxyTimeout 单次代理超时(覆盖 transport 默认 0 = 无超时)。
// skillhub 页面 + 资源一次完整加载 ~5s,留 20s 充裕。
const iframeProxyTimeout = 20 * time.Second

// marketIframeProxy 反向代理 GET /api/skillbox/market-iframe-proxy/:site/*path
//
// site: skillhub | skillssh
// 其余 path 段透传到上游同路径。
//
// 实现:
//   1. 校验 site 在白名单里,否则 400
//   2. 拿上游 origin + path 拼出目标 URL
//   3. 用自定义 transport 发请求(过滤 hop-by-hop 头,删除 X-Frame-Options/CSP)
//   4. 把上游响应原样回写,只删 X-Frame-Options / Content-Security-Policy
func marketIframeProxy(c *gin.Context) {
	site := c.Param("site")
	upstream, ok := marketIframeProxyUpstreams[site]
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "unknown site: " + site,
		})
		return
	}

	// *path 段:gin 给出来带 / 前缀,直接拼到 origin 后面即可。
	// 例:site=skillhub, *path=/skills?sortBy=curated_score → https://skillhub.cn/skills?sortBy=curated_score
	rawPath := c.Param("path")
	target := upstream + rawPath
	if parsed, err := url.Parse(target); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad target url"})
		return
	} else {
		// 把客户端 query 拼到 target 上(gin 的 *path 不会带 query,但有些客户端会)
		if c.Request.URL.RawQuery != "" {
			if parsed.RawQuery == "" {
				parsed.RawQuery = c.Request.URL.RawQuery
			} else {
				parsed.RawQuery = parsed.RawQuery + "&" + c.Request.URL.RawQuery
			}
		}
		target = parsed.String()
	}

	// 构造上游请求。方法透传(虽然 init 只注册了 GET,这里保险起见,POST/PUT 也会被
	// router 路由到同 handler — 但实际只有 GET 会到这里,因为 path 路由只挂 GET)。
	req, err := http.NewRequestWithContext(c.Request.Context(), c.Request.Method, target, nil)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "build request: " + err.Error()})
		return
	}

	// 透传少量必要的请求头(其它一律不带,避免暴露客户端 IP / 登录 cookie)。
	// Accept / Accept-Language 让上游根据浏览器偏好返回内容。
	copyHeader(req.Header, c.Request.Header, "Accept", "Accept-Language", "User-Agent")
	// Host 由 transport 自动覆盖;不要透传(否则上游会拒绝)

	client := &http.Client{
		Timeout: iframeProxyTimeout,
	}
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "upstream: " + err.Error()})
		return
	}
	defer resp.Body.Close()

	// 拷贝响应头 → 删 X-Frame-Options / Content-Security-Policy → 写回
	for k, vs := range resp.Header {
		if isStrippedHeader(k) {
			continue
		}
		for _, v := range vs {
			c.Header(k, v)
		}
	}

	// status code
	c.Status(resp.StatusCode)

	// body
	_, _ = io.Copy(c.Writer, resp.Body)
}

// isStrippedHeader 是否要被抹掉的头(命中 X-Frame-Options / CSP)。
func isStrippedHeader(name string) bool {
	n := strings.ToLower(name)
	if n == "x-frame-options" || n == "content-security-policy" {
		return true
	}
	// 也有 Content-Security-Policy-Report-Only,不过不常见,一起抹了免得审计报错
	if n == "content-security-policy-report-only" {
		return true
	}
	return false
}

// copyHeader 把 src 里的指定 header 拷到 dst(用第一个值)。
func copyHeader(dst, src http.Header, names ...string) {
	for _, n := range names {
		if v := src.Get(n); v != "" {
			dst.Set(n, v)
		}
	}
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:    "/api/skillbox/market-iframe-proxy/:site/*path",
		Handler: marketIframeProxy,
		// gin 的 *path 是 GET 路由参数,这里直接挂 GET(handler 内部按 method
		// 透传到上游,只 GET 实际会被命中,RouterAppend 也只挂 GET)。
		HttpType:       ginp.HttpGet,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.market.iframe_proxy",
		Swagger: &ginp.SwaggerInfo{
			Title:       "market.iframe_proxy",
			Description: "iframe 反代三方站点页面(skillhub / skills.sh),抹掉 X-Frame-Options / CSP,让前端 iframe 可加载",
		},
	})
}
