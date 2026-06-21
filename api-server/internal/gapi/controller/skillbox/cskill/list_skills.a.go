package cskill

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"ginp-api/internal/db/dbs"
	"ginp-api/internal/gapi/service/skill/sskill"
	"ginp-api/pkg/ginp"
	"ginp-api/pkg/logger"
)

// RequestListSkills 列表请求。query + body 兼容。
type RequestListSkills struct {
	Scope     string `json:"scope" form:"scope"`
	ProjectID uint   `json:"project_id" form:"project_id"`
	Keyword   string `json:"keyword" form:"keyword"`
	Page      int    `json:"page" form:"page"`
	Size      int    `json:"size" form:"size"`
}

// RespondListSkills 列表响应。
type RespondListSkills = sskill.ListResult

// ListSkills GET /api/skillbox/skills
func ListSkills(c *ginp.ContextPlus, req *RequestListSkills) {
	store, err := sskill.NewStore()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	svc := sskill.New(dbs.GetWriteDb(), dbs.GetReadDb(), store)
	out, err := svc.List(sskill.ListQuery{
		Scope:     req.Scope,
		ProjectID: req.ProjectID,
		Keyword:   req.Keyword,
		Page:      req.Page,
		Size:      req.Size,
	})
	if err != nil {
		logger.Error("skill list: %v", err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, out)
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/skills",
		Handler:        ginp.BindParamsHandler(ListSkills, &RequestListSkills{}),
		HttpType:       ginp.HttpGet,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.skills.list",
		Swagger: &ginp.SwaggerInfo{
			Title:         "skills.list",
			Description:   "列出 skill,支持 scope / project_id / keyword 过滤 + 分页",
			RequestParams: RequestListSkills{},
		},
	})
}

// itoa 暂留(后续分页可能用)
var _ = strconv.Itoa
