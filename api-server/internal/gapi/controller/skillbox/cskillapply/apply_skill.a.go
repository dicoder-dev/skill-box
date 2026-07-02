package cskillapply

import (
	"errors"

	"github.com/gin-gonic/gin"
	"ginp-api/internal/db/dbs"
	"ginp-api/internal/gapi/service/project/sproject"
	"ginp-api/internal/gapi/service/skill/sskill"
	"ginp-api/internal/gapi/service/skillapp/sskillapp"
	"ginp-api/internal/settings"
	"ginp-api/pkg/ginp"
	"ginp-api/pkg/logger"
)

// RequestApplySkill 单 skill apply 请求。2026-06-24 改造:用 (scope, name) 定位 skill。
type RequestApplySkill struct {
	Scope     string   `json:"scope"`
	ProjectID uint     `json:"project_id"`
	Name      string   `json:"name"`
	Tools     []string `json:"tools"`
}

// RespondApplySkill 响应。
type RespondApplySkill = sskillapp.ApplyResult

// ApplySkill POST /api/skillbox/skills/apply
func ApplySkill(c *ginp.ContextPlus, req *RequestApplySkill) {
	svc := newService()
	out, err := svc.Apply(&sskillapp.ApplyInput{
		Scope:     req.Scope,
		ProjectID: req.ProjectID,
		Name:      req.Name,
		Tools:     req.Tools,
	})
	if err != nil {
		switch {
		case errors.Is(err, sskillapp.ErrSkillNotFound):
			c.JSON(404, gin.H{"error": err.Error()})
		case errors.Is(err, sskillapp.ErrEmptyTools):
			c.JSON(400, gin.H{"error": err.Error()})
		default:
			logger.Error("skill apply: %v", err)
			c.JSON(500, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(200, out)
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/skills/apply",
		Handler:        ginp.BindParamsHandler(ApplySkill, &RequestApplySkill{}),
		HttpType:       ginp.HttpPost,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.skills.apply",
		Swagger: &ginp.SwaggerInfo{
			Title:         "skills.apply",
			Description:   "把一个 skill 落到一个或多个目标工具;用 scope+name 定位",
			RequestParams: RequestApplySkill{},
		},
	})
}

// newService 工厂 - 统一从 dbs 取 db。
//
// 2026-06-29 增:WithProjectService — scope=project 的 apply 需要 sproject 查
// entity.Project.RootPath,把 project_id 解析成真实项目根传给 applier。
// 2026-07-02 增:WithSettings — 让 apply 按 settings.apply_mode 切换 copy/symlink。
func newService() *sskillapp.Service {
	ww := dbs.GetWriteDb()
	rr := dbs.GetReadDb()
	return sskillapp.New(ww, rr, func() (*sskill.Service, error) {
		store, err := sskill.NewStore()
		if err != nil {
			return nil, err
		}
		return sskill.New(store), nil
	}).
		WithProjectService(sproject.New(ww, rr)).
		WithSettings(settings.New(ww, rr))
}
