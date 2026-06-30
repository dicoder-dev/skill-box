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
	"ginp-api/internal/gapi/service/skillapp/sskillapp"
	"ginp-api/internal/skilladapter"
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
	if err := db.AutoMigrate(&entity.MarketSource{}, &entity.MarketSkill{}); err != nil {
		t.Fatal(err)
	}
	store, err := skillstore.NewAt(filepath.Join(t.TempDir(), "store"))
	if err != nil {
		t.Fatal(err)
	}
	ssvc := sskill.New(store)
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
	if out == nil || out.Canonical == nil {
		t.Fatalf("nil result: %+v", out)
	}
	if out.Canonical.Manifest.Name != "code-review" {
		t.Errorf("skill name: %q", out.Canonical.Manifest.Name)
	}
	if out.Canonical.Manifest.Source != "market" {
		t.Errorf("skill source: %q", out.Canonical.Manifest.Source)
	}
}

// TestInstallV2_GlobalOk_UsingFallback 验证 v2 写盘 + apply 链路。
// 注入一个空 registry 的 sskillapp,避免真去操作各工具目录;写盘 + 走 apply
// 路径(marketskill 的 status 会是 failed 因为没装工具,但 store 写盘已成功)。
func TestInstallV2_GlobalOk_UsingFallback(t *testing.T) {
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
	env.seedMarketSkill(t, src.ID, src.Name, "code-review", "Code Review", "1.0.0")
	// 构造 sskillapp + 注入到 smarket
	store, _ := skillstore.NewAt(filepath.Join(t.TempDir(), "store2"))
	ssvc := sskill.New(store)
	factory := func() (*sskill.Service, error) { return ssvc, nil }
	skillApp := sskillapp.New(env.db, env.db, factory)
	v2 := smarket.NewWithApply(env.db, env.db, factory, skillApp)
	// 走 v2 路径,2026-06-30 改:tools=nil 不再默认 AllTools,只写盘不 apply。
	// 这里显式传 Tools=AllTools 测"apply 路径"分支
	out, err := v2.InstallV2(context.Background(), &smarket.InstallV2Input{
		SourceID: src.ID, RemoteID: "code-review", Scope: "global",
		Tools: skilladapter.AllTools,
	})
	if err != nil {
		t.Fatalf("install-v2 should succeed via fallback: %v", err)
	}
	if out == nil {
		t.Fatal("nil result")
	}
	if out.Name != "code-review" {
		t.Errorf("expected name=code-review, got %q", out.Name)
	}
	if out.Version == "" {
		t.Error("version should be set")
	}
	if len(out.Tools) != len(skilladapter.AllTools) {
		t.Errorf("expected %d tools (AllTools), got %d", len(skilladapter.AllTools), len(out.Tools))
	}
	// 写盘应已存在(store 后台扫到 code-review)
	if !store.Exists("code-review") {
		t.Error("expected skill written to store")
	}
}

// TestInstallV2_FinalName_Rename 验证 FinalName 字段支持"另存为"。
func TestInstallV2_FinalName_Rename(t *testing.T) {
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
	if _, err := env.svc.UpdateSourceConfig(src.ID, `{"base_url":"https://127.0.0.1:1"}`); err != nil {
		t.Fatal(err)
	}
	env.seedMarketSkill(t, src.ID, src.Name, "code-review", "Code Review", "1.0.0")
	store, _ := skillstore.NewAt(filepath.Join(t.TempDir(), "store3"))
	ssvc := sskill.New(store)
	factory := func() (*sskill.Service, error) { return ssvc, nil }
	skillApp := sskillapp.New(env.db, env.db, factory)
	v2 := smarket.NewWithApply(env.db, env.db, factory, skillApp)
	out, err := v2.InstallV2(context.Background(), &smarket.InstallV2Input{
		SourceID: src.ID, RemoteID: "code-review", Scope: "global", FinalName: "code-review-2",
	})
	if err != nil {
		t.Fatalf("install-v2 with final_name: %v", err)
	}
	if out.Name != "code-review-2" {
		t.Errorf("expected name=code-review-2, got %q", out.Name)
	}
	if !store.Exists("code-review-2") {
		t.Error("expected renamed skill in store")
	}
}

