// Package conboarding - post_onboarding_import_zip_bytes.a.go
//
// POST /api/skillbox/onboarding/import-zip-bytes
//
// 2026-07-01 增:Web 端(无桌面 fs picker)用,前端 <input type="file"> 选 zip 后,
// 把字节流直接 POST 到这个 endpoint。Body 是 application/octet-stream。
//
// 为什么单独一个 endpoint:Web 端没有绝对路径,只能拿 File 对象;为了避免同一个
// endpoint 兼容 JSON 和 octet-stream 两种入参(逻辑容易错),拆成两个。
//
// 响应:跟 /import-local 同构(JSON),前端可共用渲染。
package conboarding

import (
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"ginp-api/internal/skillpkg"
	"ginp-api/internal/skillstore"
	"ginp-api/pkg/ginp"
	"ginp-api/pkg/logger"
)

// PostOnboardingImportZipBytes 入口。
func PostOnboardingImportZipBytes(c *ginp.ContextPlus) {
	// 上限 256 MB,够单 zip 装若干 skill;超过直接 413。
	const maxZipBytes = 256 << 20
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxZipBytes)

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(400, gin.H{"error": "read body: " + err.Error()})
		return
	}
	defer c.Request.Body.Close()
	if len(body) == 0 {
		c.JSON(400, gin.H{"error": "empty body; expected zip bytes"})
		return
	}

	store, err := skillstore.New()
	if err != nil {
		logger.Error("import-zip-bytes: store init failed: %v", err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	out, err := skillpkg.ImportFromZipBytes(store, body)
	if err != nil {
		switch {
		case errors.Is(err, skillpkg.ErrNoSkillMD):
			c.JSON(400, gin.H{"error": err.Error()})
		default:
			logger.Error("import-zip-bytes: %v", err)
			c.JSON(500, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(200, out)
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/onboarding/import-zip-bytes",
		Handler:        ginp.BindHandler(PostOnboardingImportZipBytes),
		HttpType:       ginp.HttpPost,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.onboarding.importZipBytes",
		Swagger: &ginp.SwaggerInfo{
			Title:       "onboarding.importZipBytes",
			Description: "接收 application/octet-stream 的 zip 字节流,识别 SKILL.md 并落地 store(Web 端用)",
		},
	})
}