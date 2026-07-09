// 市场 iframe 代理 — 反向代理三方站点页面资源,响应里抹掉 X-Frame-Options 与
// Content-Security-Policy 头,并在 HTML 响应里注入 <base href> 修正相对资源路径
// (2026-07-09 增,2026-07-09 改:补 base 注入,见下)。
//
// 为什么需要这个端点:
//   skillhub.cn 返回 `X-Frame-Options: SAMEORIGIN`,浏览器会拒绝在跨源 iframe
//   里展示。通过本端点反代后,响应头由我们控制,可以彻底抹掉这个限制。
//   CSP frame-ancestors 同理。
//
// 为什么必须注入 <base href> (2026-07-09 实测发现):
//   skillhub 页面是 Vercel Next.js 部署,HTML 里的 <script src="/_next/static/...">
//   用绝对路径(以 / 开头),浏览器解析时拼到 iframe 当前 URL 上 = 走我代理的
//   路径 /api/skillbox/market-iframe-proxy/skillhub/_next/...。但 Vercel 部署 ID
//   跟客户端 cookie 绑定,代理拿不到当前用户的 dpl cookie,所以静态资源 404。
//   解决:在 HTML head 里注入 <base href="https://skillhub.cn/">,让所有相对 URL
//   解析都从 skillhub 的 origin 开始,资源直接走 skillhub,不再绕代理。
//
// 安全与局限:
//   - 仅代理 GET,不代理 POST/PUT(避免误写三方源)
//   - 不缓存(避免 stale)
//   - HTML 注入仅插入 <base href> 一行(目标站点固定),不重写其它内容,降低引入漏洞的概率
//   - 非 HTML 响应(text/css, application/javascript, image/*)原样透传
//
// 路由:
//   GET /api/skillbox/market-iframe-proxy/:site/*path
//     :site = "skillhub" | "skillssh"
//     *path = 目标站点的 URL 路径(必须以 / 开头)
//
// 已知限制:
//   - 部分三方页面的 JS 会 fetch 绝对 URL(同源策略自动禁止),导致页面功能
//     不完整 — 这是浏览器安全策略,代理无法绕过。
//   - skillhub / skills.sh 内嵌浏览基本可用,部分交互可能不可用。

package cmarket

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"ginp-api/pkg/ginp"
)

// 允许代理的站点白名单(防 SSRF)。
var marketIframeProxyUpstreams = map[string]string{
	"skillhub": "https://skillhub.cn",
	"skillssh": "https://www.skills.sh",
}

// iframeProxyTimeout 单次代理超时。skillhub 页面 + 资源一次完整加载 ~5s,留 20s 充裕。
const iframeProxyTimeout = 20 * time.Second

// htmlContentType 响应里 Content-Type 命中 text/html 才注入 <base>。
// 非 HTML 响应原样透传(图片、css、js 等不应被改写)。
const htmlContentTypePrefix = "text/html"

// marketIframeProxy 反向代理 GET /api/skillbox/market-iframe-proxy/:site/*path
func marketIframeProxy(c *gin.Context) {
	site := c.Param("site")
	upstream, ok := marketIframeProxyUpstreams[site]
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "unknown site: " + site,
		})
		return
	}

	// 拼目标 URL
	rawPath := c.Param("path")
	target := upstream + rawPath
	parsed, err := url.Parse(target)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad target url"})
		return
	}
	if c.Request.URL.RawQuery != "" {
		if parsed.RawQuery == "" {
			parsed.RawQuery = c.Request.URL.RawQuery
		} else {
			parsed.RawQuery = parsed.RawQuery + "&" + c.Request.URL.RawQuery
		}
	}
	target = parsed.String()

	// 构造上游请求,只透传少量必要的请求头
	req, err := http.NewRequestWithContext(c.Request.Context(), c.Request.Method, target, nil)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "build request: " + err.Error()})
		return
	}
	copyHeader(req.Header, c.Request.Header, "Accept", "Accept-Language", "User-Agent")

	client := &http.Client{
		Timeout: iframeProxyTimeout,
	}
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "upstream: " + err.Error()})
		return
	}
	defer resp.Body.Close()

	// 拷贝响应头 → 删 X-Frame-Options / CSP → 写回
	for k, vs := range resp.Header {
		if isStrippedHeader(k) {
			continue
		}
		for _, v := range vs {
			c.Header(k, v)
		}
	}

	// 决定是否需要注入 <base>:
	//   - Content-Type 包含 text/html(HTML 文档)
	//   - 状态码是 200
	//   - 上游 origin 非空(白名单里都是固定 origin,这里防御性再判一次)
	ct := resp.Header.Get("Content-Type")
	isHTML := resp.StatusCode == http.StatusOK && strings.HasPrefix(strings.ToLower(ct), htmlContentTypePrefix)

	if isHTML {
		// 读完整 body,注入 <base> 后写回
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "read body: " + err.Error()})
			return
		}
		body = injectBaseHref(body, upstream)
		// body 长度变了,显式重设 Content-Length,避免下游 chunked encoding 异常
		c.Header("Content-Length", itoa(len(body)))
		c.Status(resp.StatusCode)
		_, _ = c.Writer.Write(body)
		return
	}

	// 非 HTML:流式透传
	c.Status(resp.StatusCode)
	_, _ = io.Copy(c.Writer, resp.Body)
}

