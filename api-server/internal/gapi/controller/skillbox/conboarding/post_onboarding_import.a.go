package conboarding

import (
	"time"

	"github.com/gin-gonic/gin"
	"ginp-api/internal/db/dbs"
	"ginp-api/internal/skilladapter"
	"ginp-api/internal/skillimporter"
	"ginp-api/internal/skillstore"
	"ginp-api/pkg/ginp"
	"ginp-api/pkg/logger"
)

// RequestOnboardingImport 入参。
// Items 为空时等价于"全部导入"——前端在 wizard 里点"全部"时直接传空数组即可。
type RequestOnboardingImport struct {
	Items []skillimporter.ImportItem `json:"items"`
}

// RespondOnboardingImport 导入结果汇总。
type RespondOnboardingImport struct {
	ImportedAt time.Time                    `json:"imported_at"`
	Total      int                          `json:"total"`
	OK         int                          `json:"ok"`
	Failed     int                          `json:"failed"`
	Results    []skillimporter.ImportResult `json:"results"`
}

// PostOnboardingImport 消费最近一次 scan 缓存,做选择性导入。
// 必须先调 scan;若缓存为空返回 400。
func PostOnboardingImport(c *ginp.ContextPlus, req *RequestOnboardingImport) {
	onboardingCache.RLock()
	cached := onboardingCache.lastReport
	onboardingCache.RUnlock()
	if cached == nil {
		// 错误路径仍走非信封格式:{error} 不带 code。
		// 前端拦截器在 'code' in data 时才剥信封,这里没有 code,会被原样返回,
		// 调用方 http.post 拿到 {error, status, data} 走错误分支。
		c.JSON(400, gin.H{"error": "no cached scan; call /api/skillbox/onboarding/scan first"})
		return
	}

	store, err := skillstore.New()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	// 传入 dbWrite,importer 在 store.Save 成功后会把 mskill 行 upsert 落库。
	// 之前只写盘不写库,导致前端 listSkills 走 DB 查不到记录,看起来像"成功但数据没了"。
	im := skillimporter.New(store).WithDB(dbs.GetWriteDb())

	// 还原 report(只填 importer.Import 实际用到的字段)
	report := &skillimporter.Report{
		StartedAt:   cached.ScannedAt,
		FinishedAt:  cached.ScannedAt,
		Tools:       cached.Tools,
		ToolSummary: cached.Summary,
	}
	for _, f := range cached.Found {
		// 从 SourcePath 重新读盘拿完整 Canonical(含 SKILL.md + 全部附属文件)。
		// scan 时只缓存了轻量字段(为了不把 SKILL.md 倾到前端),这里必须按
		// SourcePath 重新 ReadSkillDir 一次,否则 import 会丢失 SKILL.md 实际
		// 内容(Importer.NormalizeForStore 兜底成 "<name> skill" 占位货)。
		var c skilladapter.Canonical
		if f.SourcePath != "" {
			if full, err := skilladapter.ReadSkillDir(f.SourcePath); err == nil {
				c = full
			} else {
				// SourcePath 读不到(用户在 scan 后手动删了)—— 兜底用轻量字段,
				// 让 caller 至少能看到一条占位结果而不是整条 import 失败。
				c = skilladapter.Canonical{
					Manifest: skilladapter.Manifest{
						Name:    f.Name,
						Version: f.Version,
					},
				}
			}
		} else {
			c = skilladapter.Canonical{
				Manifest: skilladapter.Manifest{
					Name:    f.Name,
					Version: f.Version,
				},
			}
		}
		report.FoundSkills = append(report.FoundSkills, skillimporter.FoundSkill{
			ToolID:     f.ToolID,
			ToolName:   f.ToolName,
			SourcePath: f.SourcePath,
			Canonical:  c,
		})
	}
	report.TotalFound = len(report.FoundSkills)

	results, err := im.Import(report, req.Items)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	resp := RespondOnboardingImport{
		ImportedAt: time.Now(),
		Total:      len(results),
		Results:    results,
	}
	for _, r := range results {
		if r.OK {
			resp.OK++
		} else {
			resp.Failed++
		}
	}
	logger.Info("onboarding import: total=%d ok=%d failed=%d", resp.Total, resp.OK, resp.Failed)
	// 走标准业务信封 {code, msg, data},前端默认拦截器据此剥离 data。
	c.SuccessData(resp, "onboarding import done")
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/onboarding/import",
		Handler:        ginp.BindParamsHandler(PostOnboardingImport, &RequestOnboardingImport{}),
		HttpType:       ginp.HttpPost,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.onboarding.import",
		Swagger: &ginp.SwaggerInfo{
			Title:         "onboarding.import",
			Description:   "消费上次 scan 缓存,按 items 导入(空 items=全部)",
			RequestParams: RequestOnboardingImport{},
		},
	})
}
