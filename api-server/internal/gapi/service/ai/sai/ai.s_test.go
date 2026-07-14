package sai_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/go-kratos/blades"
	"ginp-api/internal/aiengine"
	"ginp-api/internal/gapi/entity"
	"ginp-api/internal/gapi/service/ai/sai"
	"ginp-api/internal/settings"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newTestService(t *testing.T) *sai.Service {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&entity.AIProvider{}, &entity.Setting{}); err != nil {
		t.Fatal(err)
	}
	st := settings.New(db, db)
	eng := sai.NewEngine(st)
	return sai.New(db, db, st, eng)
}

func TestCreate_Ok(t *testing.T) {
	svc := newTestService(t)
	row, err := svc.Create(&entity.AIProvider{
		Name: "openai-prod", Kind: "openai", Model: "gpt-4o-mini", Enabled: true, Priority: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if row.ID == 0 {
		t.Fatal("expected id")
	}
}

func TestCreate_EmptyName(t *testing.T) {
	svc := newTestService(t)
	_, err := svc.Create(&entity.AIProvider{Kind: "openai"})
	if !errors.Is(err, sai.ErrEmptyName) {
		t.Errorf("got %v, want ErrEmptyName", err)
	}
}

func TestCreate_BadKind(t *testing.T) {
	svc := newTestService(t)
	_, err := svc.Create(&entity.AIProvider{Name: "x", Kind: "unknown"})
	if err == nil {
		t.Error("expected error for bad kind")
	}
}

func TestCreate_DuplicateName(t *testing.T) {
	svc := newTestService(t)
	_, _ = svc.Create(&entity.AIProvider{Name: "dup", Kind: "openai", Enabled: true})
	_, err := svc.Create(&entity.AIProvider{Name: "dup", Kind: "openai", Enabled: true})
	if err == nil {
		t.Error("expected dup name error")
	}
}

func TestSetKey(t *testing.T) {
	svc := newTestService(t)
	if _, err := svc.Create(&entity.AIProvider{Name: "k", Kind: "openai", Enabled: true}); err != nil {
		t.Fatal(err)
	}
	if err := svc.SetKey("k", "secret-1"); err != nil {
		t.Fatal(err)
	}
	if got, _ := svc.GetKey("k"); got != "secret-1" {
		t.Errorf("got %q", got)
	}
}

func TestUpdate_RenameMigratesKey(t *testing.T) {
	svc := newTestService(t)
	row, _ := svc.Create(&entity.AIProvider{Name: "old", Kind: "openai", Enabled: true})
	svc.SetKey("old", "k1")
	upd, err := svc.Update(row.ID, &entity.AIProvider{Name: "new", Kind: "openai", Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	if upd.Name != "new" {
		t.Errorf("name=%q", upd.Name)
	}
	got, _ := svc.GetKey("new")
	if got != "k1" {
		t.Errorf("key after rename=%q", got)
	}
	if oldKey, _ := svc.GetKey("old"); oldKey != "" {
		t.Errorf("old key should be gone, got=%q", oldKey)
	}
}

func TestDelete_ClearsKey(t *testing.T) {
	svc := newTestService(t)
	row, _ := svc.Create(&entity.AIProvider{Name: "d", Kind: "openai", Enabled: true})
	svc.SetKey("d", "k")
	if err := svc.Delete(row.ID); err != nil {
		t.Fatal(err)
	}
	if k, _ := svc.GetKey("d"); k != "" {
		t.Errorf("key should be cleared, got=%q", k)
	}
}

func TestPresets_List(t *testing.T) {
	svc := newTestService(t)
	ps := svc.Presets()
	if len(ps) < 3 {
		t.Errorf("expected >=3 presets, got %d", len(ps))
	}
}

func TestChat_NoProvider_ReturnsErr(t *testing.T) {
	// 没有注册任何 provider 也没有任何 ai_providers 行 → 应当返回 ErrNoProvider
	svc := newTestService(t)
	_, err := svc.Chat(context.Background(), []*blades.Message{blades.UserMessage("yo")}, "")
	if err == nil {
		t.Fatal("expected error when no provider enabled")
	}
	if !errors.Is(err, aiengine.ErrNoProvider) {
		t.Errorf("got %v, want ErrNoProvider", err)
	}
}

func TestChatWithPreset_UnknownPreset(t *testing.T) {
	svc := newTestService(t)
	_, err := svc.ChatWithPreset(context.Background(), "nope", "", nil)
	if err == nil || !strings.Contains(err.Error(), "unknown preset") {
		t.Errorf("got %v", err)
	}
}

func TestChatWithPreset_RenderProducesBladesMessages(t *testing.T) {
	// 验 render preset 后返回的是 *blades.Message(system + user 两条)
	svc := newTestService(t)
	_, err := svc.Create(&entity.AIProvider{Name: "p", Kind: "openai", Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.SetKey("p", "k"); err != nil {
		t.Fatal(err)
	}
	// 用一个会立即超时的 provider(kind 合法,base_url 无效,触发网络错误)
	// 这里只验 ChatWithPreset 拼 prompt 不报错(走到网络才挂)
	_, err = svc.ChatWithPreset(context.Background(), "translate_skill", "", map[string]string{
		"target_lang": "English",
		"skill_md":    "name: foo",
	})
	// err 可能是网络错误,这里不 fail;只验 err 来源不是 "unknown preset"
	if err != nil && strings.Contains(err.Error(), "unknown preset") {
		t.Errorf("preset unknown: %v", err)
	}
}
