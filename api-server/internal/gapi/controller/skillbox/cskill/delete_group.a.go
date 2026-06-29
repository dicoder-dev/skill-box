package cskill

import (
	"errors"

	"github.com/gin-gonic/gin"
	"ginp-api/internal/gapi/service/skill/sskill"
	"ginp-api/pkg/ginp"
	"ginp-api/pkg/logger"
)

// RequestDeleteGroup 删分组入参。
//
// 2026-06-29 增:为支持"右键删除分组"。返回 deleted_skill_paths — 在 cascade=true
// 时填该分组下所有 skill 叶子的相对路径,供前端二次确认"是否同步清理工具目录";
// cascade=false 时若分组非空,该字段也填,作为 4xx 响应的附带信息。
type RequestDeleteGroup struct {
	GroupPath string `json:"group_path" form:"group_path"`
	Cascade   bool   `json:"cascade" form:"cascade"`
}

// DeleteGroup POST /api/skillbox/skills/group/delete
func DeleteGroup(c *ginp.ContextPlus, req *RequestDeleteGroup) {
	store, err := sskill.NewStore()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	svc := sskill.New(store)
	deleted, derr := svc.DeleteGroup(req.GroupPath, req.Cascade)
	if derr != nil {
		// 非空 + cascade=false → 400(让前端弹确认);其它 500
		if errors.Is(derr, sskill.ErrInvalidGroupPath) {
			c.JSON(400, gin.H{"error": derr.Error()})
			return
		}
		// 区分"非空拒绝" vs "物理删失败":检查 err msg 是否含 "not empty"
		// 简化:一并返 409 + 附带 deleted 列表,前端据此弹"包含 N 个 skill,确认级联删除吗"
		if !req.Cascade && len(deleted) > 0 {
			c.JSON(409, gin.H{
				"error":              derr.Error(),
				"deleted_skill_paths": deleted,
				"need_cascade":       true,
			})
			return
		}
		logger.Error("skill delete group: %v", derr)
		c.JSON(500, gin.H{"error": derr.Error(), "deleted_skill_paths": deleted})
		return
	}
	c.JSON(200, gin.H{
		"ok":                 true,
		"deleted_skill_paths": deleted,
	})
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/skills/group/delete",
		Handler:        ginp.BindParamsHandler(DeleteGroup, &RequestDeleteGroup{}),
		HttpType:       ginp.HttpPost,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.skills.group.delete",
		Swagger: &ginp.SwaggerInfo{
			Title:         "skills.group.delete",
			Description:   "删分组;cascade=true 时递归删子树,返回被删 skill 路径列表",
			RequestParams: RequestDeleteGroup{},
		},
	})
}
