package cskill

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"ginp-api/internal/gapi/service/skill/sskill"
	"ginp-api/internal/skillstore"
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
//
// 2026-06-29 改:为支持多级分组,响应新增 `tree` 字段(嵌套 TreeNode 数组),
// 同时保留旧的 `items` + `total` 字段(扁平,供未升级的前端兼容用)。
//
// 树形结构定义见 skillstore.TreeNode:每个节点有 name / path / is_group / children
// (分组时) / skill_meta(叶子时);前端直接消费。
func ListSkills(c *ginp.ContextPlus, req *RequestListSkills) {
	store, err := sskill.NewStore()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	svc := sskill.New(store)
	tree, lerr := svc.ListTree(req.Keyword)
	if lerr != nil {
		logger.Error("skill list: %v", lerr)
		c.JSON(500, gin.H{"error": lerr.Error()})
		return
	}
	// 扁平列表(供旧调用方 / 搜索过滤使用)— 从树拍平
	flattens := flattenTree(tree)
	// 给每个 item 注入 applied_tools(global scope 命中的 tool_id 列表),
	// 避免前端 N+1 调 scope-status。
	enriched := make([]map[string]any, 0, len(flattens))
	for _, it := range flattens {
		row := map[string]any{
			"name":        it.SkillMeta.Name,
			"version":     it.SkillMeta.Version,
			"description": it.SkillMeta.Description,
			"triggers":    it.SkillMeta.Triggers,
			"path":        it.Path,
			"group_path":  groupPathOf(it.Path),
		}
		if it.SkillMeta.UpdatedAt != "" {
			row["updated_at"] = it.SkillMeta.UpdatedAt
		}
		row["applied_tools"] = GlobalAppliedTools(it.SkillMeta.Name)
		enriched = append(enriched, row)
	}
	// 分页(兼容字段)— 现在是文件扫描,实际一次性返回
	page := req.Page
	if page <= 0 {
		page = 1
	}
	size := req.Size
	if size <= 0 {
		size = 20
	}
	total := len(enriched)
	start := (page - 1) * size
	end := start + size
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}
	c.JSON(200, gin.H{
		"items": enriched[start:end],
		"total": total,
		"page":  page,
		"size":  size,
		"tree":  tree,
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
			Description:   "列出 skill(树形),支持 keyword 模糊匹配 + 分页;数据来源是 ~/.skill-box/skills/<group>/<name>/SKILL.md",
			RequestParams: RequestListSkills{},
		},
	})
}

// flattenTree 把 TreeNode 数组拍平(只取 skill 叶子),保持 List 旧行为的顺序。
func flattenTree(nodes []skillstore.TreeNode) []skillstore.TreeNode {
	var out []skillstore.TreeNode
	for _, n := range nodes {
		if !n.IsGroup {
			out = append(out, n)
			continue
		}
		out = append(out, flattenTree(n.Children)...)
	}
	return out
}

// groupPathOf 从 skill 完整 path 反推 group_path(去掉最后一段叶子名)。
func groupPathOf(fullPath string) string {
	gp, _ := sskill.SplitPath(fullPath)
	return gp
}

var _ = strconv.Itoa
