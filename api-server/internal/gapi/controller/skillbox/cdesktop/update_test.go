package cdesktop

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ginp-api/pkg/ginp"

	"github.com/gin-gonic/gin"
)

// manifestServer 起一个返回指定 manifest 的测试服务器,模拟一个更新源(Gitee / GitHub)。
func manifestServer(t *testing.T, version, notes string, assetURL string) *httptest.Server {
	t.Helper()
	body := `{"channel":"stable","version":"` + version + `","released_at":"2026-10-30T00:00:00Z","min_supported":"1.0.0",` +
		`"notes":{"zh-CN":"` + notes + `","en-US":"` + notes + `"},` +
		`"assets":[{"os":"darwin","arch":"arm64","kind":"app","size":1,"sha256":"aa","urls":["` + assetURL + `"]}]}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv
}

// checkUpdate 走完整 HTTP 链路(路由注册 → 参数绑定 → controller),返回 check 响应。
func checkUpdate(t *testing.T, lang string) RespondUpdateCheck {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	ginp.RegisterRouter(r)

	url := "/api/desktop/update/check"
	if lang != "" {
		url += "?lang=" + lang
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, url, nil))
	if w.Code != http.StatusOK {
		t.Fatalf("check 返回 %d, body=%s", w.Code, w.Body.String())
	}
	var resp RespondUpdateCheck
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v, body=%s", err, w.Body.String())
	}
	return resp
}

func TestUpdateCheckPicksGiteeForChinese(t *testing.T) {
	gitee := manifestServer(t, "9.9.9", "来自 Gitee 的新版本", "https://gitee.com/dicoder/skill-box/releases/download/v9.9.9/app.zip")
	github := manifestServer(t, "9.9.9", "来自 GitHub 的新版本", "https://github.com/dicoder-dev/skill-box/releases/download/v9.9.9/app.zip")
	t.Setenv("SKILLBOX_UPDATER_SOURCES", "gitee="+gitee.URL+",github="+github.URL)

	resp := checkUpdate(t, "zh-CN")
	if resp.Source != "gitee" {
		t.Errorf("中文用户 source = %q, 期望 gitee", resp.Source)
	}
	if resp.Notes["zh-CN"] != "来自 Gitee 的新版本" {
		t.Errorf("中文用户拿到的 manifest 源不对: notes=%v", resp.Notes)
	}
	if len(resp.Assets) != 1 || resp.Assets[0].URLs[0] != "https://gitee.com/dicoder/skill-box/releases/download/v9.9.9/app.zip" {
		t.Errorf("中文用户安装包地址应指向 Gitee: %+v", resp.Assets)
	}
}

func TestUpdateCheckPicksGitHubForEnglish(t *testing.T) {
	gitee := manifestServer(t, "9.9.9", "来自 Gitee 的新版本", "https://gitee.com/dicoder/skill-box/releases/download/v9.9.9/app.zip")
	github := manifestServer(t, "9.9.9", "来自 GitHub 的新版本", "https://github.com/dicoder-dev/skill-box/releases/download/v9.9.9/app.zip")
	t.Setenv("SKILLBOX_UPDATER_SOURCES", "gitee="+gitee.URL+",github="+github.URL)

	resp := checkUpdate(t, "en-US")
	if resp.Source != "github" {
		t.Errorf("英文用户 source = %q, 期望 github", resp.Source)
	}
	if resp.Notes["en-US"] != "来自 GitHub 的新版本" {
		t.Errorf("英文用户拿到的 manifest 源不对: notes=%v", resp.Notes)
	}
	if len(resp.Assets) != 1 || resp.Assets[0].URLs[0] != "https://github.com/dicoder-dev/skill-box/releases/download/v9.9.9/app.zip" {
		t.Errorf("英文用户安装包地址应指向 GitHub: %+v", resp.Assets)
	}
}

func TestUpdateCheckFallsBackWhenPreferredSourceDown(t *testing.T) {
	// Gitee 源返回 500 → 中文用户应自动降级到 GitHub
	broken := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(broken.Close)
	github := manifestServer(t, "9.9.9", "来自 GitHub 的新版本", "https://github.com/dicoder-dev/skill-box/releases/download/v9.9.9/app.zip")
	t.Setenv("SKILLBOX_UPDATER_SOURCES", "gitee="+broken.URL+",github="+github.URL)

	resp := checkUpdate(t, "zh-CN")
	if resp.Source != "github" {
		t.Errorf("首选源不可用时应降级, source = %q, 期望 github", resp.Source)
	}
}

func TestUpdateCheckWithoutLangFallsBackToGitHub(t *testing.T) {
	gitee := manifestServer(t, "9.9.9", "来自 Gitee 的新版本", "https://gitee.com/dicoder/skill-box/releases/download/v9.9.9/app.zip")
	github := manifestServer(t, "9.9.9", "来自 GitHub 的新版本", "https://github.com/dicoder-dev/skill-box/releases/download/v9.9.9/app.zip")
	t.Setenv("SKILLBOX_UPDATER_SOURCES", "gitee="+gitee.URL+",github="+github.URL)

	// 不传 lang(老客户端)→ 按默认(英文/未知)走 GitHub
	resp := checkUpdate(t, "")
	if resp.Source != "github" {
		t.Errorf("未传语言时 source = %q, 期望 github", resp.Source)
	}
}
