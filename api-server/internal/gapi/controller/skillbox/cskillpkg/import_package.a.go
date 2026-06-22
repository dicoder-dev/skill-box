// Package cskillpkg - import_package.a.go
// POST /api/skillbox/pkg/import
//
// 入参: 原始 .skillbox zip 字节流(application/octet-stream)+ 查询参数 target_scope / project_id / skills
// 行为: 解析 → 选条目 → sskill.Service.Create 装入;返回 JSON 导入汇总
//
// 为什么 raw body:zip 走 multipart 太啰嗦,直接 octet-stream + 查询参数表达选择更轻
package cskillpkg

import (
	"errors"
	"io"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"ginp-api/internal/db/dbs"
	"ginp-api/internal/gapi/service/skill/sskill"
	"ginp-api/internal/gapi/service/skillpkg/sskillpkg"
	"ginp-api/internal/skilladapter"
	"ginp-api/internal/skillpkg"
	"ginp-api/pkg/ginp"
	"ginp-api/pkg/logger"
)

// RequestImportPackage 导入 JSON 入参(给"只走 JSON"的客户端用,优先 raw body)。
type RequestImportPackage struct {
	TargetScope string                     `json:"target_scope"`
	ProjectID   uint                       `json:"project_id"`
	Skills      []skillpkg.ImportSkillEntry `json:"skills"`
}

// ImportPackage POST /api/skillbox/pkg/import
func ImportPackage(c *gin.Context) {
	targetScope := c.Query("target_scope")
	if targetScope == "" {
		targetScope = skilladapter.ScopeGlobal
	}
	projectIDStr := c.Query("project_id")
	var projectID uint
	if projectIDStr != "" {
		v, err := strconv.ParseUint(projectIDStr, 10, 64)
		if err != nil {
			c.JSON(400, gin.H{"error": "invalid project_id: " + err.Error()})
			return
		}
		projectID = uint(v)
	}
	// 可选: ?skills=alpha@0.1.0,beta@1.0.0
	var selected []skillpkg.ImportSkillEntry
	if sel := strings.TrimSpace(c.Query("skills")); sel != "" {
		for _, k := range strings.Split(sel, ",") {
			k = strings.TrimSpace(k)
			if k == "" {
				continue
			}
			selected = append(selected, skillpkg.ImportSkillEntry{Key: k})
		}
	}

	// 读 body(zip 字节)
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(400, gin.H{"error": "read body: " + err.Error()})
		return
	}
	defer c.Request.Body.Close()
	if len(body) == 0 {
		c.JSON(400, gin.H{"error": "empty body; expected .skillbox zip"})
		return
	}

	store, err := sskill.NewStore()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	svc := sskillpkg.New(dbs.GetWriteDb(), dbs.GetReadDb(), func() (*sskill.Service, error) {
		return sskill.New(dbs.GetWriteDb(), dbs.GetReadDb(), store), nil
	})

	out, err := svc.Import(body, skillpkg.ImportRequest{
		TargetScope: targetScope,
		ProjectID:   projectID,
		Skills:      selected,
	})
	if err != nil {
		switch {
		case errors.Is(err, skillpkg.ErrEmptySkills),
			errors.Is(err, skillpkg.ErrInvalidManifest),
			errors.Is(err, skillpkg.ErrInvalidSkillMeta),
			errors.Is(err, skillpkg.ErrUnknownSkillKey):
			c.JSON(400, gin.H{"error": err.Error()})
		default:
			logger.Error("import package: %v", err)
			c.JSON(500, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(200, out)
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/pkg/import",
		Handler:        ImportPackage,
		HttpType:       ginp.HttpPost,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.pkg.import",
		Swagger: &ginp.SwaggerInfo{
			Title:         "pkg.import",
			Description:   "接收 application/octet-stream 的 .skillbox zip,解析后装入 store。查询参数 target_scope=global|project,可选 project_id / skills=key1,key2",
			RequestParams: gin.H{},
		},
	})
}
