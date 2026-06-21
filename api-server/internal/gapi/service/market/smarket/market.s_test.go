package smarket_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"ginp-api/internal/gapi/entity"
	"ginp-api/internal/gapi/service/market/smarket"
	"ginp-api/internal/gapi/service/skill/sskill"
	"ginp-api/internal/skillmarket"
	mmarketskill "ginp-api/internal/gapi/model/skillbox/mmarketskill"
	"ginp-api/internal/skillstore"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// testEnv in-memory db + temp store + smarket service + raw db(供测试种数据)。
type testEnv struct {
	svc *smarket.Service
	db  *gorm.DB
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&entity.MarketSource{}, &entity.MarketSkill{}, &entity.Skill{}, &entity.SkillFile{}); err != nil {
		t.Fatal(err)
	}
	store, err := skillstore.NewAt(filepath.Join(t.TempDir(), "store"))
	if err != nil {
		t.Fatal(err)
	}
	ssvc := sskill.New(db, db, store)
	factory := func() (*sskill.Service, error) { return ssvc, nil }
	return &testEnv{
		svc: smarket.New(db, db, factory),
		db:  db,
	}
}

// seedMarketSkill 直接用 db 插一行(模拟"已经 refresh 过")。
func (e *testEnv) seedMarketSkill(t *testing.T, sourceID uint, sourceName, remoteID, name, version string) *entity.MarketSkill {
	t.Helper()
	row := &entity.MarketSkill{
		SourceID:   sourceID,
		SourceName: sourceName,
		RemoteID:   remoteID,
		Name:       name,
		Version:    version,
		FetchedAt:  time.Now(),
	}
	if _, err := mmarketskill.NewModel(e.db, e.db).Create(row); err != nil {
		t.Fatal(err)
	}
	return row
}

func TestEnsureDefaultSources_Idempotent(t *testing.T) {
	env := newTestEnv(t)
	if err := env.svc.EnsureDefaultSources(); err != nil {
		t.Fatal(err)
	}
	if err := env.svc.EnsureDefaultSources(); err != nil {
		t.Fatal(err)
	}
	res, err := env.svc.ListSources()
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Items) != 2 {
		t.Fatalf("expected 2 sources, got %d", len(res.Items))
	}
}

func TestListSources_IncludesDefault(t *testing.T) {
	env := newTestEnv(t)
	_ = env.svc.EnsureDefaultSources()
	res, err := env.svc.ListSources()
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]bool{}
	for _, s := range res.Items {
		names[s.Name] = true
	}
	for _, want := range []string{"skillhub", "skills.sh"} {
		if !names[want] {
			t.Errorf("missing source %q", want)
		}
	}
}

func TestListSkills_Empty(t *testing.T) {
	env := newTestEnv(t)
	_ = env.svc.EnsureDefaultSources()
	res, err := env.svc.ListSkills(smarket.ListSkillsQuery{Page: 1, Size: 20})
	if err != nil {
		t.Fatal(err)
	}
	if res.Total != 0 || len(res.Items) != 0 {
		t.Errorf("expected empty, got %+v", res)
	}
}

func TestInstall_BadInput(t *testing.T) {
	env := newTestEnv(t)
	cases := []smarket.InstallInput{
		{},
		{SourceID: 1},
		{SourceID: 1, RemoteID: "x", Scope: "weird"},
		{SourceID: 1, RemoteID: "x", Scope: "project"},
	}
	for i, in := range cases {
		if _, err := env.svc.Install(context.Background(), &in); err == nil {
			t.Errorf("case %d: expected error, got nil", i)
		}
	}
}

func TestInstall_SourceNotFound(t *testing.T) {
	env := newTestEnv(t)
	_, err := env.svc.Install(context.Background(), &smarket.InstallInput{
		SourceID: 999, RemoteID: "x", Scope: "global",
	})
	if err == nil {
		t.Fatal("expected error for unknown source")
	}
}

func TestInstall_SkillNotFound(t *testing.T) {
	env := newTestEnv(t)
	_ = env.svc.EnsureDefaultSources()
	res, _ := env.svc.ListSources()
	if len(res.Items) == 0 {
		t.Fatal("setup failed")
	}
	src := res.Items[0]
	_, err := env.svc.Install(context.Background(), &smarket.InstallInput{
		SourceID: src.ID, RemoteID: "no-such-remote-id", Scope: "global",
	})
	if err == nil {
		t.Fatal("expected error for unknown remote id")
	}
}

// fakeRT 用全局 map 模拟响应;Install 测试需要替换 skillhub 默认 transport,
// 但 skillmarket 包里用的是 default httpClient(里面 hardcode *http.Client)。
// 这里走"修改 source.config_json.base_url" + "通过 http.DefaultTransport 拦截"的方案不行
// (http.DefaultTransport 没法替换 RoundTripper)。
// 替代方案:直接调用 orchestrator.DownloadFromSource 路径需要的 adapter → 但 default
// 走的是 skillhub.New()(即默认 httpClient)。所以这条 E2E 路径在沙盒里不能完整跑。
// 退一步:只验证 service.Install 的 happy path 用 fallback canonical(走 knownFallback)。
func TestInstall_GlobalOk_UsingFallback(t *testing.T) {
	env := newTestEnv(t)
	if err := env.svc.EnsureDefaultSources(); err != nil {
		t.Fatal(err)
	}
	res, _ := env.svc.ListSources()
	var src *entity.MarketSource
	for _, s := range res.Items {
		if s.Type == skillmarket.SourceSkillhub {
			src = s
			break
		}
	}
	if src == nil {
		t.Fatal("skillhub source not seeded")
	}
	// 让 base_url 走一个不可达地址,让 skillhub.Download 走 knownFallback 分支
	if _, err := env.svc.UpdateSourceConfig(src.ID, `{"base_url":"https://127.0.0.1:1"}`); err != nil {
		t.Fatal(err)
	}
	// 预插 market_skill
	env.seedMarketSkill(t, src.ID, src.Name, "code-review", "Code Review", "1.0.0")

	// 沙盒里 127.0.0.1:1 会立即连接拒绝 → fetchBody 返错 → 走 knownFallback 分支
	out, err := env.svc.Install(context.Background(), &smarket.InstallInput{
		SourceID: src.ID, RemoteID: "code-review", Scope: "global",
	})
	if err != nil {
		t.Fatalf("install should succeed via fallback: %v", err)
	}
	if out == nil || out.Skill == nil {
		t.Fatalf("nil result: %+v", out)
	}
	if out.Skill.Name != "code-review" {
		t.Errorf("skill name: %q", out.Skill.Name)
	}
	if out.Skill.Source != "market" {
		t.Errorf("skill source: %q", out.Skill.Source)
	}
}

// Suppress unused import warnings.
var (
	_ = bytes.NewReader
	_ = http.MethodGet
	_ = io.NopCloser
)
