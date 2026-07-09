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

func TestResolveInstallInput_RawURL(t *testing.T) {
	cases := []struct {
		name      string
		input     string
		wantRemID string
		wantErr   bool
	}{
		{
			"raw 子目录 SKILL.md",
			"https://raw.githubusercontent.com/anthropics/skills/main/skills/pdf/SKILL.md",
			"anthropics/skills@pdf",
			false,
		},
		{
			"raw 根目录 SKILL.md",
			"https://raw.githubusercontent.com/owner/repo/main/SKILL.md",
			"owner/repo@repo",
			false,
		},
		{
			"raw 路径过短",
			"https://raw.githubusercontent.com/owner/repo/main",
			"",
			true,
		},
		{
			"raw 指向 README",
			"https://raw.githubusercontent.com/owner/repo/main/README.md",
			"",
			true,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := ResolveInstallInput(c.input, "")
			if c.wantErr {
				if err == nil {
					t.Fatalf("期望报错,得到 nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("不期望报错,得到 %v", err)
			}
			if got.RemoteID != c.wantRemID {
				t.Fatalf("RemoteID=%q, want %q", got.RemoteID, c.wantRemID)
			}
		})
	}
}

func TestResolveInstallInput_PlainWithHint(t *testing.T) {
	t.Run("纯 slug + skillhub hint", func(t *testing.T) {
		got, err := ResolveInstallInput("code-review", "skillhub")
		if err != nil {
			t.Fatalf("不期望报错: %v", err)
		}
		if got.SourceType != skillmarket.SourceSkillhub || got.RemoteID != "code-review" {
			t.Fatalf("got=%+v", got)
		}
	})
	t.Run("owner/repo@skill + skillssh hint", func(t *testing.T) {
		got, err := ResolveInstallInput("anthropics/skills@pdf", "skillssh")
		if err != nil {
			t.Fatalf("不期望报错: %v", err)
		}
		if got.SourceType != skillmarket.SourceSkillsSH || got.RemoteID != "anthropics/skills@pdf" {
			t.Fatalf("got=%+v", got)
		}
	})
	t.Run("纯 slug + skillssh hint → 报错", func(t *testing.T) {
		_, err := ResolveInstallInput("code-review", "skillssh")
		if err == nil {
			t.Fatalf("期望报错(必须 owner/repo@skill 格式)")
		}
	})
	t.Run("owner/repo@skill + skillhub hint → 报错", func(t *testing.T) {
		_, err := ResolveInstallInput("anthropics/skills@pdf", "skillhub")
		if err == nil {
			t.Fatalf("期望报错(skillhub 不接受 @ 格式)")
		}
	})
	t.Run("无 hint 也无 URL → 报错", func(t *testing.T) {
		_, err := ResolveInstallInput("code-review", "")
		if err == nil {
			t.Fatalf("期望报错")
		}
	})
	t.Run("非法 slug → 报错", func(t *testing.T) {
		_, err := ResolveInstallInput("code review with space", "skillhub")
		if err == nil {
			t.Fatalf("期望报错")
		}
	})
	t.Run("空输入", func(t *testing.T) {
		_, err := ResolveInstallInput("   ", "")
		if err == nil {
			t.Fatalf("期望报错")
		}
	})
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