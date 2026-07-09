package smarket

import (
	"testing"

	"ginp-api/internal/skillmarket"
)

// 2026-07-09 增:ResolveInstallInput 解析函数单测。
//
// 覆盖:
//   - skillhub URL(/skills/{slug}、/skill/{slug}、api. 前缀)
//   - skills.sh URL(/{owner}/{repo}/{skill})
//   - GitHub tree URL(blob / tree / raw 三种 mode)
//   - raw.githubusercontent.com URL
//   - 纯 slug(配合 sourceHint=skillhub)
//   - owner/repo@skill(配合 sourceHint=skillssh)
//   - 异常输入(空 / 非法域名 / 非法 slug / 缺 owner / README.md / 缺 sourceHint 等)

func TestResolveInstallInput_SkillHubURL(t *testing.T) {
	cases := []struct {
		name      string
		input     string
		wantType  string
		wantRemID string
		wantErr   bool
	}{
		{"skillhub /skills/{slug}", "https://skillhub.cn/skills/code-review", skillmarket.SourceSkillhub, "code-review", false},
		{"skillhub /skill/{slug}", "https://skillhub.cn/skill/code-review", skillmarket.SourceSkillhub, "code-review", false},
		{"skillhub 带尾部斜杠", "https://skillhub.cn/skills/code-review/", skillmarket.SourceSkillhub, "code-review", false},
		{"skillhub 带 query", "https://skillhub.cn/skills/code-review?from=hot", skillmarket.SourceSkillhub, "code-review", false},
		{"skillhub api. 前缀", "https://api.skillhub.cn/skills/code-review", skillmarket.SourceSkillhub, "code-review", false},
		{"skillhub www. 前缀", "https://www.skillhub.cn/skills/code-review", skillmarket.SourceSkillhub, "code-review", false},
		{"skillhub 多级路径兜底", "https://skillhub.cn/u/pskoett/self-improving-agent", skillmarket.SourceSkillhub, "self-improving-agent", false},
		{"skillhub 空 slug", "https://skillhub.cn/skills/", "", "", true},
		{"skillhub 无路径", "https://skillhub.cn", "", "", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := ResolveInstallInput(c.input, "")
			if c.wantErr {
				if err == nil {
					t.Fatalf("期望报错,得到 nil; got=%+v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("不期望报错,得到 %v", err)
			}
			if got.SourceType != c.wantType {
				t.Fatalf("SourceType=%q, want %q", got.SourceType, c.wantType)
			}
			if got.RemoteID != c.wantRemID {
				t.Fatalf("RemoteID=%q, want %q", got.RemoteID, c.wantRemID)
			}
			if got.SourceName == "" {
				t.Fatalf("SourceName 不能为空")
			}
			if got.ResolvedURL == "" {
				t.Fatalf("ResolvedURL 不能为空(URL 输入)")
			}
		})
	}
}

func TestResolveInstallInput_SkillsSHURL(t *testing.T) {
	cases := []struct {
		name      string
		input     string
		wantType  string
		wantRemID string
		wantErr   bool
	}{
		{"标准三段", "https://skills.sh/anthropics/skills/pdf", skillmarket.SourceSkillsSH, "anthropics/skills@pdf", false},
		{"带尾部斜杠", "https://skills.sh/anthropics/skills/pdf/", skillmarket.SourceSkillsSH, "anthropics/skills@pdf", false},
		{"www. 前缀", "https://www.skills.sh/anthropics/skills/pdf", skillmarket.SourceSkillsSH, "anthropics/skills@pdf", false},
		{"路径过短", "https://skills.sh/anthropics", "", "", true},
		{"路径过短无 skill", "https://skills.sh/anthropics/skills", "", "", true},
		{"空路径", "https://skills.sh", "", "", true},
		{"非法 owner", "https://skills.sh/-bad/skills/pdf", "", "", true},
		{"非法 skill", "https://skills.sh/owner/repo/", "", "", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := ResolveInstallInput(c.input, "")
			if c.wantErr {
				if err == nil {
					t.Fatalf("期望报错,得到 nil; got=%+v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("不期望报错,得到 %v", err)
			}
			if got.SourceType != skillmarket.SourceSkillsSH {
				t.Fatalf("SourceType=%q, want %q", got.SourceType, skillmarket.SourceSkillsSH)
			}
			if got.RemoteID != c.wantRemID {
				t.Fatalf("RemoteID=%q, want %q", got.RemoteID, c.wantRemID)
			}
		})
	}
}

func TestResolveInstallInput_GitHubTreeURL(t *testing.T) {
	cases := []struct {
		name      string
		input     string
		wantRemID string
		wantErr   bool
	}{
		{
			"blob 子目录 SKILL.md",
			"https://github.com/anthropics/skills/blob/main/skills/pdf/SKILL.md",
			"anthropics/skills@pdf",
			false,
		},
		{
			"tree 子目录(无 SKILL.md 末段)",
			"https://github.com/anthropics/skills/tree/main/skills/pdf",
			"anthropics/skills@pdf",
			false,
		},
		{
			"blob 根目录 SKILL.md",
			"https://github.com/owner/repo/blob/main/SKILL.md",
			"owner/repo@repo",
			false,
		},
		{
			"raw mode",
			"https://github.com/owner/repo/raw/main/skills/foo/SKILL.md",
			"owner/repo@foo",
			false,
		},
		{
			"master 分支",
			"https://github.com/owner/repo/blob/master/skills/foo/SKILL.md",
			"owner/repo@foo",
			false,
		},
		{
			"嵌套路径",
			"https://github.com/owner/repo/blob/main/skills/sub/deep/SKILL.md",
			"owner/repo@deep",
			false,
		},
		{
			"路径过短",
			"https://github.com/owner/repo",
			"",
			true,
		},
		{
			"非法 mode",
			"https://github.com/owner/repo/issues/1",
			"",
			true,
		},
		{
			"指向 README 报错",
			"https://github.com/owner/repo/blob/main/README.md",
			"",
			true,
		},
		{
			"owner 非法",
			"https://github.com/-bad/repo/blob/main/skills/foo/SKILL.md",
			"",
			true,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := ResolveInstallInput(c.input, "")
			if c.wantErr {
				if err == nil {
					t.Fatalf("期望报错,得到 nil; got=%+v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("不期望报错,得到 %v", err)
			}
			if got.SourceType != skillmarket.SourceSkillsSH {
				t.Fatalf("SourceType=%q, want %q", got.SourceType, skillmarket.SourceSkillsSH)
			}
			if got.RemoteID != c.wantRemID {
				t.Fatalf("RemoteID=%q, want %q", got.RemoteID, c.wantRemID)
			}
			if got.ResolvedURL == "" {
				t.Fatalf("ResolvedURL 不能为空")
			}
		})
	}
}

// 2026-07-09 改:raw.githubusercontent.com 不再支持(skillssh 跟 github 拆开后,raw 不是 skill 入口),
// 整个 raw 测试删除。

func TestResolveInstallInput_NonURLInputsRejected(t *testing.T) {
	// 2026-07-09 改:所有 source 都要求粘详情页 URL,纯 slug / owner/repo@skill 全部拒绝
	cases := []struct {
		name  string
		input string
		hint  string
	}{
		{"纯 slug + skillhub hint", "code-review", "skillhub"},
		{"owner/repo@skill + skillssh hint", "anthropics/skills@pdf", "skillssh"},
		{"纯 slug + skillssh hint", "code-review", "skillssh"},
		{"owner/repo@skill + skillhub hint", "anthropics/skills@pdf", "skillhub"},
		{"纯 slug + 无 hint", "code-review", ""},
		{"非法 slug + skillhub hint", "code review with space", "skillhub"},
		{"空输入", "   ", ""},
		{"空输入 + hint", "", "skillhub"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := ResolveInstallInput(c.input, c.hint)
			if err == nil {
				t.Fatalf("期望 %q 报错(非 URL 输入)", c.input)
			}
		})
	}
}

func TestResolveInstallInput_HintNarrowsDomain(t *testing.T) {
	// 2026-07-09 改:hint 强制限定域名,跨域 URL 必须报错
	cases := []struct {
		name  string
		input string
		hint  string
	}{
		// skillhub tab 不接受 skills.sh URL
		{"skillhub tab + skills.sh URL", "https://skills.sh/anthropics/skills/pdf", "skillhub"},
		// skillhub tab 不接受 github URL
		{"skillhub tab + github URL", "https://github.com/anthropics/skills/blob/main/skills/pdf/SKILL.md", "skillhub"},
		// skills.sh tab 不接受 skillhub URL
		{"skills.sh tab + skillhub URL", "https://skillhub.cn/skills/code-review", "skillssh"},
		// skills.sh tab 不接受 github URL
		{"skills.sh tab + github URL", "https://github.com/anthropics/skills/blob/main/skills/pdf/SKILL.md", "skillssh"},
		// github tab 不接受 skillhub URL
		{"github tab + skillhub URL", "https://skillhub.cn/skills/code-review", "github"},
		// github tab 不接受 skills.sh URL
		{"github tab + skills.sh URL", "https://skills.sh/anthropics/skills/pdf", "github"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := ResolveInstallInput(c.input, c.hint)
			if err == nil {
				t.Fatalf("期望跨域 URL 报错,得到 nil")
			}
		})
	}
}

func TestResolveInstallInput_UnsupportedURL(t *testing.T) {
	cases := []string{
		"https://example.com/skills/foo",
		"https://gist.github.com/foo/bar",
		"https://bitbucket.org/foo/bar",
		"not a url at all but has :// in middle",
	}
	for _, c := range cases {
		t.Run(c, func(t *testing.T) {
			_, err := ResolveInstallInput(c, "")
			if err == nil {
				t.Fatalf("期望 %q 报错", c)
			}
		})
	}
}