package ctag

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"ginp-api/internal/gapi/service/skillaudit/sskillaudit"
	"ginp-api/pkg/ginp"
	"ginp-api/pkg/logger"
)

// RequestDiffTag diff 请求。2026-06-24 改造:用 (scope, name) 定位 skill。
type RequestDiffTag struct {
	Scope      string `json:"scope" form:"scope"`
	Name       string `json:"name" form:"name"`
	LeftTagID  uint   `json:"left_tag_id" form:"left_tag_id"`
	RightTagID uint   `json:"right_tag_id" form:"right_tag_id"`
}

// RespondDiffTag 响应。
type RespondDiffTag = sskillaudit.DiffOutput

// DiffTag GET /api/skillbox/skills/tags/diff
func DiffTag(c *ginp.ContextPlus, req *RequestDiffTag) {
	if req.Scope == "" {
		req.Scope = c.Query("scope")
	}
	if req.Name == "" {
		req.Name = c.Query("name")
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
		Scope:      req.Scope,
		Name:       req.Name,
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
			Description:   "对比两个 tag / current 的文件差异;用 scope+name 定位",
			RequestParams: RequestDiffTag{},
		},
	})
}
