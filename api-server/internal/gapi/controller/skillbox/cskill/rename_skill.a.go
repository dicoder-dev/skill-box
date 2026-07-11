package cskill

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
	"ginp-api/internal/gapi/service/skill/sskill"
	"ginp-api/pkg/ginp"
	"ginp-api/pkg/logger"
)

// RequestRenameSkill 重命名 skill 入参(2026-07-11 增)。
// src_group_path 是 skill 所在的分组相对路径(可空 = 根);old_name / new_name 都是
// 单段名(走 NormalizeName 规约)。新名必须 ≠ 旧名。
type RequestRenameSkill struct {
	SrcGroupPath string `json:"src_group_path" form:"src_group_path"`
	OldName      string `json:"old_name" form:"old_name"`
	NewName      string `json:"new_name" form:"new_name"`
}

// RenameSkill POST /api/skillbox/skills/rename
//
// 2026-07-11 增:为支持"文档右键重命名"。只改 SKILL.md 所在目录的最后一段名
// (group_path 不变),底层是同分组内 os.Rename,O(1) 几乎无 IO。
//   - 非法名(归一化后为空)→ 400
//   - 新名 = 旧名 → 400
//   - 源不存在 → 404
//   - 同名冲突(目标目录已存在)→ 409 { code: 'target_exists' }
func RenameSkill(c *ginp.ContextPlus, req *RequestRenameSkill) {
	store, err := sskill.NewStore()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	svc := sskill.New(store)
	newPath, rerr := svc.RenameSkill(req.SrcGroupPath, req.OldName, req.NewName)
	if rerr != nil {
		// 非法(空名 / 归一化失败 / 新旧同名)
		if errors.Is(rerr, sskill.ErrEmptyName) || errors.Is(rerr, sskill.ErrInvalidSkillName) {
			c.JSON(400, gin.H{"error": rerr.Error()})
			return
		}
		// 源不存在
		if errors.Is(rerr, sskill.ErrNotFound) {
			c.JSON(404, gin.H{"error": rerr.Error()})
			return
		}
		msg := rerr.Error()
		// store 层错误(包了 ErrNotFound)用字符串兜底识别
		if strings.Contains(msg, "not found") || strings.Contains(msg, "no such file") {
			c.JSON(404, gin.H{"error": msg})
			return
		}
		// 同名冲突
		if strings.Contains(msg, "already exists") {
			c.JSON(409, gin.H{"error": msg, "code": "target_exists"})
			return
		}
		logger.Error("skill rename: %v", rerr)
		c.JSON(500, gin.H{"error": msg})
		return
	}
	c.JSON(200, gin.H{"ok": true, "new_skill_path": newPath})
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/skills/rename",
		Handler:        ginp.BindParamsHandler(RenameSkill, &RequestRenameSkill{}),
		HttpType:       ginp.HttpPost,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.skills.rename",
		Swagger: &ginp.SwaggerInfo{
			Title:         "skills.rename",
			Description:   "在同分组内重命名 skill 目录名(group_path 不变);新名非法 → 400,源不存在 → 404,同名冲突 → 409",
			RequestParams: RequestRenameSkill{},
		},
	})
}
