package conboarding

import (
	"sync"
	"time"

	"ginp-api/internal/skilladapter"
	"ginp-api/pkg/ginp"
)

// onboardingCache 用于把"上次 scan 的 Report"在包内共享,避免 import 时再跑一次 scan。
// 进程内单例;并发安全(读多写少,用 RWMutex)。
var onboardingCache = struct {
	sync.RWMutex
	lastReport *skillimporterReportEnvelope
}{}

// skillimporterReportEnvelope 包一层时间戳,避免直接暴露 importer.Report。
type skillimporterReportEnvelope struct {
	ScannedAt time.Time             `json:"scanned_at"`
	Tools     []string              `json:"tools"`
	ToolPaths []string              `json:"tool_paths"`
	Summary   map[string]int        `json:"summary"`
	Found     []onboardingFoundLite `json:"found"`
	HasReport bool                  `json:"has_report"`
}

// onboardingFoundLite 状态接口只暴露轻量字段,避免把整个 canonical 倾到前端。
type onboardingFoundLite struct {
	ToolID     string `json:"tool_id"`
	ToolName   string `json:"tool_name"`
	Name       string `json:"name"`
	Version    string `json:"version"`
	SourcePath string `json:"source_path"`
}

// RequestOnboardingStatus 无入参。
type RequestOnboardingStatus struct{}

// RespondOnboardingStatus onboarding 当前状态。
type RespondOnboardingStatus struct {
	Adapters  []adapterStatus `json:"adapters"`
	LastScan  *time.Time      `json:"last_scan,omitempty"`
	HasReport bool            `json:"has_report"`
	TotalFound int             `json:"total_found"`
}

type adapterStatus struct {
	ToolID      string `json:"tool_id"`
	DisplayName string `json:"display_name"`
	Icon        string `json:"icon"`
	GlobalPath  string `json:"global_path"`
	GlobalOK    bool   `json:"global_ok"`
}

// GetOnboardingStatus 返回所有已注册 adapter 的状态(目录是否存在)+ 上次 scan 摘要。
func GetOnboardingStatus(c *ginp.ContextPlus, _ *RequestOnboardingStatus) {
	onboardingCache.RLock()
	cached := onboardingCache.lastReport
	onboardingCache.RUnlock()

	adapters := make([]adapterStatus, 0, len(skilladapter.All()))
	for _, a := range skilladapter.All() {
		paths, _ := a.DiscoverPaths(skilladapter.ScopeGlobal)
		s := adapterStatus{
			ToolID:      a.ToolID(),
			DisplayName: a.DisplayName(),
			Icon:        a.Icon(),
		}
		if len(paths) > 0 {
			s.GlobalPath = paths[0]
			s.GlobalOK = pathExists(paths[0])
		}
		adapters = append(adapters, s)
	}

	resp := RespondOnboardingStatus{Adapters: adapters}
	if cached != nil {
		t := cached.ScannedAt
		resp.LastScan = &t
		resp.HasReport = true
		resp.TotalFound = len(cached.Found)
	}
	// 走标准业务信封 {code, msg, data},前端默认拦截器据此剥离 data。
	c.SuccessData(resp, "onboarding status ok")
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/onboarding/status",
		Handler:        ginp.BindParamsHandler(GetOnboardingStatus, &RequestOnboardingStatus{}),
		HttpType:       ginp.HttpGet,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.onboarding.status",
		Swagger: &ginp.SwaggerInfo{
			Title:         "onboarding.status",
			Description:   "返回 5 个 adapter 的发现状态 + 上次扫描摘要",
			RequestParams: RequestOnboardingStatus{},
		},
	})
}
