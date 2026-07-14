package ssupdater

import "ginp-api/internal/gapi/controller/skillbox/cdesktop/hooks"

// DefaultManifestURLs 在 desktop / web 都没显式注入 UpdaterManifestURLs 时兜底。
//
// 这里返回 build/updater/manifest.example.json 的 sample 路径(由 release 阶段覆盖),
// 真正生产环境会改成 GitHub release + Gitea mirror 的 raw 链接。
//
// MVP 阶段:开发期返 example 的 file:// 路径,生产配置由 desktop 包 SetDesktopHooks 时覆盖。
// 不返空数组,保证 controller 阶段不会 503。
func DefaultManifestURLs(h hooks.BootstrapHooks) []string {
	if h.UpdaterManifestURLs != nil {
		if urls := h.UpdaterManifestURLs(); len(urls) > 0 {
			return urls
		}
	}
	// 兜底链:GitHub raw + Gitea raw + 本地 example(只对 dev 有意义)
	return []string{
		"https://raw.githubusercontent.com/dicoder/skill-box/main/build/updater/manifest.example.json",
		"https://gitea.example.com/dicoder/skill-box/raw/branch/main/build/updater/manifest.example.json",
	}
}
