package cskill

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"ginp-api/internal/gapi/service/skill/sskill"
	"ginp-api/internal/skilladapter"
	"ginp-api/pkg/ginp"
	"ginp-api/pkg/logger"
)

// RequestListSkills 列表请求。keyword + 分页。
type RequestListSkills struct {
	Keyword string `json:"keyword" form:"keyword"`
	Page    int    `json:"page" form:"page"`
	Size    int    `json:"size" form:"size"`
}

// ListSkills GET /api/skillbox/skills
func ListSkills(c *ginp.ContextPlus, req *RequestListSkills) {
	store, err := sskill.NewStore()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	svc := sskill.New(store)
	items, lerr := svc.List(req.Keyword)
	if lerr != nil {
		logger.Error("skill list: %v", lerr)
		c.JSON(500, gin.H{"error": lerr.Error()})
		return
	}
	// 分页(page/size 仅作为兼容字段保留,但因为是文件扫描,实际一次性返回)
	page := req.Page
	if page <= 0 {
		page = 1
	}
	size := req.Size
	if size <= 0 {
		size = 20
	}
	total := len(items)
	start := (page - 1) * size
	end := start + size
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}
	// 2026-06-25:给每个 item 注入 applied_tools(global scope 命中的 tool_id 列表),
	// 前端列表项直接展示"哪些工具已全局应用",避免 N+1 调 scope-status。
	enriched := make([]map[string]any, 0, end-start)
	for _, it := range items[start:end] {
		row := map[string]any{
			"name":        it.Name,
			"version":     it.Version,
			"description": it.Description,
			"triggers":    it.Triggers,
		}
		if it.Author != "" {
			row["author"] = it.Author
		}
		if it.UpdatedAt != "" {
			row["updated_at"] = it.UpdatedAt
		}
		row["applied_tools"] = GlobalAppliedTools(it.Name)
		enriched = append(enriched, row)
	}
	c.JSON(200, gin.H{
		"items": enriched,
		"total": total,
		"page":  page,
		"size":  size,
	})
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/skills",
		Handler:        ginp.BindParamsHandler(ListSkills, &RequestListSkills{}),
		HttpType:       ginp.HttpGet,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.skills.list",
		Swagger: &ginp.SwaggerInfo{
			Title:         "skills.list",
			Description:   "列出 skill,支持 keyword 模糊匹配 + 分页;数据来源是 ~/.skill-box/skills/<name>/SKILL.md",
			RequestParams: RequestListSkills{},
		},
	})
}

// itoa 暂留(后续分页可能用)
var _ = strconv.Itoa
var _ = skilladapter.ScopeGlobal
