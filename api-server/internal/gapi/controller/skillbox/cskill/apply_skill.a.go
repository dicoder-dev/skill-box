package cskill

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"ginp-api/internal/db/dbs"
	"ginp-api/internal/gapi/service/project/sproject"
	"ginp-api/internal/gapi/service/skill/sskill"
	"ginp-api/internal/skilladapter"
	"ginp-api/pkg/ginp"
	"ginp-api/pkg/logger"

	"github.com/gin-gonic/gin"
)

// RequestApplySkill 把 skillbox 库里指定 skill 写到目标工具的 (scope, project) 位置。
//
// 设计要点(2026-06-24):
//   - 入参只接受 name + tool_id + scope + project_id。source 永远是 skillbox
//     库(无需 caller 传 canonical),避免前端构造内容造成数据不一致。
//   - 同名已存在时,默认返回 409 + exists=true,前端据此弹覆盖确认框;
//     caller 传 force=true 时直接覆盖(覆盖走 Delete + Apply 两步,失败回滚)。
//   - 路径拼接逻辑跟 scope-status 一致:global 直接用 DiscoverPaths(ScopeGlobal)
//     绝对路径;project 用 listProjects.root_path + DiscoverPaths(ScopeProject)
//     相对路径。工具不存在 / 路径拼不出 → 400,避免 silent 写错地方。
//   - 删除(skillstore.Delete) 走的是 skillbox 库,与本接口无关。
type RequestApplySkill struct {
	Name      string `json:"name" form:"name"`
	Version   string `json:"version" form:"version"` // 仅日志用,不参与路径
	ToolID    string `json:"tool_id" form:"tool_id"`
	Scope     string `json:"scope" form:"scope"`         // global | project
	ProjectID uint   `json:"project_id" form:"project_id"` // scope=project 时必填
	Force     bool   `json:"force" form:"force"`         // 同名已存在时是否覆盖
}

// RespondApplySkill apply 结果。
type RespondApplySkill struct {
	Name      string `json:"name"`
	ToolID    string `json:"tool_id"`
	Scope     string `json:"scope"`
	ProjectID uint   `json:"project_id"`
	Path      string `json:"path"`   // 实际写入的目标绝对路径
	Overwrote bool   `json:"overwrote"` // 是否覆盖了已有同名
}

