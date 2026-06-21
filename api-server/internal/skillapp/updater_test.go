package skillapp_test

import (
	"testing"

	"ginp-api/internal/gapi/entity"
	"ginp-api/internal/skillapp"
)

func TestSemverCmp_Basic(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"1.0.0", "1.0.0", 0},
		{"1.0.0", "1.0.1", -1},
		{"1.0.1", "1.0.0", 1},
		{"0.1.0", "0.10.0", -1},
		{"2.0.0", "1.99.99", 1},
		{"1.0.0-beta", "1.0.0", 0}, // pre-release 简化掉 → 主版本相等
		{"1.0.0+build", "1.0.0", 0}, // build 简化掉
	}
	for _, c := range cases {
		if got := skillapp.SemverCmpForTest(c.a, c.b); got != c.want {
			t.Errorf("semverCmp(%q, %q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestCheckUpdates_BySourceRef(t *testing.T) {
	local := []*entity.Skill{
		{ID: 1, Name: "alpha", Version: "0.1.0", Scope: "global", Source: "market", SourceRef: "skillhub:alpha-1"},
		{ID: 2, Name: "beta", Version: "0.5.0", Scope: "global", Source: "local", SourceRef: ""},
	}
	market := []*entity.MarketSkill{
		{Name: "alpha", SourceName: "skillhub", RemoteID: "alpha-1", Version: "0.2.0"},
		{Name: "beta", SourceName: "skillhub", RemoteID: "beta-1", Version: "0.5.0"},
		{Name: "gamma", SourceName: "skillhub", RemoteID: "gamma-1", Version: "1.0.0"},
	}
	u := skillapp.NewUpdater()
	items := u.CheckUpdates(local, market)
	if len(items) != 2 {
		t.Fatalf("len = %d, want 2", len(items))
	}
	// alpha: 0.1.0 → 0.2.0 → 更新
	if items[0].SkillName != "alpha" || !items[0].UpdateAvailable {
		t.Errorf("alpha item = %+v", items[0])
	}
	if items[0].MarketSource != "skillhub" || items[0].MarketVersion != "0.2.0" {
		t.Errorf("alpha source/version wrong: %+v", items[0])
	}
	// beta: 0.5.0 → 0.5.0 → 不更新
	if items[1].SkillName != "beta" || items[1].UpdateAvailable {
		t.Errorf("beta item = %+v", items[1])
	}
}

func TestCheckUpdates_FallbackByName(t *testing.T) {
	// 本地没有 source=market,只按名字匹配 + 取最大版本
	local := []*entity.Skill{
		{ID: 1, Name: "alpha", Version: "0.1.0", Scope: "global", Source: "local"},
	}
	market := []*entity.MarketSkill{
		{Name: "alpha", SourceName: "skillssh", RemoteID: "alpha-7", Version: "0.3.0"},
		{Name: "alpha", SourceName: "skillhub", RemoteID: "alpha-3", Version: "0.5.0"},
	}
	u := skillapp.NewUpdater()
	items := u.CheckUpdates(local, market)
	if len(items) != 1 {
		t.Fatalf("len = %d, want 1", len(items))
	}
	if !items[0].UpdateAvailable {
		t.Errorf("expected update available, got %+v", items[0])
	}
	// 应该选最大版本 0.5.0(skillhub)
	if items[0].MarketVersion != "0.5.0" {
		t.Errorf("MarketVersion = %s, want 0.5.0", items[0].MarketVersion)
	}
}

func TestCheckUpdates_NoMatch(t *testing.T) {
	local := []*entity.Skill{
		{ID: 1, Name: "delta", Version: "0.1.0", Scope: "global"},
	}
	market := []*entity.MarketSkill{
		{Name: "echo", Version: "1.0.0"},
	}
	u := skillapp.NewUpdater()
	items := u.CheckUpdates(local, market)
	if len(items) != 1 {
		t.Fatalf("len = %d, want 1", len(items))
	}
	if items[0].UpdateAvailable {
		t.Errorf("expected no update, got %+v", items[0])
	}
	if items[0].MarketVersion != "" {
		t.Errorf("MarketVersion should be empty: %s", items[0].MarketVersion)
	}
}
