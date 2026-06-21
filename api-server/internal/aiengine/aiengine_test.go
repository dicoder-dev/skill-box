package aiengine_test

import (
	"context"
	"errors"
	"testing"

	"ginp-api/internal/aiengine"
	"ginp-api/internal/gapi/entity"
)

// --- presets ---

func TestRenderPreset_TemplateSubst(t *testing.T) {
	p := aiengine.Preset{
		System:      "You are X.",
		UserTemplate: "Hello {name}, your skill is {skill_md}",
	}
	msgs := aiengine.RenderPreset(p, map[string]string{"name": "brody", "skill_md": "review-pr"})
	if len(msgs) != 2 {
		t.Fatalf("len=%d", len(msgs))
	}
	if msgs[0].Role != aiengine.RoleSystem || msgs[0].Content != "You are X." {
		t.Errorf("system=%+v", msgs[0])
	}
	if msgs[1].Content != "Hello brody, your skill is review-pr" {
		t.Errorf("user=%q", msgs[1].Content)
	}
}

func TestRenderPreset_MissingVarLeftAsIs(t *testing.T) {
	p := aiengine.Preset{UserTemplate: "Hello {name}"}
	msgs := aiengine.RenderPreset(p, map[string]string{})
	if msgs[1].Content != "Hello {name}" {
		t.Errorf("user=%q", msgs[0].Content)
	}
}

func TestAllPresets_HaveID(t *testing.T) {
	seen := map[string]bool{}
	for _, p := range aiengine.AllPresets {
		if p.ID == "" {
			t.Errorf("preset with empty id: %+v", p)
		}
		if seen[p.ID] {
			t.Errorf("duplicate preset id: %s", p.ID)
		}
		seen[p.ID] = true
		if p.System == "" || p.UserTemplate == "" {
			t.Errorf("preset %s missing system/user template", p.ID)
		}
	}
}

// --- manager ---

type fakeSecret struct{ keys map[string]string }

func (f *fakeSecret) Resolve(name string) (string, error) {
	return f.keys[name], nil
}

func TestManager_SelectByName(t *testing.T) {
	mgr := aiengine.NewManager(&fakeSecret{keys: map[string]string{}})
	rows := []*entity.AIProvider{
		{Name: "a", Kind: "openai", Priority: 1, Enabled: true},
		{Name: "b", Kind: "anthropic", Priority: 2, Enabled: true},
	}
	got, err := mgr.Select(rows, "b")
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "b" {
		t.Errorf("got %s", got.Name)
	}
}

func TestManager_SelectByPriority(t *testing.T) {
	mgr := aiengine.NewManager(&fakeSecret{keys: map[string]string{}})
	rows := []*entity.AIProvider{
		{Name: "a", Kind: "openai", Priority: 5, Enabled: true},
		{Name: "b", Kind: "anthropic", Priority: 1, Enabled: true},
		{Name: "c", Kind: "openai_compat", Priority: 3, Enabled: true},
	}
	got, _ := mgr.Select(rows, "")
	if got.Name != "b" {
		t.Errorf("priority order wrong: got %s", got.Name)
	}
}

func TestManager_Select_DisabledSkipped(t *testing.T) {
	mgr := aiengine.NewManager(&fakeSecret{keys: map[string]string{}})
	rows := []*entity.AIProvider{
		{Name: "a", Kind: "openai", Priority: 1, Enabled: false},
		{Name: "b", Kind: "openai", Priority: 2, Enabled: true},
	}
	got, _ := mgr.Select(rows, "")
	if got.Name != "b" {
		t.Errorf("expected b, got %s", got.Name)
	}
}

func TestManager_Select_NoneAvailable(t *testing.T) {
	mgr := aiengine.NewManager(&fakeSecret{keys: map[string]string{}})
	rows := []*entity.AIProvider{
		{Name: "a", Kind: "openai", Priority: 1, Enabled: false},
	}
	_, err := mgr.Select(rows, "")
	if !errors.Is(err, aiengine.ErrNoProvider) {
		t.Errorf("got %v, want ErrNoProvider", err)
	}
}

func TestManager_Build_UnknownKind(t *testing.T) {
	mgr := aiengine.NewManager(&fakeSecret{keys: map[string]string{}})
	_, _, err := mgr.Build(&entity.AIProvider{Name: "x", Kind: "fake"})
	if !errors.Is(err, aiengine.ErrUnknownKind) {
		t.Errorf("got %v, want ErrUnknownKind", err)
	}
}

