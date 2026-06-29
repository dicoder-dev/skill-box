package cskill

import (
	"errors"

	"github.com/gin-gonic/gin"
	"ginp-api/internal/gapi/service/skill/sskill"
	"ginp-api/pkg/ginp"
	"ginp-api/pkg/logger"
)

// RequestMoveSkill 移动 skill 到另一分组。
//
// 2026-06-29 增:为支持拖拽。src_group_path / dst_group_path 都允许空(=根下);
// name 是叶子名(走 NormalizeName)。移动整个分组请用 move_group。
type RequestMoveSkill struct {
	SrcGroupPath string `json:"src_group_path" form:"src_group_path"`
	DstGroupPath string `json:"dst_group_path" form:"dst_group_path"`
	Name         string `json:"name" form:"name"`
}

// MoveSkill POST /api/skillbox/skills/move
func MoveSkill(c *ginp.ContextPlus, req *RequestMoveSkill) {
	store, err := sskill.NewStore()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	svc := sskill.New(store)
	if merr := svc.MoveSkill(req.SrcGroupPath, req.Name, req.DstGroupPath); merr != nil {
		switch {
		case errors.Is(merr, sskill.ErrEmptyName),
			errors.Is(merr, sskill.ErrInvalidGroupPath):
			c.JSON(400, gin.H{"error": merr.Error()})
		case errors.Is(merr, sskill.ErrNotFound):
			c.JSON(404, gin.H{"error": merr.Error()})
		default:
			// "target already exists" 等冲突 → 409
			logger.Warn("skill move: %v", merr)
			c.JSON(409, gin.H{"error": merr.Error()})
		}
		return
	}
	c.JSON(200, gin.H{"ok": true})
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/skills/move",
		Handler:        ginp.BindParamsHandler(MoveSkill, &RequestMoveSkill{}),
		HttpType:       ginp.HttpPost,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.skills.move",
		Swagger: &ginp.SwaggerInfo{
			Title:         "skills.move",
			Description:   "移动 skill 到另一分组(叶子名不变)",
			RequestParams: RequestMoveSkill{},
		},
	})
}
