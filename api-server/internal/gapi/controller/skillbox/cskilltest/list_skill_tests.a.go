// Package cskilltest - list_skill_tests.a.go
// GET /api/skillbox/skills/test/list?skill_id=&page=&size=
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

// RequestListSkillTests 列表入参(支持 query / body 兼容)。
type RequestListSkillTests struct {
	SkillID uint `json:"skill_id" form:"skill_id"`
	Page    int  `json:"page" form:"page"`
	Size    int  `json:"size" form:"size"`
}

// ListSkillTests GET /api/skillbox/skills/test/list
func ListSkillTests(c *ginp.ContextPlus, req *RequestListSkillTests) {
	store, err := sskill.NewStore()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	// list 不需要 ai manager
	svc := sskilltest.New(dbs.GetWriteDb(), dbs.GetReadDb(), store, nil, nil)
	out, err := svc.List(&sskilltest.ListRequest{
		SkillID: req.SkillID,
		Page:    req.Page,
		Size:    req.Size,
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
			Description:   "列出 skill 测试 run,按 skill_id 过滤 + 分页",
			RequestParams: RequestListSkillTests{},
		},
	})
}

// 保留 strconv 引用(后续可能用)
var _ = strconv.Itoa
