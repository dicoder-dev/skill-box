package smarket

import "testing"

// 2026-07-09 增:回归测试,防止 . 等特殊字符导致 CreateGroup 建目录跟
// Manifest.GroupPath 不一致(lock 路径 ENOENT bug)。
func TestNormalizeGroupPathForMarket(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"skillhub", "skillhub"},
		{"skills.sh", "skills-sh"}, // 关键 case:点 → 短横
		{"github", "github"},
		{"Skills.sh", "skills-sh"}, // 大小写归一
		{"SKILLHUB", "skillhub"},
		{"", ""},
		{"  skills.sh  ", "skills-sh"},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			got := normalizeGroupPathForMarket(c.in)
			if got != c.want {
				t.Fatalf("normalizeGroupPathForMarket(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

func TestDefaultGroupPathFor(t *testing.T) {
	cases := []struct {
		sourceType, want string
	}{
		{"skillhub", "skillhub"},
		{"skillssh", "skills.sh"},
		{"github", "github"},
		{"unknown", ""},
		{"", ""},
	}
	for _, c := range cases {
		t.Run(c.sourceType, func(t *testing.T) {
			got := defaultGroupPathFor(c.sourceType)
			if got != c.want {
				t.Fatalf("defaultGroupPathFor(%q) = %q, want %q", c.sourceType, got, c.want)
			}
		})
	}
}