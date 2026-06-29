package cskill

import (
	"errors"

	"github.com/gin-gonic/gin"
	"ginp-api/internal/gapi/service/skill/sskill"
	"ginp-api/pkg/ginp"
	"ginp-api/pkg/logger"
)

// RequestMoveGroup 移动整个分组到另一分组(子路径)下。
//
// 2026-06-29 增:前端拖动分组时调用。src_group_path 不能为空(根不可移),
// dst_group_path 可为空(挪到根下)。分组名(最后一段)在挪动后保持不变。
type RequestMoveGroup struct {
	SrcGroupPath string `json:"src_group_path" form:"src_group_path"`
	DstGroupPath string `json:"dst_group_path" form:"dst_group_path"`
}

// MoveGroup POST /api/skillbox/skills/group/move
func MoveGroup(c *ginp.ContextPlus, req *RequestMoveGroup) {
	store, err := sskill.NewStore()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	svc := sskill.New(store)
	if merr := svc.MoveGroup(req.SrcGroupPath, req.DstGroupPath); merr != nil {
		switch {
		case errors.Is(merr, sskill.ErrInvalidGroupPath):
			c.JSON(400, gin.H{"error": merr.Error()})
		case errors.Is(merr, sskill.ErrNotFound):
			c.JSON(404, gin.H{"error": merr.Error()})
		default:
			// "already exists" / "empty src" 等冲突 → 409
			logger.Warn("skill move group: %v", merr)
			c.JSON(409, gin.H{"error": merr.Error()})
		}
		return
	}
	c.JSON(200, gin.H{"ok": true})
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/skills/group/move",
		Handler:        ginp.BindParamsHandler(MoveGroup, &RequestMoveGroup{}),
		HttpType:       ginp.HttpPost,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.skills.group.move",
		Swagger: &ginp.SwaggerInfo{
			Title:         "skills.group.move",
			Description:   "把整个分组从 src_group_path 移到 dst_groupPath 下(name 不变,取 src 的最后一段);dst 可空 = 挪到根下;同层同名冲突 → 409",
			RequestParams: RequestMoveGroup{},
		},
	})
}
