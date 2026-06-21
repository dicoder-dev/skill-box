package ctag

import (
	"errors"

	"github.com/gin-gonic/gin"
	"ginp-api/internal/gapi/service/skillaudit/sskillaudit"
	"ginp-api/pkg/ginp"
	"ginp-api/pkg/logger"
)

// RequestRollbackTag 回滚请求。
type RequestRollbackTag struct {
	TagID uint `json:"tag_id"`
}

// RespondRollbackTag 响应。
type RespondRollbackTag = sskillaudit.RollbackOutput

// RollbackTag POST /api/skillbox/skills/tags/rollback
//
// 把 skill 当前状态回滚到指定 tag 的内容。
// 内部:先打一个 _pre_rollback_<ts> 隐式 tag(覆盖当前状态)→ 把目标 tag 的 files 写回 skillstore。
func RollbackTag(c *ginp.ContextPlus, req *RequestRollbackTag) {
	svc := newService()
	out, err := svc.Rollback(&sskillaudit.RollbackInput{TagID: req.TagID})
	if err != nil {
		switch {
		case errors.Is(err, sskillaudit.ErrTagNotFound):
			c.JSON(404, gin.H{"error": err.Error()})
		case errors.Is(err, sskillaudit.ErrSkillNotFound):
			c.JSON(404, gin.H{"error": err.Error()})
		default:
			logger.Error("tag rollback: %v", err)
			c.JSON(500, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(200, out)
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/skills/tags/rollback",
		Handler:        ginp.BindParamsHandler(RollbackTag, &RequestRollbackTag{}),
		HttpType:       ginp.HttpPost,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.skills.tags.rollback",
		Swagger: &ginp.SwaggerInfo{
			Title:         "skills.tags.rollback",
			Description:   "回滚 skill 到指定 tag(自动打 _pre_rollback 隐式 tag)",
			RequestParams: RequestRollbackTag{},
		},
	})
}
