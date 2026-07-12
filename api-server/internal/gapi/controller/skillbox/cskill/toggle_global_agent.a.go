package cskill

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"ginp-api/internal/gapi/service/skill/sskill"
	"ginp-api/internal/skillstore"
	"ginp-api/pkg/ginp"
	"ginp-api/pkg/logger"

	"github.com/gin-gonic/gin"
)

// RequestToggleGlobalAgent 把 skill 切换为「全局 Agent」状态。
//
// 设计依据(2026-07-12):
//   后端不维护独立的 global_agent 字段 —— "全局 Agent" 的单一事实是磁盘上
//   ~/.agents/skills/<name>/SKILL.md 是否存在。store.buildTreeNode 每次 ListTree
//   都实时检测该路径,store 是"只读系统",不缓存"曾经导入过"的历史状态。
//
//   用户在 Skill 作用域面板上点 "全局 Agent" tag:
//     - enabled=true  → 把当前 skill 的所有文件镜像到 ~/.agents/skills/<name>/
//     - enabled=false → 删除 ~/.agents/skills/<name>/ 整个目录
//
//   写盘后无需任何 store 状态变更 —— 下次 ListTree 自动 detect 到新状态,
//   左侧 skill 卡片 tag 与本面板 tag 自动同步。
//
// 入参:
//   name       - skill name(必填;SKILL.md manifest.name,跟目录名一致)
//   version    - skill version(可选,只用于日志)
//   group_path - skill 在 store 内的分组相对路径(2026-07-12 增:支持多级分组;
//                旧版只用 name 走 store.Load 只能命中根下直接子目录,
//                像 "frontend/react/use-cache" 这类嵌套 skill 必然 404)
//   enabled    - true=开启镜像, false=删除目录
type RequestToggleGlobalAgent struct {
	Name      string `json:"name" form:"name"`
	Version   string `json:"version" form:"version"`
	GroupPath string `json:"group_path" form:"group_path"`
	Enabled   bool   `json:"enabled" form:"enabled"`
}

// RespondToggleGlobalAgent 响应。
type RespondToggleGlobalAgent struct {
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
	Path    string `json:"path"` // 镜像目录绝对路径(~/.agents/skills/<name>);enabled=false 时为空
}

