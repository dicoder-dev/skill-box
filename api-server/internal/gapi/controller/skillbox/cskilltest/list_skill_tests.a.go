package cskilltest

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"ginp-api/internal/db/dbs"
	"ginp-api/internal/gapi/service/skill/sskill"
	"ginp-api/internal/gapi/service/skilltester/sskilltest"
	"ginp-api/pkg/ginp"
	"ginp-api/pkg/logger"
)

// RequestListSkillTests 列表入参(支持 query / body 兼容)。2026-06-24 改造:用 (scope, name) 定位。
type RequestListSkillTests struct {
	Scope string `json:"scope" form:"scope"`
	Name  string `json:"name" form:"name"`
	Page  int    `json:"page" form:"page"`
	Size  int    `json:"size" form:"size"`
}

// ListSkillTests GET /api/skillbox/skills/test/list
func ListSkillTests(c *ginp.ContextPlus, req *RequestListSkillTests) {
	if req.Scope == "" {
		req.Scope = c.Query("scope")
	}
	if req.Name == "" {
		req.Name = c.Query("name")
	}
	store, err := sskill.NewStore()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	svc := sskilltest.New(dbs.GetWriteDb(), dbs.GetReadDb(), store, nil, nil)
	out, err := svc.List(&sskilltest.ListRequest{
		Scope: req.Scope,
		Name:  req.Name,
		Page:  req.Page,
		Size:  req.Size,
	})
	if err != nil {
		logger.Error("skill test list: %v", err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, out)
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/skills/test/list",
		Handler:        ginp.BindParamsHandler(ListSkillTests, &RequestListSkillTests{}),
		HttpType:       ginp.HttpGet,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.skills.test.list",
		Swagger: &ginp.SwaggerInfo{
			Title:         "skills.test.list",
			Description:   "列出 skill 测试 run,按 scope+name 过滤 + 分页",
			RequestParams: RequestListSkillTests{},
		},
	})
}

// 保留 strconv 引用(后续可能用)
var _ = strconv.Itoa
