package smarket

import (
	"testing"

	"ginp-api/internal/skillmarket"
)

// 2026-07-09 增:回归测试,防止 . 等特殊字符导致 CreateGroup 建目录跟
// Manifest.GroupPath 不一致(lock 路径 ENOENT bug)。
func TestNormalizeGroupPathForMarket(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		// 2026-07-10 改:新增 "skillhub-cn" 兼容旧 "skillhub"(DB 老记录)
		{"skillhub-cn", "skillhub-cn"},
		{"skillhub", "skillhub"}, // 老 sourceType 仍认
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
		// 2026-07-10 改:SourceSkillhub 改成 "skillhub-cn",对应分组目录也是 "skillhub-cn"。
		// "skillhub" 老字符串保留兼容(DB 老记录),但走旧 ID 时分组也写老名,不会跟新名冲突。
		{skillmarket.SourceSkillhub, "skillhub-cn"}, // "skillhub-cn"
		{"skillhub", "skillhub"},                   // 兼容老 type
		{"skillssh", "skills.sh"},
		{"github", ""}, // 2026-07-09 改:github 不在 defaultGroupPathFor 里(由 deriveGroupPath 按 owner 生成)
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

// 2026-07-09 增:github source 按 owner 分组(用户要求"anthropics/skills@pdf → anthropics 组")
func TestDeriveGroupPath(t *testing.T) {
	cases := []struct {
		name       string
		sourceType string
		remoteID   string
		want       string
	}{
		// 2026-07-10 改:SourceSkillhub 改 "skillhub-cn" 后,deriveGroupPath 也返 "skillhub-cn"。
		{"skillhub-cn 固定", "skillhub-cn", "code-review", "skillhub-cn"},
		{"skillhub 兼容", "skillhub", "code-review", "skillhub"},
		{"skillssh 固定", "skillssh", "anthropics/skills@pdf", "skills.sh"},
		{"github 按 owner", "github", "anthropics/skills@skills/pdf", "anthropics"},
		{"github 其它 owner", "github", "vercel-labs/agent-skills@vercel-react-best-practices", "vercel-labs"},
		{"github 缺 owner 兜底", "github", "no-owner", "github"},
		{"未知 source", "unknown", "foo", ""},
		{"空", "", "", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := deriveGroupPath(c.sourceType, c.remoteID)
			if got != c.want {
				t.Fatalf("deriveGroupPath(%q, %q) = %q, want %q", c.sourceType, c.remoteID, got, c.want)
			}
		})
	}
}

// 2026-07-09 增:splitOwnerFromRemote 单元测试
func TestSplitOwnerFromRemote(t *testing.T) {
	cases := []struct {
		in       string
		wantOwner string
		wantOK   bool
	}{
		{"anthropics/skills@skills/pdf", "anthropics", true},
		{"vercel-labs/agent-skills@vercel-react-best-practices", "vercel-labs", true},
		{"anthropics/skills@pdf", "anthropics", true},
		{"no-at-sign-but-has-slash", "", false}, // 字符串里只有 - 没 /,拆不出
		{"slug", "", false},     // 没 @ 也没 /,拆不出 owner
		{"/repo@skill", "", false}, // 第一个字符就是 /,slash=0
		{"owner/@skill", "", false}, // 末段是空(空 skill name,但拆 owner 应该成功;这个 case 视实现而定)
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			owner, ok := splitOwnerFromRemote(c.in)
			if ok != c.wantOK {
				t.Fatalf("ok=%v, want %v", ok, c.wantOK)
			}
			if ok && owner != c.wantOwner {
				t.Errorf("owner=%q, want %q", owner, c.wantOwner)
			}
		})
	}
}