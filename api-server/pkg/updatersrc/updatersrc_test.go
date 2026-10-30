package updatersrc

import (
	"reflect"
	"testing"
)

func TestNormalizeLocale(t *testing.T) {
	cases := map[string]string{
		"zh-CN":   LocaleZH,
		"zh":      LocaleZH,
		"zh_CN":   LocaleZH,
		"ZH-Hans": LocaleZH,
		"en-US":   LocaleEN,
		"en":      LocaleEN,
		"en_GB":   LocaleEN,
		"":        "",
		"ja-JP":   "",
		"zh-Hant": LocaleZH,
	}
	for in, want := range cases {
		if got := NormalizeLocale(in); got != want {
			t.Errorf("NormalizeLocale(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestIsChinese(t *testing.T) {
	if !IsChinese("zh-CN") {
		t.Error("zh-CN 应该是中文")
	}
	if IsChinese("en-US") {
		t.Error("en-US 不应该是中文")
	}
	if IsChinese("") {
		t.Error("空 locale 不应该判成中文")
	}
}

func TestPreferredIDs(t *testing.T) {
	if got := PreferredIDs("zh-CN"); !reflect.DeepEqual(got, []string{SourceGitee, SourceGitHub}) {
		t.Errorf("中文源顺序 = %v, 期望 [gitee github]", got)
	}
	if got := PreferredIDs("en-US"); !reflect.DeepEqual(got, []string{SourceGitHub, SourceGitee}) {
		t.Errorf("英文源顺序 = %v, 期望 [github gitee]", got)
	}
	// 未知语言走默认(英文优先)
	if got := PreferredIDs("ja-JP"); !reflect.DeepEqual(got, []string{SourceGitHub, SourceGitee}) {
		t.Errorf("未知语言源顺序 = %v, 期望 [github gitee]", got)
	}
}

func TestOrderByLocale(t *testing.T) {
	sources := DefaultSources()
	zh := Order(sources, "zh-CN")
	if len(zh) != 2 || zh[0].ID != SourceGitee || zh[1].ID != SourceGitHub {
		t.Errorf("中文排序 = %v, 期望 gitee 在前", ids(zh))
	}
	en := Order(sources, "en-US")
	if len(en) != 2 || en[0].ID != SourceGitHub || en[1].ID != SourceGitee {
		t.Errorf("英文排序 = %v, 期望 github 在前", ids(en))
	}
}

func TestOrderKeepsUnmatchedSources(t *testing.T) {
	// 只配了 legacy 源(无 gitee/github ID)→ 不能被偏好规则丢掉
	sources := []Source{{ID: "legacy-0", ManifestURL: "https://example.com/a.json"}}
	out := Order(sources, "zh-CN")
	if len(out) != 1 || out[0].ID != "legacy-0" {
		t.Errorf("legacy 源排序丢了 = %v", ids(out))
	}
}

func TestManifestURLs(t *testing.T) {
	sources := DefaultSources()
	zh := ManifestURLs(sources, "zh-CN")
	if len(zh) != 2 {
		t.Fatalf("中文 manifest urls 数量 = %d, 期望 2", len(zh))
	}
	if zh[0] != sources[0].ManifestURL {
		t.Errorf("中文首选 url = %q, 期望 %q", zh[0], sources[0].ManifestURL)
	}
	en := ManifestURLs(sources, "en-US")
	if en[0] != sources[1].ManifestURL {
		t.Errorf("英文首选 url = %q, 期望 %q", en[0], sources[1].ManifestURL)
	}
}

func TestParseSources(t *testing.T) {
	raw := "gitee=https://gitee.com/x/y/raw/main/m.json, github=https://raw.githubusercontent.com/x/y/main/m.json"
	got := ParseSources(raw)
	if len(got) != 2 {
		t.Fatalf("解析数量 = %d, 期望 2", len(got))
	}
	if got[0].ID != SourceGitee || got[0].ManifestURL != "https://gitee.com/x/y/raw/main/m.json" {
		t.Errorf("第 1 个源 = %+v", got[0])
	}
	if got[1].ID != SourceGitHub || got[1].ManifestURL != "https://raw.githubusercontent.com/x/y/main/m.json" {
		t.Errorf("第 2 个源 = %+v", got[1])
	}
}

func TestParseSourcesIgnoresBadItems(t *testing.T) {
	got := ParseSources("gitee=https://a/b.json, ,bad-item,github=")
	if len(got) != 1 || got[0].ID != SourceGitee {
		t.Errorf("非法项应被忽略, 实际 = %+v", got)
	}
}

func TestParseSourcesDuplicateOverride(t *testing.T) {
	got := ParseSources("gitee=https://a/1.json,gitee=https://a/2.json")
	if len(got) != 1 {
		t.Fatalf("同 ID 应合并, 实际数量 = %d", len(got))
	}
	if got[0].ManifestURL != "https://a/2.json" {
		t.Errorf("同 ID 应后者覆盖, 实际 = %q", got[0].ManifestURL)
	}
}

func TestParseLegacyURLs(t *testing.T) {
	got := ParseLegacyURLs("https://a/1.json, https://a/2.json ,")
	if len(got) != 2 || got[0] != "https://a/1.json" || got[1] != "https://a/2.json" {
		t.Errorf("旧配置解析 = %v", got)
	}
	if ParseLegacyURLs("   ") != nil {
		t.Error("空白旧配置应返 nil")
	}
}

func TestSourcesEnvOverride(t *testing.T) {
	t.Setenv(EnvSources, "gitee=https://gitee.com/x/y/raw/main/m.json")
	got := Sources()
	if len(got) != 1 || got[0].ID != SourceGitee {
		t.Errorf("env 覆盖失效 = %+v", got)
	}
}

func TestSourcesLegacyFallback(t *testing.T) {
	t.Setenv(EnvSources, "")
	t.Setenv(EnvURLs, "https://a/1.json,https://a/2.json")
	got := Sources()
	if len(got) != 2 {
		t.Fatalf("旧配置应解析出 2 个源, 实际 = %d", len(got))
	}
	// legacy 源没有 gitee/github ID,按语言排序时保持原顺序
	zh := ManifestURLs(got, "zh-CN")
	if zh[0] != "https://a/1.json" || zh[1] != "https://a/2.json" {
		t.Errorf("legacy 源排序应保持原顺序 = %v", zh)
	}
}

func TestSourcesDefault(t *testing.T) {
	t.Setenv(EnvSources, "")
	t.Setenv(EnvURLs, "")
	got := Sources()
	if len(got) != len(DefaultSources()) {
		t.Fatalf("默认源数量 = %d, 期望 %d", len(got), len(DefaultSources()))
	}
}

func TestSourceIDOf(t *testing.T) {
	sources := DefaultSources()
	if got := SourceIDOf(sources, sources[0].ManifestURL); got != SourceGitee {
		t.Errorf("SourceIDOf = %q, 期望 %q", got, SourceGitee)
	}
	if got := SourceIDOf(sources, "https://unknown/x.json"); got != "" {
		t.Errorf("未知 url 应返空, 实际 = %q", got)
	}
}

// ids 便于断言输出。
func ids(list []Source) []string {
	out := make([]string, 0, len(list))
	for _, s := range list {
		out = append(out, s.ID)
	}
	return out
}
