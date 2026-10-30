package cproject

import (
	"errors"
	"path/filepath"
	"time"

	"ginp-api/internal/db/dbs"
	"ginp-api/internal/gapi/service/project/sproject"
	"ginp-api/internal/skilladapter"
	"ginp-api/pkg/ginp"
	"ginp-api/pkg/logger"

	"github.com/gin-gonic/gin"
)

// RequestScanProject 扫描单个项目被哪些 Agent / skill 引用。
//
// 设计要点(2026-06-29):
//   - 纯读接口,每次请求重扫磁盘;不写 DB(对齐 scope-status 的策略,避免"用户刚 cp skill"看不到)。
//   - 复用 skilladapter.BaseAdapter.Scan + DiscoverPaths(ScopeProject),不新写扫描逻辑;
//     symlink 解析也走 EvalSymlinks(SourceDir 字段已是真实路径)。
//   - agents 数组只返回"该项目中至少命中 1 个 skill"的 Agent,0 命中不进数组 → 卡片 UI 更干净。
type RequestScanProject struct {
	ProjectID uint `json:"project_id" form:"project_id"`
}

// scannedSkill 单条 skill 的展示信息。
//
// SourcePath 是 EvalSymlinks 后的真实磁盘绝对路径(由 BaseAdapter.readSkillDir 写入),
// 既给前端展示,也便于日后"在文件夹打开该 skill"按钮直接用 platform.fs.reveal。
type scannedSkill struct {
	Name       string `json:"name"`
	SourcePath string `json:"source_path"`
}

// scannedAgentSkill Agent 维度的聚合,前端以 chip(claude5 样式)展示。
//
// Skills 字段一并返回,避免前端再 N+1 调接口取详情;count 直接 len(skills) 即可,
// 但保留独立字段便于日后扩展(比如按"是否启用"过滤后 count != len(skills))。
type scannedAgentSkill struct {
	AgentID     string         `json:"agent_id"`
	DisplayName string         `json:"display_name"`
	Icon        string         `json:"icon"`
	Count       int            `json:"count"`
	Skills      []scannedSkill `json:"skills"`
}

// RespondScanProject 响应。
type RespondScanProject struct {
	ProjectID uint                `json:"project_id"`
	ScannedAt time.Time           `json:"scanned_at"`
	Agents    []scannedAgentSkill `json:"agents"`
}

// ScanProject GET /api/skillbox/projects/scan?project_id=N
func ScanProject(c *ginp.ContextPlus, req *RequestScanProject) {
	if req.ProjectID == 0 {
		c.JSON(400, gin.H{"error": "project_id is required"})
		return
	}

	// 1. 取项目根
	svc := sproject.New(dbs.GetWriteDb(), dbs.GetReadDb())
	p, err := svc.GetByID(req.ProjectID)
	if err != nil {
		if errors.Is(err, sproject.ErrNotFound) {
			c.JSON(404, gin.H{"error": "project not found"})
			return
		}
		logger.Error("project scan: getByID %d: %v", req.ProjectID, err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	root := p.RootPath
	if root == "" {
		c.JSON(400, gin.H{"error": "project root_path is empty"})
		return
	}

	// 2. 对每个 adapter 跑 project scope 扫描
	adapters := skilladapter.All()
	out := RespondScanProject{
		ProjectID: req.ProjectID,
		ScannedAt: time.Now(),
		Agents:    []scannedAgentSkill{},
	}
	for _, a := range adapters {
		rels, err := a.DiscoverPaths(skilladapter.ScopeProject)
		if err != nil {
			logger.Warn("project scan: %s DiscoverPaths(project) failed: %v", a.AgentID(), err)
			continue
		}
		var skills []scannedSkill
		for _, rel := range rels {
			abs := filepath.Join(root, rel)
			cans, err := a.Scan(abs)
			if err != nil {
				// Scan 内部已经把"目录不存在"当 nil,这里非空 err 通常是权限问题
				logger.Warn("project scan: %s Scan(%s) failed: %v", a.AgentID(), abs, err)
				continue
			}
			for _, k := range cans {
				skills = append(skills, scannedSkill{
					Name:       k.Manifest.Name,
					SourcePath: k.SourceDir,
				})
			}
		}
		if len(skills) == 0 {
			continue // 该 Agent 在本项目下没 skill → 不进 agents 数组
		}
		out.Agents = append(out.Agents, scannedAgentSkill{
			AgentID:     a.AgentID(),
			DisplayName: a.DisplayName(),
			Icon:        a.Icon(),
			Count:       len(skills),
			Skills:      skills,
		})
	}

	c.SuccessData(out, "scan ok")
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/projects/scan",
		Handler:        ginp.BindParamsHandler(ScanProject, &RequestScanProject{}),
		HttpType:       ginp.HttpGet,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.projects.scan",
		Swagger: &ginp.SwaggerInfo{
			Title:         "projects.scan",
			Description:   "扫描指定项目被哪些 Agent / skill 引用,纯文件系统检查,每次请求都重扫(不读 DB)。",
			RequestParams: RequestScanProject{},
		},
	})
}
