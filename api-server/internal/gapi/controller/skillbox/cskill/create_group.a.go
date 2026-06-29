package cskill

import (
	"errors"

	"github.com/gin-gonic/gin"
	"ginp-api/internal/gapi/service/skill/sskill"
	"ginp-api/internal/skilladapter"
	"ginp-api/pkg/ginp"
	"ginp-api/pkg/logger"
)

// RequestCreateGroup 新建分组入参。
// group_path 允许多级(用 '/' 分隔,如 "frontend/react");空字符串 = 根下,等价于 noop。
type RequestCreateGroup struct {
	GroupPath string `json:"group_path" form:"group_path"`
}

// CreateGroup POST /api/skillbox/skills/group/create
func CreateGroup(c *ginp.ContextPlus, req *RequestCreateGroup) {
	store, err := sskill.NewStore()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	svc := sskill.New(store)
	if gerr := svc.CreateGroup(req.GroupPath); gerr != nil {
		if errors.Is(gerr, sskill.ErrInvalidGroupPath) {
			c.JSON(400, gin.H{"error": gerr.Error()})
			return
		}
		logger.Error("skill create group: %v", gerr)
		c.JSON(500, gin.H{"error": gerr.Error()})
		return
	}
	// 规范化后的 group_path 回传,前端可以直接拿去做选中态
	norm := skilladapter.NormalizeGroupName(req.GroupPath)
	c.JSON(200, gin.H{"ok": true, "group_path": norm})
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/skills/group/create",
		Handler:        ginp.BindParamsHandler(CreateGroup, &RequestCreateGroup{}),
		HttpType:       ginp.HttpPost,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.skills.group.create",
		Swagger: &ginp.SwaggerInfo{
			Title:         "skills.group.create",
			Description:   "新建分组目录(可多级);已存在不报错",
			RequestParams: RequestCreateGroup{},
		},
	})
}
