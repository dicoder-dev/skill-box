package conboarding

import (
	"os"

	"github.com/gin-gonic/gin"
	"ginp-api/internal/skillimporter"
	"ginp-api/internal/skillstore"
	"ginp-api/pkg/ginp"
	"ginp-api/pkg/logger"
)

// RequestOnboardingScan 无入参(后续可加 scope=global|project)。
type RequestOnboardingScan struct{}

// RespondOnboardingScan scan 后的精简 Report。
type RespondOnboardingScan = skillimporterReportEnvelope

// PostOnboardingScan 跑一次跨工具扫描,把结果缓存到包级变量供后续 import 用。
func PostOnboardingScan(c *ginp.ContextPlus, _ *RequestOnboardingScan) {
	store, err := skillstore.New()
	if err != nil {
		logger.Error("onboarding scan: store init failed: %v", err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	im := skillimporter.New(store)
	report, err := im.Scan("")
	if err != nil {
		logger.Error("onboarding scan: %v", err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	envelope := skillimporterReportEnvelope{
		ScannedAt: report.FinishedAt,
		Tools:     report.Tools,
		Summary:   report.ToolSummary,
		HasReport: true,
	}
	toolPaths := map[string]string{}
	for _, d := range report.Dirs {
		if d.Exists {
			toolPaths[d.ToolID] = d.Path
		}
	}
	for _, tid := range report.Tools {
		if p, ok := toolPaths[tid]; ok {
			envelope.ToolPaths = append(envelope.ToolPaths, p)
		}
	}
	for _, fs := range report.FoundSkills {
		envelope.Found = append(envelope.Found, onboardingFoundLite{
			ToolID:     fs.ToolID,
			ToolName:   fs.ToolName,
			Name:       fs.Canonical.Manifest.Name,
			Version:    fs.Canonical.Manifest.Version,
			SourcePath: fs.SourcePath,
		})
	}
	if envelope.Summary == nil {
		envelope.Summary = map[string]int{}
	}

	onboardingCache.Lock()
	onboardingCache.lastReport = &envelope
	onboardingCache.Unlock()

	logger.Info("onboarding scan: %s", report.String())
	// 走标准业务信封 {code, msg, data},前端默认拦截器据此剥离 data。
	c.SuccessData(envelope, "onboarding scan done")
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/onboarding/scan",
		Handler:        ginp.BindParamsHandler(PostOnboardingScan, &RequestOnboardingScan{}),
		HttpType:       ginp.HttpPost,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.onboarding.scan",
		Swagger: &ginp.SwaggerInfo{
			Title:         "onboarding.scan",
			Description:   "跑一次跨工具扫描并把 Report 缓存到进程,供 import 接口消费",
			RequestParams: RequestOnboardingScan{},
		},
	})
}

func pathExists(p string) bool {
	if p == "" {
		return false
	}
	_, err := os.Stat(p)
	return err == nil
}
