package cskill

import (
	"errors"

	"github.com/gin-gonic/gin"
	"ginp-api/internal/gapi/service/skill/sskill"
	"ginp-api/pkg/ginp"
	"ginp-api/pkg/logger"
)

// RequestDeleteSkill 删除入参。
//
// 2026-06-29 改:支持多级分组 — 用 path(完整相对路径,如 "frontend/react")
// 替代旧版只用 name。name 字段仍兼容旧调用(空时由 path 推导);cascade_tools
// 是前端在确认弹窗里勾选"同步清理工具目录"后传 true,后端只删 skillbox 库内
// 的副本(走 DeleteByPath),工具目录的清理由前端循环调 forceUndoApply 实现
// (后端保持单一职责)。
type RequestDeleteSkill struct {
	Name string `json:"name" form:"name"`
	// Path 完整相对路径(可空,空时旧逻辑:从 Name 推导 — 但 Name 含分组的话
	// 必须用 Path 传)
	Path string `json:"path" form:"path"`
}

// DeleteSkill POST /api/skillbox/skills/delete
func DeleteSkill(c *ginp.ContextPlus, req *RequestDeleteSkill) {
	store, err := sskill.NewStore()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	svc := sskill.New(store)
	// 解析 path
	groupPath, name := sskill.SplitPath(req.Path)
	if name == "" {
		// 旧调用兜底:用 Name(没分组时 name = req.Name)
		name = req.Name
	}
	if name == "" {
		c.JSON(400, gin.H{"error": "name or path is required"})
		return
	}
	if _, derr := svc.DeleteByPath(groupPath, name); derr != nil {
		switch {
		case errors.Is(derr, sskill.ErrEmptyName),
			errors.Is(derr, sskill.ErrInvalidGroupPath):
			c.JSON(400, gin.H{"error": derr.Error()})
		default:
			logger.Error("skill delete: %v", derr)
			c.JSON(500, gin.H{"error": derr.Error()})
		}
		return
	}
	c.JSON(200, gin.H{"ok": true})
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/skills/delete",
		Handler:        ginp.BindParamsHandler(DeleteSkill, &RequestDeleteSkill{}),
		HttpType:       ginp.HttpPost,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.skills.delete",
		Swagger: &ginp.SwaggerInfo{
			Title:         "skills.delete",
			Description:   "按 path/name 删 skill(整个目录);幂等",
			RequestParams: RequestDeleteSkill{},
		},
	})
}
