package ctag

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"ginp-api/internal/gapi/service/skillaudit/sskillaudit"
	"ginp-api/pkg/ginp"
	"ginp-api/pkg/logger"
)

// RequestDiffTag diff 请求。query + body 兼容。
type RequestDiffTag struct {
	SkillID    uint `json:"skill_id" form:"skill_id"`
	LeftTagID  uint `json:"left_tag_id" form:"left_tag_id"`
	RightTagID uint `json:"right_tag_id" form:"right_tag_id"`
}

// RespondDiffTag 响应。
type RespondDiffTag = sskillaudit.DiffOutput

// DiffTag GET /api/skillbox/skills/tags/diff
//
// 对比两个视图(0 = current 状态,>0 = 该 tag 的文件)的文件差异。
// 不传 right_tag_id 时默认对应当前状态(便于"tag vs current")。
func DiffTag(c *ginp.ContextPlus, req *RequestDiffTag) {
	// 兼容 query string
	if req.SkillID == 0 {
		if s := c.Query("skill_id"); s != "" {
			if n, err := strconv.ParseUint(s, 10, 64); err == nil {
				req.SkillID = uint(n)
			}
		}
	}
	if req.LeftTagID == 0 {
		if s := c.Query("left_tag_id"); s != "" {
			if n, err := strconv.ParseUint(s, 10, 64); err == nil {
				req.LeftTagID = uint(n)
			}
		}
	}
	if req.RightTagID == 0 {
		if s := c.Query("right_tag_id"); s != "" {
			if n, err := strconv.ParseUint(s, 10, 64); err == nil {
				req.RightTagID = uint(n)
			}
		}
	}
	svc := newService()
	out, err := svc.Diff(&sskillaudit.DiffInput{
		SkillID:    req.SkillID,
		LeftTagID:  req.LeftTagID,
		RightTagID: req.RightTagID,
	})
	if err != nil {
		logger.Error("tag diff: %v", err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, out)
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/skills/tags/diff",
		Handler:        ginp.BindParamsHandler(DiffTag, &RequestDiffTag{}),
		HttpType:       ginp.HttpGet,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.skills.tags.diff",
		Swagger: &ginp.SwaggerInfo{
			Title:         "skills.tags.diff",
			Description:   "对比两个 tag / current 的文件差异",
			RequestParams: RequestDiffTag{},
		},
	})
}
