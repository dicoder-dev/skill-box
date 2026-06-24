// Package cskillpkg - preview_package.a.go
// POST /api/skillbox/pkg/preview
//
// 入参: .skillbox zip 字节流(application/octet-stream)
// 行为: 只解析 manifest,返回 skill 索引,用于前端"导入前预览包内容"
package cskillpkg

import (
	"io"

	"github.com/gin-gonic/gin"
	"ginp-api/internal/db/dbs"
	"ginp-api/internal/gapi/service/skill/sskill"
	"ginp-api/internal/gapi/service/skillpkg/sskillpkg"
	"ginp-api/pkg/ginp"
)

// PreviewPackage POST /api/skillbox/pkg/preview
func PreviewPackage(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(400, gin.H{"error": "read body: " + err.Error()})
		return
	}
	defer c.Request.Body.Close()
	if len(body) == 0 {
		c.JSON(400, gin.H{"error": "empty body"})
		return
	}

	store, err := sskill.NewStore()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	svc := sskillpkg.New(dbs.GetWriteDb(), dbs.GetReadDb(), func() (*sskill.Service, error) {
		return sskill.New(store), nil
	})
	mf, err := svc.ParseManifest(body)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, mf)
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/pkg/preview",
		Handler:        PreviewPackage,
		HttpType:       ginp.HttpPost,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.pkg.preview",
		Swagger: &ginp.SwaggerInfo{
			Title:         "pkg.preview",
			Description:   "接收 .skillbox zip,只解析 manifest 返回 skill 索引",
		},
	})
}
