package cmarket

import (
	"context"
	"errors"
	"strings"
	"time"

	"ginp-api/internal/gapi/service/market/smarket"
	"ginp-api/pkg/ginp"
	"ginp-api/pkg/logger"

	"github.com/gin-gonic/gin"
)

// RequestListMarketSkillsRemote 列表请求(2026-07-01 增)。
//
// 与 RequestListMarketSkillsWithInstalled 字段一致,但内部走 adapter.Discover,
// 不读 market_skills 表 — 数据永远是三方源最新。
type RequestListMarketSkillsRemote struct {
	SourceID uint   `json:"source_id" form:"source_id"`
	Keyword  string `json:"keyword" form:"keyword"`
	Page     int    `json:"page" form:"page"`
	Size     int    `json:"size" form:"size"`
}

// RespondListMarketSkillsRemote 响应。结构与 ListSkillsWithInstalledResult 一致,
// 让前端替换调用即可,无需改 schema。
type RespondListMarketSkillsRemote = smarket.ListSkillsWithInstalledResult

// ErrMarketRemoteUnreachable 三方源不可达(2026-07-03 增)。
//
// 2026-07-03 背景:之前 err.Error() 直接吐给前端,前端只能展示成 "Timeout" 或
// "code=500",无法区分"三方源挂了"和"自己后端挂了"。
// 现在 controller 把所有 ctx 超时 / 网络层错误统一包装成这个 sentinel + 502,
// 前端 store 检测 status===502 + 错误前缀 = 走 banner + fallback 列表路径。
//
// 注意:smarket / adapter / registry 几层错误都透传到 controller 统一包装,
// 这样 adapter 内部不需要感知 HTTP 层语义。
var ErrMarketRemoteUnreachable = errors.New("market_remote_unreachable")

// wrapMarketRemoteErr 把各种"看起来是三方源问题"的 err 包成 sentinel + 502。
//
// 触发条件:
//   - ctx.DeadlineExceeded 或 ctx.Canceled(超时/取消)
//   - 字符串里含 net.OpError / connection refused / timeout / i/o timeout / EOF
//     / no such host 等典型三方源网络错误
//
// 不命中:其它业务错误(如 source_id 找不到)原样返回,由前端按业务码处理。
func wrapMarketRemoteErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return errors.Join(ErrMarketRemoteUnreachable, err)
	}
	low := strings.ToLower(err.Error())
	for _, needle := range []string{
		"timeout", "deadline exceeded", "context deadline",
		"connection refused", "no such host", "i/o timeout",
		"network is unreachable", "tls handshake", "eof",
		"connection reset", "dial tcp",
	} {
		if strings.Contains(low, needle) {
			return errors.Join(ErrMarketRemoteUnreachable, err)
		}
	}
	return err
}

// ListMarketSkillsRemote GET /api/skillbox/market/skills-remote
//
// 2026-07-01 增:走 adapter.Discover,每次都打三方源,完全不读本地缓存。
// skillhub:走 /api/skills?keyword=&pageSize=100;
// skills.sh:走 /api/audits/0..49 + substring(API 无搜索参数,只能 substring 过滤);
// installed 二次扫本地 store,不影响主列表。
//
// 2026-07-01 改:45s → 90s。
// 原因:skillhub 去掉 maxDiscoverItems 硬上限后,翻页跑到 total 全部才能停。
// 实测 1000 条 = 2.3s,推算 40000 条 = 90s;留 90s 撑住 skillhub 全网当前量级。
// 90s 内若 ctx 取消,skillhub.Discover 会 break 并保留已拿到的 items(不全 fallback)。
//
// 2026-07-03 改:
//   - err 通过 wrapMarketRemoteErr 识别"三方源不可达",统一返 502 + sentinel 错误。
//   - skillhub adapter 内部已加 8s hard deadline,UI 不再假死;此 ctx=90s 主要给 skillssh
//     海外稳定路径留余量。超时仍按 502 处理。
func ListMarketSkillsRemote(c *ginp.ContextPlus, req *RequestListMarketSkillsRemote) {
	if req.SourceID == 0 {
		c.JSON(400, gin.H{"error": "source_id 必填"})
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	svc := newService()
	out, err := svc.ListSkillsRemote(ctx, smarket.ListSkillsQuery{
		SourceID: req.SourceID,
		Keyword:  req.Keyword,
		Page:     req.Page,
		Size:     req.Size,
	})
	if err != nil {
		wrapped := wrapMarketRemoteErr(err)
		if errors.Is(wrapped, ErrMarketRemoteUnreachable) {
			// 2026-07-03 增:三方源超时/网络层失败 → 502 + sentinel 前缀,
			// 前端 store 用 status===502 + 错误前缀识别"远端不可达",走 banner 兜底。
			logger.Warn("market list remote (remote unreachable): %v", err)
			c.JSON(502, gin.H{
				"error":  "market_remote_unreachable: " + err.Error(),
				"reason": "remote_unreachable",
			})
			return
		}
		logger.Error("market list remote: %v", err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, out)
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/market/skills-remote",
		Handler:        ginp.BindParamsHandler(ListMarketSkillsRemote, &RequestListMarketSkillsRemote{}),
		HttpType:       ginp.HttpGet,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.market.skills.list_remote",
		Swagger: &ginp.SwaggerInfo{
			Title:         "market.skills.list_remote",
			Description:   "列三方市场 skill(纯远端,不读本地缓存)",
			RequestParams: RequestListMarketSkillsRemote{},
		},
	})
}
