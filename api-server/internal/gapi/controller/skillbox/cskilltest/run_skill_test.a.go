// Package cskilltest - run_skill_test.a.go
// POST /api/skillbox/skills/test/run
//
// 入参: { scope, project_id, name, version, trigger?, options?: {...} }
// 行为: 跑 static + script + ai,落 skill_test_runs + skill_test_results,返回 Run + Results
package cskilltest

import (
	"errors"

	"github.com/gin-gonic/gin"
	"ginp-api/internal/aiengine"
	"ginp-api/internal/db/dbs"
	"ginp-api/internal/gapi/service/skill/sskill"
	"ginp-api/internal/gapi/service/skilltester/sskilltest"
	"ginp-api/internal/settings"
	"ginp-api/internal/skilltester"
	"ginp-api/pkg/ginp"
	"ginp-api/pkg/logger"
)

// RequestRunSkillTest 测试入参。
type RequestRunSkillTest struct {
	Scope     string                  `json:"scope"`
	ProjectID uint                    `json:"project_id"`
	Name      string                  `json:"name"`
	Version   string                  `json:"version"`
	Trigger   string                  `json:"trigger"`
	Options   *skilltester.Options    `json:"options,omitempty"`
	// AIProvider 走查用 provider(冗余在 options 里也支持,顶层方便前端)
	AIProvider string `json:"ai_provider,omitempty"`
	// ScriptCommand / ScriptTimeoutSec 同上
	ScriptCommand    string `json:"script_command,omitempty"`
	ScriptTimeoutSec int    `json:"script_timeout_sec,omitempty"`
}

// RunSkillTest POST /api/skillbox/skills/test/run
func RunSkillTest(c *ginp.ContextPlus, req *RequestRunSkillTest) {
	store, err := sskill.NewStore()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	st := settings.New(dbs.GetWriteDb(), dbs.GetReadDb())
	mgr := aiengine.NewManager(nil) // 不需要 secret,因为 BuildFromConfig 不解析
	_ = mgr
	// 改用 sai.NewManager(st) 注入真 secret
	mgr = sskilltest.NewManagerForTester(st)
	svc := sskilltest.New(dbs.GetWriteDb(), dbs.GetReadDb(), store, st, mgr)

	opts := skilltester.Options{}
	if req.Options != nil {
		opts = *req.Options
	}
	// 顶层字段覆盖
	if req.AIProvider != "" {
		opts.AIProvider = req.AIProvider
	}
	if req.ScriptCommand != "" {
		opts.ScriptCommand = req.ScriptCommand
	}
	if req.ScriptTimeoutSec > 0 {
		opts.ScriptTimeoutSec = req.ScriptTimeoutSec
	}

	out, err := svc.Run(&sskilltest.RunRequest{
		Scope:     req.Scope,
		ProjectID: req.ProjectID,
		Name:      req.Name,
		Version:   req.Version,
		Trigger:   req.Trigger,
		Options:   opts,
	})
	if err != nil {
		switch {
		case errors.Is(err, sskilltest.ErrEmptyKey):
			c.JSON(400, gin.H{"error": err.Error()})
		case errors.Is(err, sskilltest.ErrNotFound):
			c.JSON(404, gin.H{"error": "skill not found in db"})
		case errors.Is(err, sskilltest.ErrStoreLoad):
			c.JSON(500, gin.H{"error": "store load: " + err.Error()})
		default:
			logger.Error("skill test run: %v", err)
			c.JSON(500, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(200, out)
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/skills/test/run",
		Handler:        ginp.BindParamsHandler(RunSkillTest, &RequestRunSkillTest{}),
		HttpType:       ginp.HttpPost,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.skills.test.run",
		Swagger: &ginp.SwaggerInfo{
			Title:         "skills.test.run",
			Description:   "跑一次 skill 测试(static + script + ai),落库",
			RequestParams: RequestRunSkillTest{},
		},
	})
}
