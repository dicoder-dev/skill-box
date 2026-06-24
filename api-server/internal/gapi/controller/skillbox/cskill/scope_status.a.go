package cskill

import (
	"os"
	"path/filepath"
	"strconv"

	"ginp-api/internal/db/dbs"
	"ginp-api/internal/gapi/service/project/sproject"
	"ginp-api/internal/skilladapter"
	"ginp-api/pkg/ginp"
	"ginp-api/pkg/logger"

	"github.com/gin-gonic/gin"
)

// RequestSkillScopeStatus 实时扫描所有已注册 adapter 的路径,返回某 skill 在
// (tool, scope, project) 笛卡尔积下哪些位置真实存在 SKILL.md。
//
// 关键设计(2026-06-24):
//   - 不存数据库,纯文件系统检查;每次请求都重扫,保证"用户刚把 skill 拷到
//     ~/.codex/skills/xxx"也能立即看到,不需要 import 流程。
//   - 入参只接受 name(必填)+ version(可选,只用于日志/未来扩展;目录名只
//     取决于 name,不影响判断)。
//   - 返回结构: tools 数组列出所有已知工具(供前端渲染第一行),hits 数组
//     给出 (tool, scope, project_id) 组合 + exists + 绝对路径,前端用 exists
//     字段决定 chip 高亮。
type RequestSkillScopeStatus struct {
	Name    string `json:"name" form:"name"`
	Version string `json:"version" form:"version"`
}

// RespondSkillScopeStatus 响应。
type RespondSkillScopeStatus struct {
	Name    string             `json:"name"`
	Version string             `json:"version"`
	Tools   []scopeStatusTool  `json:"tools"`
	Hits    []scopeStatusHit   `json:"hits"`
	Projects []scopeStatusProject `json:"projects"`
}

type scopeStatusTool struct {
	ToolID      string `json:"tool_id"`
	DisplayName string `json:"display_name"`
	Icon        string `json:"icon"`
}

type scopeStatusProject struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
	Alias string `json:"alias"`
	RootPath string `json:"root_path"`
}

type scopeStatusHit struct {
	ToolID      string `json:"tool_id"`
	Scope       string `json:"scope"`        // global | project
	ProjectID   uint   `json:"project_id"`   // 0 表示 global
	ProjectLabel string `json:"project_label,omitempty"` // 拼好的展示文本(仅 project 命中用)
	Path        string `json:"path"`         // 候选 skill 目录(不含 name 子目录)
	Resolved    string `json:"resolved"`     // 实际检查的绝对路径(<path>/<name>/)
	Exists      bool   `json:"exists"`
	IsSystem    bool   `json:"is_system"`    // 候选路径是否属 adapter 的 system 档(只读参考)
}

// GetSkillScopeStatus GET /api/skillbox/skills/scope-status?name=&version=
func GetSkillScopeStatus(c *ginp.ContextPlus, req *RequestSkillScopeStatus) {
	name := req.Name
	if name == "" {
		c.JSON(400, gin.H{"error": "name is required"})
		return
	}
	version := req.Version

	// 1. 列出所有已注册 adapter(顺序稳定,已排序)
	adapters := skilladapter.All()
	tools := make([]scopeStatusTool, 0, len(adapters))
	for _, a := range adapters {
		tools = append(tools, scopeStatusTool{
			ToolID:      a.ToolID(),
			DisplayName: a.DisplayName(),
			Icon:        a.Icon(),
		})
	}

	// 2. 列出所有项目(取 id/name/alias/root_path;root_path 用来拼 project scope 绝对路径)
	projects := []scopeStatusProject{}
	hits := []scopeStatusHit{}
	svc := sproject.New(dbs.GetWriteDb(), dbs.GetReadDb())
	if list, err := svc.List(sproject.ListQuery{Page: 1, Size: 500}); err == nil && list != nil {
		for _, p := range list.Items {
			if p == nil || p.ID == 0 {
				continue
			}
			projects = append(projects, scopeStatusProject{
				ID:       p.ID,
				Name:     p.Name,
				Alias:    p.Alias,
				RootPath: p.RootPath,
			})
		}
	} else if err != nil {
		logger.Warn("scope-status: list projects failed: %v", err)
	}

	// 3. 对每个 adapter,扫 global + project 候选路径
	for _, a := range adapters {
		// Global:DiscoverPaths 返回绝对路径(直接用)
		if globalPaths, err := a.DiscoverPaths(skilladapter.ScopeGlobal); err == nil {
			for _, p := range globalPaths {
				resolved := filepath.Join(p, name)
				hits = append(hits, scopeStatusHit{
					ToolID:   a.ToolID(),
					Scope:    skilladapter.ScopeGlobal,
					ProjectID: 0,
					Path:     p,
					Resolved: resolved,
					Exists:   skillDirExists(resolved),
					IsSystem: a.IsSystemPath(p),
				})
			}
		} else {
			logger.Warn("scope-status: %s DiscoverPaths(global) failed: %v", a.ToolID(), err)
		}

		// Project:DiscoverPaths 返回相对路径(每个项目一条命中)
		if projRels, err := a.DiscoverPaths(skilladapter.ScopeProject); err == nil {
			for _, p := range projects {
				root := p.RootPath
				if root == "" {
					continue
				}
				for _, rel := range projRels {
					abs := filepath.Join(root, rel)
					resolved := filepath.Join(abs, name)
					label := p.Alias
					if label == "" {
						label = p.Name
					}
					if label == "" {
						label = "#" + strconv.FormatUint(uint64(p.ID), 10)
					}
					hits = append(hits, scopeStatusHit{
						ToolID:       a.ToolID(),
						Scope:        skilladapter.ScopeProject,
						ProjectID:    p.ID,
						ProjectLabel: label,
						Path:         abs,
						Resolved:     resolved,
						Exists:       skillDirExists(resolved),
						IsSystem:     false,
					})
				}
			}
		} else {
			logger.Warn("scope-status: %s DiscoverPaths(project) failed: %v", a.ToolID(), err)
		}
	}

	c.SuccessData(RespondSkillScopeStatus{
		Name:     name,
		Version:  version,
		Tools:    tools,
		Projects: projects,
		Hits:     hits,
	}, "scope-status ok")
}

// skillDirExists 检查 <resolved> 是否是存在的 skill 目录(自身含 SKILL.md 或含子目录里有 SKILL.md)。
//
// 实现:必须 <resolved>/SKILL.md 存在(对齐 BaseAdapter.readSkillDir 入口);
// 不做深度递归 — scope-status 关心"我有没有放在工具期望的位置",不是
// "我有没有放在任意子目录里"。后者是 Scan 的活,scope-status 故意不做,
// 避免误报(用户随手建的子目录不应该算"已应用")。
func skillDirExists(resolved string) bool {
	if resolved == "" {
		return false
	}
	st, err := os.Stat(resolved)
	if err != nil || !st.IsDir() {
		return false
	}
	if _, err := os.Stat(filepath.Join(resolved, "SKILL.md")); err == nil {
		return true
	}
	return false
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/skills/scope-status",
		Handler:        ginp.BindParamsHandler(GetSkillScopeStatus, &RequestSkillScopeStatus{}),
		HttpType:       ginp.HttpGet,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.skills.scope-status",
		Swagger: &ginp.SwaggerInfo{
			Title:         "skills.scope-status",
			Description:   "实时扫描所有 adapter 的 (global + project) 路径,返回该 skill 在哪些位置真实存在。",
			RequestParams: RequestSkillScopeStatus{},
		},
	})
}