// ToggleGlobalAgent POST /api/skillbox/skills/global-agent/toggle
//
// 行为:
//   - enabled=true  → 从 store 读 Canonical,把所有 file 写到 ~/.agents/skills/<name>/
//   - enabled=false → os.RemoveAll 删 ~/.agents/skills/<name/>(目录不存在也 ok)
//
// 失败情形:
//   - name 为空 → 400
//   - store 里找不到该 skill → 404(用户没法标记一个未导入的 skill)
//   - 写盘 / 删盘 IO 错误 → 500
func ToggleGlobalAgent(c *ginp.ContextPlus, req *RequestToggleGlobalAgent) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		c.JSON(400, gin.H{"error": "name is required"})
		return
	}

	store, err := sskill.NewStore()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	// 全局 Agent 路径固定为 ~/.agents/skills/<name>
	// 不走 skilladapter.All() 多 candidate 拼装 —— skillbox 的"全局 Agent"定义
	// 跟 adapter 无关(adapter 都是共用 ~/.agents/skills 这个共享池)。
	target, terr := globalAgentDir(name)
	if terr != nil {
		c.JSON(500, gin.H{"error": terr.Error()})
		return
	}

	if !req.Enabled {
		// 关闭:删目录。允许目录本就不存在(idempotent),视为成功。
		if rerr := os.RemoveAll(target); rerr != nil {
			logger.Error("toggle-global-agent: remove %s failed: %v", target, rerr)
			c.JSON(500, gin.H{"error": rerr.Error()})
			return
		}
		c.SuccessData(RespondToggleGlobalAgent{
			Name:    name,
			Enabled: false,
			Path:    "",
		}, "global-agent disabled")
		return
	}

	// 开启:从 store 读 canonical,把所有 file 镜像写盘。
	// 2026-07-12 改:用 LoadByPath 支持多级分组 —— 旧版 store.Load(name) 只
	// 命中根下直接子目录,分组下的 skill(例如 "frontend/react/use-cache")
	// 100% 报 ErrNotFound。
	canonical, lerr := store.LoadByPath(req.GroupPath, name)
	if lerr != nil {
		if errors.Is(lerr, skillstore.ErrNotFound) {
			c.JSON(404, gin.H{
				"error":      "skill not found in store: " + req.GroupPath + "/" + name,
				"group_path": req.GroupPath,
				"name":       name,
			})
			return
		}
		logger.Error("toggle-global-agent: load %s/%s failed: %v", req.GroupPath, name, lerr)
		c.JSON(500, gin.H{"error": lerr.Error()})
		return
	}

	// 先清空旧目录(幂等:用户重复开启时不会留下旧版本残留),再 MkdirAll。
	// 用 RemoveAll 而不是 IsNotExist 判断,避免"目录存在但被破坏"的中间态。
	if rerr := os.RemoveAll(target); rerr != nil {
		logger.Error("toggle-global-agent: clean %s failed: %v", target, rerr)
		c.JSON(500, gin.H{"error": rerr.Error()})
		return
	}
	if merr := os.MkdirAll(target, 0o755); merr != nil {
		logger.Error("toggle-global-agent: mkdir %s failed: %v", target, merr)
		c.JSON(500, gin.H{"error": merr.Error()})
		return
	}

	// 镜像所有 file。Path 是相对 skill 根,Join target 后写。
	// file.Path 不允许 ..(防御性,跟 store.Save 的校验一致),filepath.Clean
	// 后必须仍以 <name>/ 开头才不会写到目录外。
	for _, f := range canonical.Files {
		// 兜底:即使 Path 为空也跳过,避免写出意外的物理文件。
		fp := strings.TrimSpace(f.Path)
		if fp == "" {
			continue
		}
		dst := filepath.Join(target, fp)
		// 防止 path 跳出去:clean 后必须以 target + separator 开头
		cleaned := filepath.Clean(dst)
		if !strings.HasPrefix(cleaned, filepath.Clean(target)+string(filepath.Separator)) {
			logger.Warn("toggle-global-agent: skip unsafe path %q (would escape %s)", fp, target)
			continue
		}
		if werr := os.MkdirAll(filepath.Dir(cleaned), 0o755); werr != nil {
			logger.Error("toggle-global-agent: mkdir parent for %s failed: %v", cleaned, werr)
			c.JSON(500, gin.H{"error": werr.Error()})
			return
		}
		if werr := os.WriteFile(cleaned, []byte(f.Content), 0o644); werr != nil {
			logger.Error("toggle-global-agent: write %s failed: %v", cleaned, werr)
			c.JSON(500, gin.H{"error": werr.Error()})
			return
		}
	}

	c.SuccessData(RespondToggleGlobalAgent{
		Name:    name,
		Enabled: true,
		Path:    target,
	}, "global-agent enabled")
}

// globalAgentDir 拼出 ~/.agents/skills/<name> 绝对路径。
// 复用 store.resolveGlobalSourcePath 的路径约定(都基于 home + .agents/skills),
// 保证 toggle 写出去的路径跟 buildTreeNode 实时检测到的路径字面一致。
func globalAgentDir(name string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "", errors.New("resolve home dir failed")
	}
	return filepath.Join(home, ".agents", "skills", name), nil
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/skills/global-agent/toggle",
		Handler:        ginp.BindParamsHandler(ToggleGlobalAgent, &RequestToggleGlobalAgent{}),
		HttpType:       ginp.HttpPost,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.skills.global-agent.toggle",
		Swagger: &ginp.SwaggerInfo{
			Title:         "skills.global-agent.toggle",
			Description:   "切换 skill 的「全局 Agent」状态:enabled=true 把当前 skill 镜像到 ~/.agents/skills/<name>/;enabled=false 删除该目录。",
			RequestParams: RequestToggleGlobalAgent{},
		},
	})
}