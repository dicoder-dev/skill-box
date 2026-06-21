// Package cskilltest - get_skill_test.a.go
// GET /api/skillbox/skills/test/get?id=
package cskilltest

import (
	"errors"

	"github.com/gin-gonic/gin"
	"ginp-api/internal/db/dbs"
	"ginp-api/internal/gapi/service/skill/sskill"
	"ginp-api/internal/gapi/service/skilltester/sskilltest"
	"ginp-api/pkg/ginp"
	"ginp-api/pkg/logger"
)

// RequestGetSkillTest 详情入参。
type RequestGetSkillTest struct {
	ID uint `json:"id" form:"id"`
}

// GetSkillTest GET /api/skillbox/skills/test/get
func GetSkillTest(c *ginp.ContextPlus, req *RequestGetSkillTest) {
	store, err := sskill.NewStore()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	svc := sskilltest.New(dbs.GetWriteDb(), dbs.GetReadDb(), store, nil, nil)
	detail, err := svc.Get(req.ID)
	if err != nil {
		if errors.Is(err, sskilltest.ErrNotFound) {
			c.JSON(404, gin.H{"error": "test run not found"})
			return
		}
		logger.Error("skill test get: %v", err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, detail)
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/skills/test/get",
		Handler:        ginp.BindParamsHandler(GetSkillTest, &RequestGetSkillTest{}),
		HttpType:       ginp.HttpGet,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.skills.test.get",
		Swagger: &ginp.SwaggerInfo{
			Title:         "skills.test.get",
			Description:   "拿一次 skill 测试 run 详情 + 关联 results",
			RequestParams: RequestGetSkillTest{},
		},
	})
}