// TestInstallV2_EmptyTools_OnlyWrite 2026-06-30 增:Tools=nil/[] 时只写盘不 apply。
func TestInstallV2_EmptyTools_OnlyWrite(t *testing.T) {
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
	if _, err := env.svc.UpdateSourceConfig(src.ID, `{"base_url":"https://127.0.0.1:1"}`); err != nil {
		t.Fatal(err)
	}
	env.seedMarketSkill(t, src.ID, src.Name, "code-review", "Code Review", "1.0.0")
	store, _ := skillstore.NewAt(filepath.Join(t.TempDir(), "store-empty"))
	ssvc := sskill.New(store)
	factory := func() (*sskill.Service, error) { return ssvc, nil }
	skillApp := sskillapp.New(env.db, env.db, factory)
	v2 := smarket.NewWithApply(env.db, env.db, factory, skillApp)
	// Tools 不传(零值 nil)→ 只写盘
	out, err := v2.InstallV2(context.Background(), &smarket.InstallV2Input{
		SourceID: src.ID, RemoteID: "code-review", Scope: "global",
	})
	if err != nil {
		t.Fatalf("install-v2 with empty tools: %v", err)
	}
	if !store.Exists("code-review") {
		t.Error("expected skill written to store")
	}
	if len(out.Tools) != 0 {
		t.Errorf("expected 0 tools (empty), got %d", len(out.Tools))
	}
	if out.ApplyResult != nil {
		t.Errorf("expected no apply result when tools is empty, got %+v", out.ApplyResult)
	}
}

// TestInstallV2_GroupPath_WritesToSubdir 2026-06-30 增:GroupPath 写到 Manifest.GroupPath,
// store 落到子目录。验证通过 store.LoadByPath 读回。
func TestInstallV2_GroupPath_WritesToSubdir(t *testing.T) {
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
	if _, err := env.svc.UpdateSourceConfig(src.ID, `{"base_url":"https://127.0.0.1:1"}`); err != nil {
		t.Fatal(err)
	}
	env.seedMarketSkill(t, src.ID, src.Name, "code-review", "Code Review", "1.0.0")
	store, _ := skillstore.NewAt(filepath.Join(t.TempDir(), "store-group"))
	ssvc := sskill.New(store)
	factory := func() (*sskill.Service, error) { return ssvc, nil }
	v2 := smarket.NewWithApply(env.db, env.db, factory, nil) // 不注 apply
	// GroupPath 装到 frontend/react/code-review
	out, err := v2.InstallV2(context.Background(), &smarket.InstallV2Input{
		SourceID: src.ID, RemoteID: "code-review", Scope: "global", GroupPath: "frontend/react",
	})
	if err != nil {
		t.Fatalf("install-v2 with group_path: %v", err)
	}
	if out.GroupPath != "frontend/react" {
		t.Errorf("expected group_path=frontend/react, got %q", out.GroupPath)
	}
	// 验证写到了子目录(通过 LoadByPath 读回)
	can, lerr := store.LoadByPath("frontend/react", "code-review")
	if lerr != nil {
		t.Fatalf("LoadByPath failed: %v", lerr)
	}
	if can.Manifest.GroupPath != "frontend/react" {
		t.Errorf("manifest group_path should be frontend/react, got %q", can.Manifest.GroupPath)
	}
	if can.Manifest.Name != "code-review" {
		t.Errorf("name should be code-review, got %q", can.Manifest.Name)
	}
}