// injectBaseHref 在 HTML head 里插入 <base href="...">。
// 优先插在 <head> 后,其次 <html> 后,再其次 body 起始前;都找不到时拼在头部。
// 同一会话的相对 URL 解析基准变成 upstream,这样浏览器对 /_next/... 等相对
// 路径资源直接走 skillhub/skillssh 的 origin,不再绕本代理。
func injectBaseHref(body []byte, origin string) []byte {
	tag := []byte(`<base href="` + origin + `/">`)

	// 优先 </head> 之前
	if idx := bytes.Index(body, []byte("</head>")); idx >= 0 {
		out := make([]byte, 0, len(body)+len(tag))
		out = append(out, body[:idx]...)
		out = append(out, tag...)
		out = append(out, body[idx:]...)
		return out
	}
	// 退到 <head> 之后(已经有 <head>)
	if idx := bytes.Index(body, []byte("<head")); idx >= 0 {
		// 找 <head ...> 结束 '>' 后插入
		end := bytes.IndexByte(body[idx:], '>')
		if end >= 0 {
			insertAt := idx + end + 1
			out := make([]byte, 0, len(body)+len(tag))
			out = append(out, body[:insertAt]...)
			out = append(out, tag...)
			out = append(out, body[insertAt:]...)
			return out
		}
	}
	// 退到 <html> 之后
	if idx := bytes.Index(body, []byte("<html")); idx >= 0 {
		end := bytes.IndexByte(body[idx:], '>')
		if end >= 0 {
			insertAt := idx + end + 1
			out := make([]byte, 0, len(body)+len(tag))
			out = append(out, body[:insertAt]...)
			out = append(out, tag...)
			out = append(out, body[insertAt:]...)
			return out
		}
	}
	// 退到 <body> 之前
	if idx := bytes.Index(body, []byte("<body")); idx >= 0 {
		out := make([]byte, 0, len(body)+len(tag))
		out = append(out, body[:idx]...)
		out = append(out, tag...)
		out = append(out, body[idx:]...)
		return out
	}
	// 都没有就直接拼在头部
	out := make([]byte, 0, len(body)+len(tag))
	out = append(out, tag...)
	out = append(out, body...)
	return out
}

// isStrippedHeader 是否要被抹掉的头(命中 X-Frame-Options / CSP)。
func isStrippedHeader(name string) bool {
	n := strings.ToLower(name)
	if n == "x-frame-options" || n == "content-security-policy" {
		return true
	}
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

// itoa 是 strconv.Itoa 的小封装,避免本文件多 import 一个包。
// 字节长度最大 ~几十 MB,int 装得下。
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:    "/api/skillbox/market-iframe-proxy/:site/*path",
		Handler: marketIframeProxy,
		HttpType:       ginp.HttpGet,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.market.iframe_proxy",
		Swagger: &ginp.SwaggerInfo{
			Title:       "market.iframe_proxy",
			Description: "iframe 反代三方站点页面(skillhub / skills.sh),抹掉 X-Frame-Options / CSP,注入 <base href> 修正相对资源路径",
		},
	})
}
