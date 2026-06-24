// Package cskillpkg - export_package.a.go
// POST /api/skillbox/pkg/export
//
// 入参: { skills: [{scope, project_id, name, version}, ...], source_app?, source_desc? }
// 行为: 用 sskillpkg.BuildExport 拼 .skillbox zip 字节流,直接返回 application/octet-stream
// 注意: 这是"raw response"端点,不走 BindParamsHandler(避免自动 200 + JSON 包装)
package cskillpkg

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/gin-gonic/gin"
	"ginp-api/internal/db/dbs"
	"ginp-api/internal/gapi/service/skill/sskill"
	"ginp-api/internal/gapi/service/skillpkg/sskillpkg"
	"ginp-api/internal/skillpkg"
	"ginp-api/pkg/ginp"
	"ginp-api/pkg/logger"
)

// RequestExportPackage 导出入参。
type RequestExportPackage struct {
	Skills     []skillpkg.SkillRef `json:"skills"`
	SourceApp  string              `json:"source_app"`
	SourceDesc string              `json:"source_desc"`
}

// ExportPackage POST /api/skillbox/pkg/export
func ExportPackage(c *gin.Context) {
	var req RequestExportPackage
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "bad json: " + err.Error()})
		return
	}
	if len(req.Skills) == 0 {
		c.JSON(400, gin.H{"error": "skills is required (at least 1)"})
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

	data, fails, err := svc.BuildExport(skillpkg.ExportRequest{
		Skills:     req.Skills,
		SourceApp:  req.SourceApp,
		SourceDesc: req.SourceDesc,
	})
	if err != nil {
		// 全部失败时也返 4xx(避免吐个无意义 zip)
		if errors.Is(err, skillpkg.ErrEmptySkills) {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		// 其他:已经返了 partial zip + failures;继续走下载
		logger.Warn("export package partial: %v; failures=%v", err, fails)
	}

	filename := fmt.Sprintf("skillbox-%s.skillbox", time.Now().UTC().Format("20060102-150405"))
	c.Header("Content-Disposition", `attachment; filename="`+filename+`"`)
	c.Header("Content-Type", "application/octet-stream")
	c.Header("X-Skillbox-Partial", fmt.Sprintf("%t", len(fails) > 0))
	if len(fails) > 0 {
		// 部分失败时,在 header 里给一行最常见的失败,方便前端展示
		c.Header("X-Skillbox-Failures", firstN(fails, 5))
	}
	c.Status(200)
	if _, err := io.Copy(c.Writer, bytes.NewReader(data)); err != nil {
		logger.Error("export write: %v", err)
	}
}

func firstN(s []string, n int) string {
	if len(s) <= n {
		return joinAll(s, "; ")
	}
	return joinAll(s[:n], "; ") + fmt.Sprintf(" (+%d more)", len(s)-n)
}

func joinAll(s []string, sep string) string {
	out := ""
	for i, v := range s {
		if i > 0 {
			out += sep
		}
		out += v
	}
	return out
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/pkg/export",
		Handler:        ExportPackage,
		HttpType:       ginp.HttpPost,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.pkg.export",
		Swagger: &ginp.SwaggerInfo{
			Title:         "pkg.export",
			Description:   "导出指定 skill 列表为 .skillbox zip 包,直接返回 application/octet-stream",
			RequestParams: RequestExportPackage{},
		},
	})
}
