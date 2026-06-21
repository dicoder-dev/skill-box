package ctag

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	"ginp-api/internal/gapi/service/skillaudit/sskillaudit"
	"ginp-api/pkg/ginp"
	"ginp-api/pkg/logger"
)

// RequestDeleteTag 删 tag 请求。
type RequestDeleteTag struct {
	TagID uint `json:"tag_id" form:"tag_id"`
}

// RespondDeleteTag 响应。
type RespondDeleteTag struct {
	TagID    uint `json:"tag_id"`
	Deleted  int  `json:"deleted"` // 删除的 file_snapshot 行数
	Remained int  `json:"remained"`
}

// DeleteTag POST /api/skillbox/skills/tags/delete
//
// 删除一个 tag(包括其 file_snapshots)。隐式 tag 也允许删。
func DeleteTag(c *ginp.ContextPlus, req *RequestDeleteTag) {
	if req.TagID == 0 {
		if s := c.Query("tag_id"); s != "" {
			if n, err := strconv.ParseUint(s, 10, 64); err == nil {
				req.TagID = uint(n)
			}
		}
	}
	svc := newService()
	if err := svc.DeleteTag(req.TagID); err != nil {
		if errors.Is(err, sskillaudit.ErrTagNotFound) {
			c.JSON(404, gin.H{"error": err.Error()})
			return
		}
		logger.Error("tag delete: %v", err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, RespondDeleteTag{TagID: req.TagID, Deleted: -1})
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/skills/tags/delete",
		Handler:        ginp.BindParamsHandler(DeleteTag, &RequestDeleteTag{}),
		HttpType:       ginp.HttpPost,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.skills.tags.delete",
		Swagger: &ginp.SwaggerInfo{
			Title:         "skills.tags.delete",
			Description:   "删 tag(包括它的 file_snapshots)",
			RequestParams: RequestDeleteTag{},
		},
	})
}