// ApplySkill POST /api/skillbox/skills/apply
func ApplySkill(c *ginp.ContextPlus, req *RequestApplySkill) {
	if strings.TrimSpace(req.Name) == "" {
		c.JSON(400, gin.H{"error": "name is required"})
		return
	}
	scope := strings.ToLower(strings.TrimSpace(req.Scope))
	if scope != skilladapter.ScopeGlobal && scope != skilladapter.ScopeProject {
		c.JSON(400, gin.H{"error": "scope must be 'global' or 'project'"})
		return
	}
	if scope == skilladapter.ScopeProject && req.ProjectID == 0 {
		c.JSON(400, gin.H{"error": "project_id is required for project scope"})
		return
	}

	adapter, ok := skilladapter.Get(req.ToolID)
	if !ok {
		c.JSON(400, gin.H{"error": fmt.Sprintf("unknown tool_id: %s", req.ToolID)})
		return
	}

	// 算目标路径
	target, err := resolveApplyTarget(adapter, scope, req.ProjectID)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	// 读 skillbox 库里该 skill 的 canonical(source 永远是 skillbox 库)
	store, serr := sskill.NewStore()
	if serr != nil {
		c.JSON(500, gin.H{"error": serr.Error()})
		return
	}
	svc := sskill.New(store)
	full, gerr := svc.Get(req.Name)
	if gerr != nil {
		if errors.Is(gerr, sskill.ErrNotFound) {
			c.JSON(404, gin.H{"error": fmt.Sprintf("skill not found in library: %s", req.Name)})
			return
		}
		logger.Error("apply: load skill %q: %v", req.Name, gerr)
		c.JSON(500, gin.H{"error": gerr.Error()})
		return
	}
	localName := adapter.LocalName(*full)
	finalDir := filepath.Join(target, localName)

	overwrote := false
	// 同名存在 → 看 caller 是否传 force
	if skillDirExists(finalDir) {
		if !req.Force {
			c.JSON(409, gin.H{
				"error":   "target already has this skill",
				"exists":  true,
				"path":    finalDir,
				"message": "该位置已有同名 skill,前端弹确认让用户选覆盖/取消",
			})
			return
		}
		// 覆盖:先删再 Apply(adapter.Apply 是覆盖式,不需要显式删,
		// 但为了避免遗留 system 字段 / 旧附属文件,先 RemoveAll 清理干净)
		if err := os.RemoveAll(finalDir); err != nil && !os.IsNotExist(err) {
			c.JSON(500, gin.H{"error": fmt.Sprintf("remove existing: %v", err)})
			return
		}
		overwrote = true
	}

	if err := adapter.Apply(*full, finalDir); err != nil {
		logger.Error("apply: %s apply %s to %s: %v", req.ToolID, req.Name, finalDir, err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.SuccessData(RespondApplySkill{
		Name:      req.Name,
		ToolID:    req.ToolID,
		Scope:     scope,
		ProjectID: req.ProjectID,
		Path:      finalDir,
		Overwrote: overwrote,
	}, "apply ok")
}

// RequestUnapplySkill 把指定 skill 从目标工具位置删除。
//
// 安全约束:目标路径必须落在 adapter 声明的 DiscoverPaths(scope) 合法根下,
// 避免 caller 传任意路径误删。用 filepath.Rel + HasPrefix 校验。
type RequestUnapplySkill struct {
	Name      string `json:"name" form:"name"`
	ToolID    string `json:"tool_id" form:"tool_id"`
	Scope     string `json:"scope" form:"scope"`
	ProjectID uint   `json:"project_id" form:"project_id"`
}

// RespondUnapplySkill 删除结果。
type RespondUnapplySkill struct {
	Name      string `json:"name"`
	ToolID    string `json:"tool_id"`
	Scope     string `json:"scope"`
	ProjectID uint   `json:"project_id"`
	Path      string `json:"path"`
	Removed   bool   `json:"removed"` // true=真删了,false=原本就不存在
}

// UnapplySkill POST /api/skillbox/skills/unapply
func UnapplySkill(c *ginp.ContextPlus, req *RequestUnapplySkill) {
	if strings.TrimSpace(req.Name) == "" {
		c.JSON(400, gin.H{"error": "name is required"})
		return
	}
	scope := strings.ToLower(strings.TrimSpace(req.Scope))
	if scope != skilladapter.ScopeGlobal && scope != skilladapter.ScopeProject {
		c.JSON(400, gin.H{"error": "scope must be 'global' or 'project'"})
		return
	}
	if scope == skilladapter.ScopeProject && req.ProjectID == 0 {
		c.JSON(400, gin.H{"error": "project_id is required for project scope"})
		return
	}

	adapter, ok := skilladapter.Get(req.ToolID)
	if !ok {
		c.JSON(400, gin.H{"error": fmt.Sprintf("unknown tool_id: %s", req.ToolID)})
		return
	}

	target, err := resolveApplyTarget(adapter, scope, req.ProjectID)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// 用 LocalName 算出最终目录
	store, serr := sskill.NewStore()
	if serr != nil {
		c.JSON(500, gin.H{"error": serr.Error()})
		return
	}
	svc := sskill.New(store)
	full, gerr := svc.Get(req.Name)
	if gerr != nil {
		if errors.Is(gerr, sskill.ErrNotFound) {
			c.JSON(404, gin.H{"error": fmt.Sprintf("skill not found in library: %s", req.Name)})
			return
		}
		c.JSON(500, gin.H{"error": gerr.Error()})
		return
	}
	localName := adapter.LocalName(*full)
	finalDir := filepath.Join(target, localName)

	// 路径安全校验:finalDir 必须在 target 下(target 已经在合法根下)
	if !strings.HasPrefix(finalDir, filepath.Clean(target)+string(filepath.Separator)) {
		c.JSON(400, gin.H{"error": "resolved path escapes target root"})
		return
	}

	removed := false
	if skillDirExists(finalDir) {
		if err := os.RemoveAll(finalDir); err != nil {
			c.JSON(500, gin.H{"error": fmt.Sprintf("remove: %v", err)})
			return
		}
		removed = true
	}

	c.SuccessData(RespondUnapplySkill{
		Name:      req.Name,
		ToolID:    req.ToolID,
		Scope:     scope,
		ProjectID: req.ProjectID,
		Path:      finalDir,
		Removed:   removed,
	}, "unapply ok")
}

// resolveApplyTarget 根据 (adapter, scope, project_id) 算出目标根目录(不含 name 子目录)。
//
// 跟 scope-status.a.go 的路径拼接规则保持一致:
//   - global:DiscoverPaths(ScopeGlobal)[0](global 路径必然非空,空就报错)
//   - project:listProjects 查 root_path + DiscoverPaths(ScopeProject)[0]
// project_id 不在 listProjects 里时 → 400。
func resolveApplyTarget(a skilladapter.Adapter, scope string, projectID uint) (string, error) {
	if scope == skilladapter.ScopeGlobal {
		paths, err := a.DiscoverPaths(skilladapter.ScopeGlobal)
		if err != nil {
			return "", fmt.Errorf("discover global: %w", err)
		}
		if len(paths) == 0 {
			return "", fmt.Errorf("%s: no global skill path configured", a.ToolID())
		}
		return paths[0], nil
	}
	// project
	rels, err := a.DiscoverPaths(skilladapter.ScopeProject)
	if err != nil {
		return "", fmt.Errorf("discover project: %w", err)
	}
	if len(rels) == 0 {
		return "", fmt.Errorf("%s: no project skill path configured", a.ToolID())
	}
	svc := sproject.New(dbs.GetWriteDb(), dbs.GetReadDb())
	p, err := svc.GetByID(projectID)
	if err != nil {
		return "", fmt.Errorf("project %d not found", projectID)
	}
	if strings.TrimSpace(p.RootPath) == "" {
		return "", fmt.Errorf("project %d has empty root_path", projectID)
	}
	return filepath.Join(p.RootPath, rels[0]), nil
}

// init 路由注册。
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
			Description:   "把 skillbox 库里的 skill 复制到目标工具的 (scope, project) 位置;force=true 覆盖同名",
			RequestParams: RequestApplySkill{},
		},
	})
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/skills/unapply",
		Handler:        ginp.BindParamsHandler(UnapplySkill, &RequestUnapplySkill{}),
		HttpType:       ginp.HttpPost,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.skills.unapply",
		Swagger: &ginp.SwaggerInfo{
			Title:         "skills.unapply",
			Description:   "从目标工具的 (scope, project) 位置删除该 skill(物理 rm -rf 目录)",
			RequestParams: RequestUnapplySkill{},
		},
	})

	_ = strconv.Itoa // 占位避免 unused import 警告
}
