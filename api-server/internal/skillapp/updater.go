package skillapp

import (
	"sort"
	"strings"

	"ginp-api/internal/gapi/entity"
	"ginp-api/internal/skilladapter"
)

// UpdateItem 单个 skill 的更新检测结果。
type UpdateItem struct {
	SkillID         uint   `json:"skill_id"`
	SkillName       string `json:"skill_name"`
	Scope           string `json:"scope"`
	ProjectID       uint   `json:"project_id,omitempty"`
	LocalVersion    string `json:"local_version"`
	MarketSource    string `json:"market_source,omitempty"`
	MarketRemoteID  string `json:"market_remote_id,omitempty"`
	MarketVersion   string `json:"market_version,omitempty"`
	UpdateAvailable bool   `json:"update_available"`
}

// Updater 本地 skill 与三方市场版本对比。
type Updater struct {
	// CompareSemver 简单版本比较(由 service 注入;默认用 semverCmp)
	CompareSemver func(local, remote string) int
}

// NewUpdater 构造 Updater。
func NewUpdater() *Updater {
	return &Updater{CompareSemver: semverCmp}
}

// CheckUpdates 拿"待比较"的两侧数据,产出 UpdateItem 列表。
// 输入:
//   - local: 从 sskill.List 拿到的 []*entity.Skill
//   - market: 从 mmarketskill.FindList 拿到的 []*entity.MarketSkill
//
// 匹配规则:同 source_name + remote_id(如果 local.source=market + SourceRef)
// 或 fallback 到 (scope, name, project_id) 简单匹配。
func (u *Updater) CheckUpdates(local []*entity.Skill, market []*entity.MarketSkill) []UpdateItem {
	// 把 market 按 (source, remote) 建索引;同时按 (name) 兜底索引
	byRemote := map[string]*entity.MarketSkill{}
	byName := map[string][]*entity.MarketSkill{}
	for _, m := range market {
		if m == nil {
			continue
		}
		key := m.SourceName + ":" + m.RemoteID
		byRemote[key] = m
		byName[m.Name] = append(byName[m.Name], m)
	}
	// local 顺序保留(让 UI 稳定)
	out := make([]UpdateItem, 0, len(local))
	seen := map[uint]bool{}
	for _, s := range local {
		if s == nil || seen[s.ID] {
			continue
		}
		seen[s.ID] = true
		ui := UpdateItem{
			SkillID:      s.ID,
			SkillName:    s.Name,
			Scope:        s.Scope,
			ProjectID:    s.ProjectID,
			LocalVersion: s.Version,
		}
		// 1) 优先:source=market + source_ref = "<source>:<remote>"
		if s.Source == "market" && s.SourceRef != "" {
			if m, ok := byRemote[s.SourceRef]; ok {
				ui.MarketSource = m.SourceName
				ui.MarketRemoteID = m.RemoteID
				ui.MarketVersion = m.Version
				ui.UpdateAvailable = u.CompareSemver(ui.LocalVersion, m.Version) < 0
				out = append(out, ui)
				continue
			}
		}
		// 2) 兜底:同名 + 最新一个 market
		if ms, ok := byName[s.Name]; ok && len(ms) > 0 {
			// 按 version 排序(粗排,只取 latest)
			sort.Slice(ms, func(i, j int) bool {
				return u.CompareSemver(ms[i].Version, ms[j].Version) > 0
			})
			m := ms[0]
			ui.MarketSource = m.SourceName
			ui.MarketRemoteID = m.RemoteID
			ui.MarketVersion = m.Version
			ui.UpdateAvailable = u.CompareSemver(ui.LocalVersion, m.Version) < 0
		}
		out = append(out, ui)
	}
	return out
}

// semverCmp 极简 semver 比较(只处理 x.y.z / 带 -prerelease / 带 +build 简化)。
// 返回 -1 / 0 / 1。
// 错误 / 无法解析:走字典序 fallback。
func semverCmp(a, b string) int {
	a, b = strings.TrimSpace(a), strings.TrimSpace(b)
	if a == b {
		return 0
	}
	pa := parseSemver(a)
	pb := parseSemver(b)
	for i := 0; i < 3; i++ {
		if pa[i] != pb[i] {
			if pa[i] < pb[i] {
				return -1
			}
			return 1
		}
	}
	return 0
}

// parseSemver 解析 "x.y.z" → [x, y, z];失败返 [0, 0, 0]。
func parseSemver(s string) [3]int {
	s = strings.SplitN(s, "-", 2)[0]
	s = strings.SplitN(s, "+", 2)[0]
	parts := strings.Split(s, ".")
	var out [3]int
	for i := 0; i < 3 && i < len(parts); i++ {
		n := 0
		for _, r := range parts[i] {
			if r < '0' || r > '9' {
				break
			}
			n = n*10 + int(r-'0')
		}
		out[i] = n
	}
	return out
}

// Suppress unused.
var _ = skilladapter.ScopeGlobal
