package cskillapply

import (
	"github.com/gin-gonic/gin"
	"ginp-api/internal/gapi/service/skillapp/sskillapp"
	"ginp-api/pkg/ginp"
	"ginp-api/pkg/logger"
)

// RequestMigrateMode 切换 apply 模式时由前端调用。
//
// Mode 必须为 "copy" 或 "symlink";与 settings 旧值相同时,后端做幂等返空
// (Total=0),不报错,便于前端在用户二次点"保存"时不显示错误。
type RequestMigrateMode struct {
	Mode string `json:"mode"`
}

// RespondMigrateMode 单次切换的明细 + 汇总。
type RespondMigrateMode = sskillapp.MigrateModeResult

// MigrateApplyMode POST /api/skillbox/skills/apply/migrate-mode
//
// 把所有 status=applied 的 skill_applies 行从 settings.apply_mode 切到 req.Mode。
// 设计意图:用户切模式时,settings.Service.SetApplyMode 会改 future apply 的
// 行为,但"已 apply 的 target_dir 是 copy 还是 symlink"不会跟着变 —— 走本接口
// 才能批量改。
//
// 失败策略:逐行迁移,某行失败不影响其他行;汇总在 Entries 里,前端可展示给用户
// 哪些行失败(常见原因:源 skill 已从 store 删 / tool adapter 不支持 symlink)。
func MigrateApplyMode(c *ginp.ContextPlus, req *RequestMigrateMode) {
	if req.Mode == "" {
		c.JSON(400, gin.H{"error": "mode required (copy/symlink)"})
		return
	}
	svc := newService()
	out, err := svc.MigrateMode(req.Mode)
	if err != nil {
		logger.Error("skill apply migrate-mode: %v", err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, out)
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/skills/apply/migrate-mode",
		Handler:        ginp.BindParamsHandler(MigrateApplyMode, &RequestMigrateMode{}),
		HttpType:       ginp.HttpPost,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.skills.apply.migrate_mode",
		Swagger: &ginp.SwaggerInfo{
			Title:         "skills.apply.migrate_mode",
			Description:   "批量切换已 apply 的 skill 在磁盘上的存在形式(copy ↔ symlink)",
			RequestParams: RequestMigrateMode{},
		},
	})
}
