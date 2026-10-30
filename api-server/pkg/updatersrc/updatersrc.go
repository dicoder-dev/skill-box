// Package updatersrc 管理应用自升级的"更新源"配置与语言路由。
//
// 背景:
// 应用可检查新版本,链路是"拉远端 manifest.json → 比对版本 → 下载安装包"。
// manifest 与安装包都托管在代码托管站上:国内用户访问 GitHub 慢且不稳,
// 英文用户访问 Gitee 反之,所以按用户当前语言选源:
//   - zh-CN → Gitee 优先,GitHub 兜底
//   - 其他   → GitHub 优先,Gitee 兜底
//
// 为什么放 pkg/ 而不是 internal/:
// 桌面端模块(skill-box/desktop/services)与后端模块(ginp-api)必须共用同一份
// 选源逻辑,而 Go 的 internal 规则禁止跨模块 import internal 包。
//
// "版本指针"约定:
// 每个源托管一份 manifest 指针文件(build/updater/manifest.<source-id>.json),
// 内容除 assets[].urls 指向各自站点外完全一致,由 scripts/release.sh 发版时生成并提交。
// 这样"选源"对整条链路生效(manifest 与安装包同站),不会出现"manifest 从 A 站取、
// 安装包却从 B 站拉"的半截状态;另一站的 manifest 作为兜底时,安装包地址也会自动跟着换站。
package updatersrc

import (
	"os"
	"strconv"
	"strings"
)

// 更新源 ID。
const (
	// SourceGitee Gitee(国内源,中文用户优先)。
	SourceGitee = "gitee"
	// SourceGitHub GitHub(全球源,英文用户优先)。
	SourceGitHub = "github"
)

// 语言标识,与前端 i18n 的 locale 取值保持一致。
const (
	// LocaleZH 简体中文。
	LocaleZH = "zh-CN"
	// LocaleEN 英文。
	LocaleEN = "en-US"
)

// 环境变量名。
const (
	// EnvSources 覆盖默认更新源,格式 "<id>=<manifest url>,<id>=<manifest url>"。
	// 例:SKILLBOX_UPDATER_SOURCES="gitee=https://gitee.com/x/y/raw/main/build/updater/manifest.gitee.json,github=https://raw.githubusercontent.com/x/y/main/build/updater/manifest.github.json"
	EnvSources = "SKILLBOX_UPDATER_SOURCES"
	// EnvURLs 兼容旧配置:逗号分隔的 manifest url 列表,数组顺序即降级顺序。
	// 新配置请用 EnvSources(带源 ID,才能按语言分源)。
	EnvURLs = "SKILLBOX_UPDATER_URLS"
)

// Source 一个更新源。
type Source struct {
	// ID 源标识(gitee / github)。
	ID string
	// ManifestURL 该源的 manifest 指针文件地址。
	ManifestURL string
}

// DefaultSources 返回内置默认源。
//
// 指向仓库里的两份"版本指针"文件(由 scripts/release.sh 发版时生成并提交):
//   - Gitee : https://gitee.com/<owner>/<repo>/raw/<branch>/build/updater/manifest.gitee.json
//   - GitHub: https://raw.githubusercontent.com/<owner>/<repo>/<branch>/build/updater/manifest.github.json
//
// 注意 Gitee 的 raw 地址里的分支名必须是该仓库的实际分支(默认 main),
// 分支名不对会直接 404,发布前请确认镜像仓库的默认分支。
func DefaultSources() []Source {
	return []Source{
		{ID: SourceGitee, ManifestURL: "https://gitee.com/dicoder/skill-box/raw/main/build/updater/manifest.gitee.json"},
		{ID: SourceGitHub, ManifestURL: "https://raw.githubusercontent.com/dicoder-dev/skill-box/main/build/updater/manifest.github.json"},
	}
}

// Sources 返回当前生效的更新源列表。
//
// 优先级:EnvSources(env 覆盖)→ EnvURLs(旧配置,拆成匿名源)→ 内置默认值。
func Sources() []Source {
	if parsed := ParseSources(os.Getenv(EnvSources)); len(parsed) > 0 {
		return parsed
	}
	if legacy := ParseLegacyURLs(os.Getenv(EnvURLs)); len(legacy) > 0 {
		out := make([]Source, 0, len(legacy))
		for i, u := range legacy {
			out = append(out, Source{ID: legacySourceID(i), ManifestURL: u})
		}
		return out
	}
	return DefaultSources()
}

