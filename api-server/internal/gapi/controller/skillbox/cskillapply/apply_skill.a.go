package cskillapply

import (
	"errors"

	"github.com/gin-gonic/gin"
	"ginp-api/internal/db/dbs"
	"ginp-api/internal/gapi/service/skill/sskill"
	"ginp-api/internal/gapi/service/skillapp/sskillapp"
	"ginp-api/pkg/ginp"
	"ginp-api/pkg/logger"
)

// RequestApplySkill 单 skill apply 请求。
type RequestApplySkill struct {
	SkillID   uint     `json:"skill_id"`
	Scope     string   `json:"scope"`
	ProjectID uint     `json:"project_id"`
	Tools     []string `json:"tools"`
}

// RespondApplySkill 响应。
type RespondApplySkill = sskillapp.ApplyResult

// ApplySkill POST /api/skillbox/skills/apply
//
// 把 skill_id 对应的 canonical skill 落到 tools 列表(每个工具一次,失败不互锁)。
// 返回每工具的 ApplyResult + 是否全部成功。
func ApplySkill(c *ginp.ContextPlus, req *RequestApplySkill) {
	svc := newService()
	out, err := svc.Apply(&sskillapp.ApplyInput{
		SkillID:   req.SkillID,
		Scope:     req.Scope,
		ProjectID: req.ProjectID,
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
			Description:   "把一个 skill 落到一个或多个目标工具",
			RequestParams: RequestApplySkill{},
		},
	})
}

// newService 工厂 - 统一从 dbs 取 db。
func newService() *sskillapp.Service {
	ww := dbs.GetWriteDb()
	rr := dbs.GetReadDb()
	return sskillapp.New(ww, rr, func() (*sskill.Service, error) {
		store, err := sskill.NewStore()
		if err != nil {
			return nil, err
		}
		return sskill.New(ww, rr, store), nil
	})
}
