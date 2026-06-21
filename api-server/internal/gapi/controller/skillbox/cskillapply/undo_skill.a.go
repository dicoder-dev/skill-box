package cskillapply

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	"ginp-api/internal/gapi/service/skillapp/sskillapp"
	"ginp-api/internal/skillapp"
	"ginp-api/pkg/ginp"
	"ginp-api/pkg/logger"
)

// RequestUndoSkill 撤销请求。
type RequestUndoSkill struct {
	ApplyID uint `json:"apply_id" form:"apply_id"`
}

// RespondUndoSkill 响应。
type RespondUndoSkill = sskillapp.UndoResult

// UndoSkill POST /api/skillbox/skills/apply/undo
//
// 撤销一条 apply(按 apply_id);同时更新 SkillApply.rolled_back_at。
func UndoSkill(c *ginp.ContextPlus, req *RequestUndoSkill) {
	svc := newService()
	if req.ApplyID == 0 {
		// query string fallback
		if s := c.Query("apply_id"); s != "" {
			if n, err := strconv.ParseUint(s, 10, 64); err == nil {
				req.ApplyID = uint(n)
			}
		}
	}
	out, err := svc.Undo(req.ApplyID)
	if err != nil {
		switch {
		case errors.Is(err, skillapp.ErrApplyNotFound):
			c.JSON(404, gin.H{"error": err.Error()})
		case errors.Is(err, skillapp.ErrAlreadyRolled):
			c.JSON(409, gin.H{"error": err.Error()})
		default:
			logger.Error("skill undo: %v", err)
			c.JSON(500, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(200, out)
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/skills/apply/undo",
		Handler:        ginp.BindParamsHandler(UndoSkill, &RequestUndoSkill{}),
		HttpType:       ginp.HttpPost,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.skills.apply.undo",
		Swagger: &ginp.SwaggerInfo{
			Title:         "skills.apply.undo",
			Description:   "撤销一条 apply(根据 apply_id)",
			RequestParams: RequestUndoSkill{},
		},
	})
}
