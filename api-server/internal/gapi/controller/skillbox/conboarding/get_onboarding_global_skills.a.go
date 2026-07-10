// Package conboarding - get_onboarding_global_skills.a.go
//
// GET /api/skillbox/onboarding/global-skills
//
// 2026-07-10 增:首页"导入技能"弹窗新增 Tab「全局目录」,这里列出 ~/.agents/skills
// 下所有候选 skill(无依赖缓存),给前端做搜索 + 多选导入。
//
// 路径解析:
//   - 用 skilladapter.All() 遍历已注册 adapter,凡是 DiscoverPaths(scope=global) 返回
//     的 path 含 ".agents/skills"(无论是否带 ~),都作为"全局目录"候选路径。
//   - 取首个存在的目录作为扫描根(去重后),不重复扫。
//
// 响应(JSON):
//   - root:   最终扫描根(磁盘绝对路径),前端展示用
//   - exists: root 是否存在
//   - items:  []GlobalSkillCandidate(name/version/description/source_path/exists)
//
// 错误:
//   - 运行时 stat / walk 失败 → 500 + {error}。
package conboarding

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"ginp-api/internal/skilladapter"
	"ginp-api/internal/skillpkg"
	"ginp-api/pkg/ginp"
	"ginp-api/pkg/logger"
)

// RequestOnboardingGlobalSkills 无入参,GET 请求。
type RequestOnboardingGlobalSkills struct{}

// RespondOnboardingGlobalSkills 响应体。
type RespondOnboardingGlobalSkills struct {
	Root   string                       `json:"root"`   // 实际扫描的磁盘绝对路径
	Exists bool                         `json:"exists"` // root 是否存在
	Items  []skillpkg.GlobalSkillCandidate `json:"items"`  // 候选列表
}

// GetOnboardingGlobalSkills 入口。
func GetOnboardingGlobalSkills(c *ginp.ContextPlus, _ *RequestOnboardingGlobalSkills) {
	root, exists := resolveGlobalSkillsRoot()

	resp := RespondOnboardingGlobalSkills{
		Root:   root,
		Exists: exists,
		Items:  []skillpkg.GlobalSkillCandidate{},
	}
	if !exists {
		// 目录不存在 → 返空列表 + exists=false,前端走"目录未创建"提示路径。
		c.SuccessData(resp, "global skills dir not found")
		return
	}

	items, err := skillpkg.ListGlobalSkills(root)
	if err != nil {
		logger.Error("global-skills: list failed: %v", err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	resp.Items = items
	c.SuccessData(resp, "global skills ok")
}

// resolveGlobalSkillsRoot 拿"全局 skill 目录"的磁盘绝对路径。
//
// 策略:遍历 skilladapter.All(),凡是 DiscoverPaths(scope=global) 返回的 path
// 含 ".agents/skills"(无论是否带 ~),都收集起来,去重后取首个存在的。
//
// 大多数 adapter(claude/codex/trae/cline)都声明 ~/.agents/skills,展开后是
// 同一个 home 路径,这里 EvalSymlinks 后统一返回一条。
func resolveGlobalSkillsRoot() (string, bool) {
	seen := map[string]bool{}
	var candidates []string
	for _, a := range skilladapter.All() {
		paths, _ := a.DiscoverPaths(skilladapter.ScopeGlobal)
		for _, p := range paths {
			if !looksLikeAgentsSkillsPath(p) {
				continue
			}
			// 展开 ~ 到 home,再 EvalSymlinks 拿真实路径(去重)。
			expanded := expandHomeTilde(p)
			if expanded == "" {
				continue
			}
			if real, err := filepath.EvalSymlinks(expanded); err == nil {
				expanded = real
			}
			if seen[expanded] {
				continue
			}
			seen[expanded] = true
			candidates = append(candidates, expanded)
		}
	}
	// 取首个 pathExists 的 candidate。
	for _, p := range candidates {
		if pathExists(p) {
			return p, true
		}
	}
	// 没一个存在就返第一个(让前端展示目录路径 + exists=false)
	if len(candidates) > 0 {
		return candidates[0], false
	}
	// 兜底:用 homeDir + /.agents/skills
	home := osUserHomeDir()
	if home == "" {
		return "", false
	}
	return filepath.Join(home, ".agents", "skills"), false
}

// looksLikeAgentsSkillsPath 判定路径是否指代 ~/.agents/skills。
//
// 兼容形式:
//   - "~/.agents/skills"
//   - "/Users/x/.agents/skills"(已展开)
//   - 末尾可能有 "/"
func looksLikeAgentsSkillsPath(p string) bool {
	if p == "" {
		return false
	}
	p = strings.TrimRight(p, "/")
	if p == "" {
		return false
	}
	// 用 filepath.Base 拿最后一段 + 第二段,避免被前缀干扰
	// 例 "/Users/x/.agents/skills" → base=.agents/skills(base 第二段是 skills)
	// 这里用更朴素的方式:去掉 "~/" 前缀后看是否以 ".agents/skills" 结尾。
	cleaned := strings.TrimPrefix(p, "~/")
	cleaned = strings.TrimPrefix(cleaned, "~")
	return strings.HasSuffix(cleaned, ".agents/skills")
}

// expandHomeTilde 把 "~/" 或 "~" 前缀展开为 home 目录。
// 没有 ~ 前缀时原样返回。
func expandHomeTilde(p string) string {
	if p == "" {
		return p
	}
	if p == "~" || strings.HasPrefix(p, "~/") {
		home := osUserHomeDir()
		if home == "" {
			return ""
		}
		if p == "~" {
			return home
		}
		return filepath.Join(home, strings.TrimPrefix(p, "~/"))
	}
	return p
}

// osUserHomeDir 跨平台拿 home 目录,失败时返 ""。
// 桌面端走 os.UserHomeDir;Web 端若没 HOME 环境变量也会失败 — 此时由前端走
// "目录未发现"分支。
func osUserHomeDir() string {
	h, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return h
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/onboarding/global-skills",
		Handler:        ginp.BindParamsHandler(GetOnboardingGlobalSkills, &RequestOnboardingGlobalSkills{}),
		HttpType:       ginp.HttpGet,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.onboarding.globalSkills",
		Swagger: &ginp.SwaggerInfo{
			Title:         "onboarding.globalSkills",
			Description:   "列出 ~/.agents/skills 下所有候选 skill(无缓存,首页导入弹窗用)",
			RequestParams: RequestOnboardingGlobalSkills{},
		},
	})
}