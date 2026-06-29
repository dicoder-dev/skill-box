package cskill

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
	"ginp-api/internal/gapi/service/skill/sskill"
	"ginp-api/pkg/ginp"
	"ginp-api/pkg/logger"
)

// RequestRenameGroup 重命名分组入参。
// src_group_path 允许多级(用 '/' 分隔,如 "frontend/react");new_name 是单段名
// (不含 '/',由 service 走 NormalizeName 规约)。
type RequestRenameGroup struct {
	SrcGroupPath string `json:"src_group_path" form:"src_group_path"`
	NewName      string `json:"new_name" form:"new_name"`
}

// RenameGroup POST /api/skillbox/skills/group/rename
//
// 2026-06-29 增:为支持"分组右键重命名"。只改最后一段(父路径不变),
// 同层同名冲突 → 409;非法名字 → 400;源不存在 → 404。
func RenameGroup(c *ginp.ContextPlus, req *RequestRenameGroup) {
	store, err := sskill.NewStore()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	svc := sskill.New(store)
	newPath, rerr := svc.RenameGroup(req.SrcGroupPath, req.NewName)
	if rerr != nil {
		// 非法(空名 / .. 段 / 含 '/')
		if errors.Is(rerr, sskill.ErrInvalidGroupPath) {
			c.JSON(400, gin.H{"error": rerr.Error()})
			return
		}
		msg := rerr.Error()
		// 源不存在(ErrNotFound)→ 404
		if strings.Contains(msg, "not found") || strings.Contains(msg, "no such file") {
			c.JSON(404, gin.H{"error": msg})
			return
		}
		// 同层同名冲突 → 409
		if strings.Contains(msg, "already exists") {
			c.JSON(409, gin.H{"error": msg, "code": "target_exists"})
			return
		}
		logger.Error("skill rename group: %v", rerr)
		c.JSON(500, gin.H{"error": msg})
		return
	}
	c.JSON(200, gin.H{"ok": true, "new_group_path": newPath})
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/skills/group/rename",
		Handler:        ginp.BindParamsHandler(RenameGroup, &RequestRenameGroup{}),
		HttpType:       ginp.HttpPost,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.skills.group.rename",
		Swagger: &ginp.SwaggerInfo{
			Title:         "skills.group.rename",
			Description:   "重命名分组的最后一段(父路径不变);已存在同名 → 409,非法名 → 400,源不存在 → 404",
			RequestParams: RequestRenameGroup{},
		},
	})
}
