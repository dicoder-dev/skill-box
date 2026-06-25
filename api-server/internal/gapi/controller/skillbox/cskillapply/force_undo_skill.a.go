package cskillapply

import (
	"strings"

	"github.com/gin-gonic/gin"
	"ginp-api/internal/gapi/service/skillapp/sskillapp"
	"ginp-api/internal/skillapp"
	"ginp-api/pkg/ginp"
	"ginp-api/pkg/logger"
)

// RequestForceUndoSkill 强制撤销请求(不依赖 apply_id)。
//
// 2026-06-25 增:用于"scope-status 命中但 DB 无 apply 记录"场景
// (用户手动 cp / 外部安装)。service 内部优先按 DB 记录走标准 Undo,
// 找不到才用 scope-status 定位磁盘 + 直接删。
type RequestForceUndoSkill struct {
	Scope     string `json:"scope" form:"scope"`
	ProjectID uint   `json:"project_id" form:"project_id"`
	Name      string `json:"name" form:"name"`
	Tool      string `json:"tool" form:"tool"`
}

// ForceUndoSkill POST /api/skillbox/skills/apply/force-undo
func ForceUndoSkill(c *ginp.ContextPlus, req *RequestForceUndoSkill) {
	if strings.TrimSpace(req.Name) == "" {
		c.JSON(400, gin.H{"error": "name is required"})
		return
	}
	if strings.TrimSpace(req.Tool) == "" {
		c.JSON(400, gin.H{"error": "tool is required"})
		return
	}
	svc := newService()
	out, err := svc.ForceUndo(&sskillapp.ForceUndoInput{
		Scope:     req.Scope,
		ProjectID: req.ProjectID,
		Name:      req.Name,
		Tool:      req.Tool,
	})
	if err != nil {
		switch err {
		case skillapp.ErrApplyNotFound:
			c.JSON(404, gin.H{"error": err.Error()})
		default:
			logger.Error("skill force undo: %v", err)
			c.JSON(500, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(200, out)
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/skills/apply/force-undo",
		Handler:        ginp.BindParamsHandler(ForceUndoSkill, &RequestForceUndoSkill{}),
		HttpType:       ginp.HttpPost,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.skills.apply.force-undo",
		Swagger: &ginp.SwaggerInfo{
			Title:         "skills.apply.force-undo",
			Description:   "按 (scope, project_id, name, tool) 强制撤销;DB 有记录走标准 Undo,无记录走 scope-status 删磁盘",
			RequestParams: RequestForceUndoSkill{},
		},
	})
}
