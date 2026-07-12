package cskill

import (
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"ginp-api/internal/gapi/service/skill/sskill"
	"ginp-api/pkg/ginp"
)

// RespondStoreInfo 返 store 物理根目录的绝对路径。
// 2026-07-03 增:供前端"在文件夹中打开"拼绝对路径用。
// 之前首页右键分组 / 未选中 skill 时,前端拿不到 store root,只能把相对
// 路径(如 "frontend/code-review" 或 "frontend")直接传给 fsutil.Reveal,
// 桌面端 hook 收到非绝对路径 → os.Stat 失败 → 500。暴露这个轻量接口后,
// 前端可以拿到绝对根再拼相对段,reveal 行为对齐详情区 openInFolder。
//
// 2026-07-12 增:返回 HomeDir + GlobalAgentRoot —— Skill 作用域面板的"全局 Agent"
// folder 按钮需要直接打开 ~/.agents/skills/ 共享池根目录,前端拿不到 home
// 绝对路径,通过该字段透传。EvalSymlinks 后跨平台一致(macOS 是
// /private/var/.../Users/x;Linux /home/x;Windows C:\Users\x)。
type RespondStoreInfo struct {
	StoreRoot       string `json:"store_root"`
	HomeDir         string `json:"home_dir"`
	GlobalAgentRoot string `json:"global_agent_root"`
}

// GetStoreInfo GET /api/skillbox/skills/store-info
func GetStoreInfo(c *ginp.ContextPlus) {
	store, err := sskill.NewStore()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	home, herr := os.UserHomeDir()
	homeDir := ""
	if herr == nil {
		homeDir = home
	}
	globalAgentRoot := ""
	if homeDir != "" {
		// 跟 skillstore.resolveGlobalSourcePath 内部约定一致:
		// home + .agents/skills(不调 EvalSymlinks —— 共享池一般是软链,
		// reveal 在 macOS 软链也能直接命中 .agents/skills 真实路径)。
		globalAgentRoot = filepath.Join(homeDir, ".agents", "skills")
	}
	c.JSON(200, RespondStoreInfo{
		StoreRoot:       store.Root(),
		HomeDir:         homeDir,
		GlobalAgentRoot: globalAgentRoot,
	})
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/skills/store-info",
		Handler:        ginp.BindHandler(GetStoreInfo),
		HttpType:       ginp.HttpGet,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.skills.storeInfo",
		Swagger: &ginp.SwaggerInfo{
			Title:       "skills.storeInfo",
			Description: "返回 skillstore 物理根目录的绝对路径(供前端在文件夹中打开分组/未选中 skill 时拼绝对路径)",
		},
	})
}
