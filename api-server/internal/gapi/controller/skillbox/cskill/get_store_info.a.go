package cskill

import (
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
type RespondStoreInfo struct {
	StoreRoot string `json:"store_root"`
}

// GetStoreInfo GET /api/skillbox/skills/store-info
func GetStoreInfo(c *ginp.ContextPlus) {
	store, err := sskill.NewStore()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, RespondStoreInfo{StoreRoot: store.Root()})
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