func TestManager_Build_OpenAI(t *testing.T) {
	mgr := aiengine.NewManager(&fakeSecret{keys: map[string]string{"x": "k1"}})
	prov, key, err := mgr.Build(&entity.AIProvider{Name: "x", Kind: "openai"})
	if err != nil {
		t.Fatal(err)
	}
	if key != "k1" {
		t.Errorf("key=%q", key)
	}
	if prov.Kind() != "openai" {
		t.Errorf("kind=%s", prov.Kind())
	}
}

func TestManager_Build_OpenAICom(t *testing.T) {
	mgr := aiengine.NewManager(&fakeSecret{keys: map[string]string{}})
	prov, _, err := mgr.Build(&entity.AIProvider{Name: "x", Kind: "openai_compat"})
	if err != nil {
		t.Fatal(err)
	}
	if prov.Kind() != "openai_compat" {
		t.Errorf("kind=%s", prov.Kind())
	}
}

func TestManager_Build_Anthropic(t *testing.T) {
	mgr := aiengine.NewManager(&fakeSecret{keys: map[string]string{}})
	prov, _, err := mgr.Build(&entity.AIProvider{Name: "x", Kind: "anthropic"})
	if err != nil {
		t.Fatal(err)
	}
	if prov.Kind() != "anthropic" {
		t.Errorf("kind=%s", prov.Kind())
	}
}

// --- streaming semantics ---

// fakeProvider 记录 chat 入参,按预设回放 event 序列。
type fakeProvider struct {
	kind   string
	events []aiengine.StreamEvent
	gotReq aiengine.ChatRequest
	gotKey string
}

func (f *fakeProvider) Kind() string { return f.kind }

func (f *fakeProvider) Chat(ctx context.Context, req aiengine.ChatRequest, apiKey string, out chan<- aiengine.StreamEvent) error {
	defer close(out)
	f.gotReq = req
	f.gotKey = apiKey
	for _, ev := range f.events {
		select {
		case <-ctx.Done():
			return nil
		case out <- ev:
		}
	}
	return nil
}

func TestManager_StreamFlow(t *testing.T) {
	fp := &fakeProvider{
		kind: "openai",
		events: []aiengine.StreamEvent{
			{Kind: "chunk", Text: "hi"},
			{Kind: "chunk", Text: " there"},
			{Kind: "done", Usage: &aiengine.Usage{PromptTokens: 3, CompletionTokens: 5}},
		},
	}
	mgr := aiengine.NewManager(&fakeSecret{keys: map[string]string{"x": "k"}})
	// 把 fake 注入(monkey patch by kind)
	mgr.Register("openai", func(cfg aiengine.Config) aiengine.Provider { return fp })

	prov, key, err := mgr.Build(&entity.AIProvider{Name: "x", Kind: "openai"})
	if err != nil {
		t.Fatal(err)
	}
	if key != "k" {
		t.Errorf("key=%q", key)
	}
	ch := make(chan aiengine.StreamEvent, 8)
	if err := prov.Chat(context.Background(), aiengine.ChatRequest{
		Model:    "m",
		Messages: []aiengine.Message{{Role: aiengine.RoleUser, Content: "yo"}},
	}, key, ch); err != nil {
		t.Fatal(err)
	}
	var got string
	for ev := range ch {
		if ev.Kind == "chunk" {
			got += ev.Text
		}
	}
	if got != "hi there" {
		t.Errorf("text=%q", got)
	}
	if fp.gotKey != "k" {
		t.Errorf("provider saw key=%q", fp.gotKey)
	}
}

func TestManager_StreamFlow_ErrorEvent(t *testing.T) {
	fp := &fakeProvider{
		kind: "openai",
		events: []aiengine.StreamEvent{
			{Kind: "error", Err: "boom"},
		},
	}
	ch := make(chan aiengine.StreamEvent, 4)
	if err := fp.Chat(context.Background(), aiengine.ChatRequest{}, "k", ch); err != nil {
		t.Fatal(err)
	}
	ev, ok := <-ch
	if !ok || ev.Kind != "error" || ev.Err != "boom" {
		t.Errorf("got %+v ok=%v", ev, ok)
	}
}
