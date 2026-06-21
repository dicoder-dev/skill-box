package ctag

import (
	"errors"

	"github.com/gin-gonic/gin"
	"ginp-api/internal/db/dbs"
	"ginp-api/internal/gapi/service/skill/sskill"
	"ginp-api/internal/gapi/service/skillaudit/sskillaudit"
	"ginp-api/pkg/ginp"
	"ginp-api/pkg/logger"
)

// RequestCreateTag 打 tag 请求。
type RequestCreateTag struct {
	SkillID uint   `json:"skill_id"`
	Tag     string `json:"tag"`
	Message string `json:"message"`
}

// RespondCreateTag 响应。
type RespondCreateTag = sskillaudit.CreateTagOutput

// CreateTag POST /api/skillbox/skills/tags/create
//
// 给一个 skill 打 tag:把当前所有文件固化到 skill_file_snapshots。
func CreateTag(c *ginp.ContextPlus, req *RequestCreateTag) {
	svc := newService()
	out, err := svc.CreateTag(&sskillaudit.CreateTagInput{
		SkillID: req.SkillID,
		Tag:     req.Tag,
		Message: req.Message,
	})
	if err != nil {
		switch {
		case errors.Is(err, sskillaudit.ErrSkillNotFound):
			c.JSON(404, gin.H{"error": err.Error()})
		case errors.Is(err, sskillaudit.ErrInvalidTag):
			c.JSON(400, gin.H{"error": err.Error()})
		case errors.Is(err, sskillaudit.ErrEmptyFiles):
			c.JSON(400, gin.H{"error": err.Error()})
		default:
			logger.Error("tag create: %v", err)
			c.JSON(500, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(200, out)
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/skills/tags/create",
		Handler:        ginp.BindParamsHandler(CreateTag, &RequestCreateTag{}),
		HttpType:       ginp.HttpPost,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.skills.tags.create",
		Swagger: &ginp.SwaggerInfo{
			Title:         "skills.tags.create",
			Description:   "给 skill 打 tag(固化当前文件到 skill_file_snapshots)",
			RequestParams: RequestCreateTag{},
		},
	})
}

// newService 工厂 - 统一从 dbs 取 db + skillstore。
func newService() *sskillaudit.Service {
	ww := dbs.GetWriteDb()
	rr := dbs.GetReadDb()
	store, err := sskill.NewStore()
	if err != nil {
		panic("ctag: skillstore init: " + err.Error())
	}
	return sskillaudit.New(ww, rr, store)
}