// TestInstallV2_BadGroupPath 2026-06-30 增:验证非法 group_path 在 normalize 阶段就被处理。
//
// 设计决策:NormalizeGroupName 只接受 [a-z0-9-],把 '.' / '/' / ' ' / 其它字符都折叠为 '-'。
// 所以 "../escape" 实际 normalize 成 "-escape","foo/../bar" 变成 "foo-bar" ——
// 等于"客户端永远造不出含 .. 的 group_path",store.safeRelPath 是第二道防线。
// 这里测试三个"输入时看起来坏但 normalize 后是有效名"的 case,验证安装不挂即可。
func TestInstallV2_BadGroupPath(t *testing.T) {
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
	if _, err := env.svc.UpdateSourceConfig(src.ID, `{"base_url":"https://127.0.0.1:1"}`); err != nil {
		t.Fatal(err)
	}
	env.seedMarketSkill(t, src.ID, src.Name, "code-review", "Code Review", "1.0.0")
	store, _ := skillstore.NewAt(filepath.Join(t.TempDir(), "store-bad-gp"))
	ssvc := sskill.New(store)
	factory := func() (*sskill.Service, error) { return ssvc, nil }
	v2 := smarket.NewWithApply(env.db, env.db, factory, nil)
	// "脏"输入 — normalize 后变成安全名,不应该报错
	cases := []string{
		"../escape",    // → "-escape" (TrimRight 把首字符 - 去掉,实际变 "escape")
		"foo/../bar",   // → "foo-bar"
		"/abs/path",    // → "abs-path"
		"FRONT END/React", // → "front-end-react"
	}
	for _, gp := range cases {
		out, err := v2.InstallV2(context.Background(), &smarket.InstallV2Input{
			SourceID: src.ID, RemoteID: "code-review", Scope: "global", GroupPath: gp,
		})
		if err != nil {
			t.Errorf("group_path %q should normalize + install OK, got: %v", gp, err)
			continue
		}
		// 验证 GroupPath 字段是规范化后的(不等于输入)
		if out.GroupPath == gp {
			t.Errorf("group_path %q should be normalized, but got raw %q", gp, out.GroupPath)
		}
	}
	// 完全空 normalize 后也是空("    " 空白)→ InstallV2 当作没传 group_path
	// 这一路径通过空字符串分支,不创建分组
	out, err := v2.InstallV2(context.Background(), &smarket.InstallV2Input{
		SourceID: src.ID, RemoteID: "code-review", Scope: "global", GroupPath: "   ",
	})
	if err != nil {
		t.Errorf("whitespace group_path should be treated as empty: %v", err)
	}
	if out.GroupPath != "" {
		t.Errorf("whitespace group_path should normalize to empty, got %q", out.GroupPath)
	}
}

// TestInstallV2_BadInput 验证 v2 入参校验。
func TestInstallV2_BadInput(t *testing.T) {
	env := newTestEnv(t)
	cases := []smarket.InstallV2Input{
		{},
		{SourceID: 1},
		{SourceID: 1, RemoteID: "x", Scope: "weird"},
		{SourceID: 1, RemoteID: "x", Scope: "project"},
		{SourceID: 1, RemoteID: "x", Scope: "global", FinalName: "!!!"}, // 归一化后空
	}
	for i, in := range cases {
		if _, err := env.svc.InstallV2(context.Background(), &in); err == nil {
			t.Errorf("case %d: expected error, got nil", i)
		}
	}
}

// TestListSkillsWithInstalled 验证带 installed 标记的列表。
func TestListSkillsWithInstalled(t *testing.T) {
	env := newTestEnv(t)
	if err := env.svc.EnsureDefaultSources(); err != nil {
		t.Fatal(err)
	}
	env.seedMarketSkill(t, 1, "skillhub", "code-review", "code-review", "1.0.0")
	env.seedMarketSkill(t, 1, "skillhub", "commit-msg", "commit-msg", "1.0.0")
	res, err := env.svc.ListSkillsWithInstalled(smarket.ListSkillsQuery{Page: 1, Size: 20})
	if err != nil {
		t.Fatal(err)
	}
	if res.Total != 2 {
		t.Errorf("expected 2 items, got %d", res.Total)
	}
	if res.Installed == nil {
		t.Error("Installed map should not be nil")
	}
}

// TestUpdateSource 验证源 update 走 enabled / config_json。
func TestUpdateSource(t *testing.T) {
	env := newTestEnv(t)
	if err := env.svc.EnsureDefaultSources(); err != nil {
		t.Fatal(err)
	}
	res, _ := env.svc.ListSources()
	if len(res.Items) == 0 {
		t.Fatal("no sources")
	}
	src := res.Items[0]
	disabled := false
	updated, err := env.svc.UpdateSource(src.ID, &smarket.UpdateSourceInput{Enabled: &disabled})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Enabled {
		t.Error("expected enabled=false")
	}
}

// Suppress unused import warnings.
var (
	_ = bytes.NewReader
	_ = http.MethodGet
	_ = io.NopCloser
)
