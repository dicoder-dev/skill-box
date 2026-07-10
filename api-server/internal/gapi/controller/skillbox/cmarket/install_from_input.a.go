package cmarket

import (
	"context"
	"errors"
	"time"

	"github.com/gin-gonic/gin"
	"ginp-api/internal/gapi/service/market/smarket"
	"ginp-api/internal/skillmarket"
	"ginp-api/pkg/ginp"
	"ginp-api/pkg/logger"
)

// RequestInstallFromInput 用户输入框下载入参(2026-07-09 增)。
//
// 与 RequestPullMarketSkillV2 的差别:不要求 caller 提前知道 source_id / remote_id;
// 把"用户原文"(详情页 URL)交给 service 解析。
//
// SourceHint:
//   - 空字符串 = auto(由后端解析 input 推断;URL 输入由域名决定)
//   - "skillhub" / "skillssh" / "github" = 强制指定 source(只接受该 source 的 URL)
//
// Scope:
//   - 空 / "global" = 全局,默认
//   - "project" = 项目级,需填 ProjectID
//
// ConflictMode(2026-07-09 增):
//   - 空 / "prompt" = 同名已存在 → 返 409(响应体含 conflict_existing_* 字段,前端弹确认)
//   - "overwrite" = 覆盖同名 skill
//   - "rename" = 自动加 -2/-3 后缀(或用 RenameTo 字段)
type RequestInstallFromInput struct {
	SourceHint   string `json:"source_hint"`
	Input        string `json:"input"`
	Scope        string `json:"scope"`
	ProjectID    uint   `json:"project_id"`
	ConflictMode string `json:"conflict_mode"`
	RenameTo     string `json:"rename_to"`
}

// RespondInstallFromInput 落盘结果(2026-07-09 增)。
type RespondInstallFromInput = smarket.InstallFromInputResult

// InstallFromInput POST /api/skillbox/market/install-from-input
//
// 用户在 MarketView 输入框粘贴 skill slug / 详情页 URL → 后端解析 → 下载 → 落 store。
//
// 错误处理:
//   - ErrInvalidInput → 400(用户输入无法识别,前端可保留输入让用户重试)
//   - ErrSourceNotFound → 404(源未注册,跟 ErrRemoteNotFound 区分)
//   - ErrRemoteNotFound → 404(slug 不存在,2026-07-10 增:前端走 errSkillNotFound 文案),
//     跟 ErrSourceNotFound 区分(用错源 vs 源 OK 但 slug 不存在)
//   - 其它 → 500(下载/写盘失败)
//
// 2026-07-09 增:首次进入市场时前端不一定先调过 ListSources,
// 后端 Service 层兜底自 seed 默认源(skillhub / skills.sh),保证首次 install 也能跑通。
func InstallFromInput(c *ginp.ContextPlus, req *RequestInstallFromInput) {
	if req == nil || req.Input == "" {
		c.JSON(400, gin.H{"error": "input 必填"})
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	out, err := newServiceV2().InstallFromInput(ctx, &smarket.InstallFromInputInput{
		SourceHint:   req.SourceHint,
		Input:        req.Input,
		Scope:        req.Scope,
		ProjectID:    req.ProjectID,
		ConflictMode: req.ConflictMode,
		RenameTo:     req.RenameTo,
	})
	if err != nil {
		switch {
		case errors.Is(err, smarket.ErrInvalidInput):
			c.JSON(400, gin.H{"error": err.Error()})
		case errors.Is(err, smarket.ErrSourceNotFound):
			c.JSON(404, gin.H{"error": err.Error()})
		case errors.Is(err, skillmarket.ErrRemoteNotFound):
			// 2026-07-10 增:slug 不存在(典型如用户粘贴错 URL / slug 已下架),
			// 返 404 + 错误信息精确。前端根据 404 + err 信息命中「errSkillNotFound」文案。
			c.JSON(404, gin.H{"error": err.Error()})
		case errors.Is(err, skillmarket.ErrSkillMalformed):
			// 2026-07-10 增:skill 在上游存在(zip 能拉到),但 SKILL.md 文件格式
			// 有问题(典型:缺 frontmatter / 内容空 / zip 不合法)。
			// 跟 ErrRemoteNotFound 区分 ——
			// - 404 「找不到」 → 换别的 skill 试试
			// - 422 「文件坏了」 → 该 skill 作者发布时漏了 metadata,找作者反馈
			// 前端在 422 状态码 + err 信息含 "malformed" 时走 errSkillMalformed 文案。
			c.JSON(422, gin.H{"error": err.Error()})
		case errors.Is(err, smarket.ErrSkillAlreadyExists):
			// 2026-07-09 增:同名冲突,返 409 + 现有 skill 信息(让前端弹覆盖确认)。
			// 即使 err != nil 也要把 out 返回(里面含 conflict_existing_* 字段),
			// gin.H 兜底空 result。
			result := gin.H{"error": err.Error()}
			if out != nil {
				result["conflict_existing_version"] = out.ConflictExistingVersion
				result["conflict_existing_path"] = out.ConflictExistingPath
				result["skill_name"] = out.SkillName
				result["source_type"] = out.SourceType
			}
			c.JSON(409, result)
		default:
			logger.Error("market install from input: %v", err)
			c.JSON(500, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(200, out)
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/market/install-from-input",
		Handler:        ginp.BindParamsHandler(InstallFromInput, &RequestInstallFromInput{}),
		HttpType:       ginp.HttpPost,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.market.install_from_input",
		Swagger: &ginp.SwaggerInfo{
			Title:         "market.install_from_input",
			Description:   "市场输入框一键拉取:用户原文(slug / 详情页 URL / GitHub URL)→ 解析 source+remoteID → 下载 → 落 store",
			RequestParams: RequestInstallFromInput{},
		},
	})
}