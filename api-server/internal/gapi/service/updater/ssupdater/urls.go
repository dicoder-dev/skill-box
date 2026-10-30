package ssupdater

import (
	"ginp-api/internal/gapi/controller/skillbox/cdesktop/hooks"
	"ginp-api/pkg/updatersrc"
)

// DefaultManifestURLs 按语言返回 manifest 源 url 列表(数组顺序 = 降级顺序)。
//
// 取值优先级:
//  1. 桌面端通过 BootstrapHooks.UpdaterManifestURLs 注入(可按 locale 定制);
//  2. 没注入时用 updatersrc 的内置源(Gitee / GitHub),按 locale 排序:
//     中文(zh-CN)→ Gitee 优先,英文(en-US)→ GitHub 优先。
//
// 不返空数组,保证 controller 阶段不会 503。
func DefaultManifestURLs(h hooks.BootstrapHooks, locale string) []string {
	if h.UpdaterManifestURLs != nil {
		if urls := h.UpdaterManifestURLs(locale); len(urls) > 0 {
			return urls
		}
	}
	return updatersrc.ManifestURLs(updatersrc.Sources(), locale)
}
