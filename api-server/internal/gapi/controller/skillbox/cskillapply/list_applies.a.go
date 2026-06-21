package cskillapply

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"ginp-api/internal/gapi/service/skillapp/sskillapp"
	"ginp-api/pkg/ginp"
	"ginp-api/pkg/logger"
)

// RequestListApplies 列表请求。query + body 兼容。
type RequestListApplies struct {
	SkillID uint   `json:"skill_id" form:"skill_id"`
	Tool    string `json:"tool" form:"tool"`
	Status  string `json:"status" form:"status"`
	Page    int    `json:"page" form:"page"`
	Size    int    `json:"size" form:"size"`
}

// RespondListApplies 响应。
type RespondListApplies = sskillapp.ListResult

// ListApplies GET /api/skillbox/skills/apply/list
//
// 列出 apply 历史;支持 skill_id / tool / status 过滤 + 分页(按 applied_at desc)。
func ListApplies(c *ginp.ContextPlus, req *RequestListApplies) {
	// 兼容 query string
	if req.SkillID == 0 {
		if s := c.Query("skill_id"); s != "" {
			if n, err := strconv.ParseUint(s, 10, 64); err == nil {
				req.SkillID = uint(n)
			}
		}
	}
	svc := newService()
	out, err := svc.List(sskillapp.ListInput{
		SkillID: req.SkillID,
		Tool:    req.Tool,
		Status:  req.Status,
		Page:    req.Page,
		Size:    req.Size,
	})
	if err != nil {
		logger.Error("skill apply list: %v", err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, out)
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/skills/apply/list",
		Handler:        ginp.BindParamsHandler(ListApplies, &RequestListApplies{}),
		HttpType:       ginp.HttpGet,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.skills.apply.list",
		Swagger: &ginp.SwaggerInfo{
			Title:         "skills.apply.list",
			Description:   "列 apply 历史(按 applied_at desc)",
			RequestParams: RequestListApplies{},
		},
	})
}