// legacySourceID 给旧配置(只有 url 列表、没有源 ID)生成占位 ID。
// 用 "legacy-0 / legacy-1" 这种明确不会被识别成 gitee/github 的名字,
// 这样按语言排序时它们既不是中文首选也不是英文首选,只按原顺序兜底。
func legacySourceID(i int) string {
	return "legacy-" + strconv.Itoa(i)
}

// ParseSources 解析 EnvSources 格式的字符串。
//
// 格式:"<id>=<url>,<id>=<url>",忽略空白项与没有 "=" 的项。
// 同一 ID 出现多次时后者覆盖前者(保持原位置)。
func ParseSources(raw string) []Source {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var out []Source
	index := map[string]int{}
	for item := range strings.SplitSeq(raw, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		eq := strings.Index(item, "=")
		if eq <= 0 {
			continue
		}
		id := strings.TrimSpace(item[:eq])
		url := strings.TrimSpace(item[eq+1:])
		if id == "" || url == "" {
			continue
		}
		if pos, ok := index[id]; ok {
			out[pos].ManifestURL = url
			continue
		}
		index[id] = len(out)
		out = append(out, Source{ID: id, ManifestURL: url})
	}
	return out
}

// ParseLegacyURLs 解析 EnvURLs 格式的逗号分隔 url 列表。
func ParseLegacyURLs(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var out []string
	for p := range strings.SplitSeq(raw, ",") {
		if v := strings.TrimSpace(p); v != "" {
			out = append(out, v)
		}
	}
	return out
}

// NormalizeLocale 把任意语言标识归一化成 zh-CN / en-US。
//
// 认得的前缀:zh / zho / chi(→ zh-CN),en / eng(→ en-US)。
// 认不出来的返 ""(调用方按"英文/默认"处理,即 GitHub 优先)。
func NormalizeLocale(locale string) string {
	l := strings.ToLower(strings.TrimSpace(locale))
	if l == "" {
		return ""
	}
	l = strings.ReplaceAll(l, "_", "-")
	switch {
	case strings.HasPrefix(l, "zh"), strings.HasPrefix(l, "zho"), strings.HasPrefix(l, "chi"):
		return LocaleZH
	case strings.HasPrefix(l, "en"), strings.HasPrefix(l, "eng"):
		return LocaleEN
	default:
		return ""
	}
}

// IsChinese 判断 locale 是否为中文(用于选择 Gitee 源)。
func IsChinese(locale string) bool {
	return NormalizeLocale(locale) == LocaleZH
}

// PreferredIDs 返回给定语言下的源偏好顺序。
//
// zh-CN → [gitee, github];其余(en-US / 未知 / 空)→ [github, gitee]。
func PreferredIDs(locale string) []string {
	if IsChinese(locale) {
		return []string{SourceGitee, SourceGitHub}
	}
	return []string{SourceGitHub, SourceGitee}
}

// Order 返回 sources 在给定语言下的排序结果(靠前的先用)。
//
// 偏好 ID 命中的源排在前面;没被偏好命中的源(例如只配了 legacy 源,
// 或者 Gitee 源被 env 删掉)按原顺序排在后面,保证不会因为偏好规则丢源。
func Order(sources []Source, locale string) []Source {
	if len(sources) == 0 {
		return nil
	}
	preferred := PreferredIDs(locale)
	byID := make(map[string]Source, len(sources))
	for _, s := range sources {
		byID[s.ID] = s
	}
	var out []Source
	used := make(map[string]bool, len(sources))
	for _, id := range preferred {
		if s, ok := byID[id]; ok && !used[s.ID] {
			used[s.ID] = true
			out = append(out, s)
		}
	}
	for _, s := range sources {
		if !used[s.ID] {
			used[s.ID] = true
			out = append(out, s)
		}
	}
	return out
}

// ManifestURLs 返回按语言优先级排序的 manifest url 列表(数组顺序 = 降级顺序)。
func ManifestURLs(sources []Source, locale string) []string {
	ordered := Order(sources, locale)
	out := make([]string, 0, len(ordered))
	for _, s := range ordered {
		if s.ManifestURL != "" {
			out = append(out, s.ManifestURL)
		}
	}
	return out
}

// SourceIDOf 反查某个 manifest url 属于哪个源(用于把实际命中的源回传给前端展示)。
// 匹配不到返 ""。
func SourceIDOf(sources []Source, manifestURL string) string {
	for _, s := range sources {
		if s.ManifestURL == manifestURL {
			return s.ID
		}
	}
	return ""
}
