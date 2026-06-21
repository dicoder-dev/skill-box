package cskillapply

import (
	"github.com/gin-gonic/gin"
	"ginp-api/internal/gapi/service/skillapp/sskillapp"
	"ginp-api/internal/skillapp"
	"ginp-api/pkg/ginp"
	"ginp-api/pkg/logger"
)

// RequestApplyBatch 多 skill 批量 apply。
type RequestApplyBatch struct {
	Items  []sskillapp.ApplyInput `json:"items"`
	Atomic bool                   `json:"atomic"`
}

// RespondApplyBatch 响应。
type RespondApplyBatch = skillapp.BatchOutput

// ApplyBatch POST /api/skillbox/skills/apply/batch
//
// 跑 (skill × tool) 笛卡尔积;atomic=true 时任一失败 → 整体回滚已成功的。
func ApplyBatch(c *ginp.ContextPlus, req *RequestApplyBatch) {
	svc := newService()
	out, err := svc.BatchApply(&sskillapp.BatchApplyInput{
		Items:  req.Items,
		Atomic: req.Atomic,
	})
	if err != nil {
		logger.Error("skill apply batch: %v", err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, out)
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/skills/apply/batch",
		Handler:        ginp.BindParamsHandler(ApplyBatch, &RequestApplyBatch{}),
		HttpType:       ginp.HttpPost,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.skills.apply.batch",
		Swagger: &ginp.SwaggerInfo{
			Title:         "skills.apply.batch",
			Description:   "批量 apply(多 skill × 多 tool)",
			RequestParams: RequestApplyBatch{},
		},
	})
}
