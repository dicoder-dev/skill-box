package aiengine_test

import (
	"context"
	"errors"
	"testing"

	"github.com/go-kratos/blades"
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
	if msgs[0].Role != blades.RoleSystem {
		t.Errorf("role[0]=%s", msgs[0].Role)
	}
	if msgs[0].Text() != "You are X." {
		t.Errorf("system=%q", msgs[0].Text())
	}
	if msgs[1].Text() != "Hello brody, your skill is review-pr" {
		t.Errorf("user=%q", msgs[1].Text())
	}
}

func TestRenderPreset_MissingVarLeftAsIs(t *testing.T) {
	p := aiengine.Preset{UserTemplate: "Hello {name}"}
	msgs := aiengine.RenderPreset(p, map[string]string{})
	if msgs[1].Text() != "Hello {name}" {
		t.Errorf("user=%q", msgs[1].Text())
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

// --- engine ---

type fakeSecret struct{ keys map[string]string }

func (f *fakeSecret) Resolve(name string) (string, error) {
	return f.keys[name], nil
}

func TestEngine_SelectByName(t *testing.T) {
	eng := aiengine.NewEngine(&fakeSecret{keys: map[string]string{}})
	rows := []*entity.AIProvider{
		{Name: "a", Kind: "openai", Priority: 1, Enabled: true},
		{Name: "b", Kind: "anthropic", Priority: 2, Enabled: true},
	}
	got, err := eng.Select(rows, "b")
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "b" {
		t.Errorf("got %s", got.Name)
	}
}

func TestEngine_SelectByPriority(t *testing.T) {
	eng := aiengine.NewEngine(&fakeSecret{keys: map[string]string{}})
	rows := []*entity.AIProvider{
		{Name: "a", Kind: "openai", Priority: 5, Enabled: true},
		{Name: "b", Kind: "anthropic", Priority: 1, Enabled: true},
		{Name: "c", Kind: "openai_compat", Priority: 3, Enabled: true},
	}
	got, _ := eng.Select(rows, "")
	if got.Name != "b" {
		t.Errorf("priority order wrong: got %s", got.Name)
	}
}

func TestEngine_Select_DisabledSkipped(t *testing.T) {
	eng := aiengine.NewEngine(&fakeSecret{keys: map[string]string{}})
	rows := []*entity.AIProvider{
		{Name: "a", Kind: "openai", Priority: 1, Enabled: false},
		{Name: "b", Kind: "openai", Priority: 2, Enabled: true},
	}
	got, _ := eng.Select(rows, "")
	if got.Name != "b" {
		t.Errorf("expected b, got %s", got.Name)
	}
}

func TestEngine_Select_NoneAvailable(t *testing.T) {
	eng := aiengine.NewEngine(&fakeSecret{keys: map[string]string{}})
	rows := []*entity.AIProvider{{Name: "a", Kind: "openai", Enabled: false}}
	_, err := eng.Select(rows, "")
	if err != aiengine.ErrNoProvider {
		t.Errorf("got %v, want ErrNoProvider", err)
	}
}

func TestEngine_Build_UnknownKind(t *testing.T) {
	eng := aiengine.NewEngine(&fakeSecret{keys: map[string]string{}})
	_, _, err := eng.Build(&entity.AIProvider{Name: "x", Kind: "fake"})
	if !errors.Is(err, aiengine.ErrUnknownKind) {
		t.Errorf("got %v, want ErrUnknownKind", err)
	}
}

func TestEngine_Build_OpenAI(t *testing.T) {
	eng := aiengine.NewEngine(&fakeSecret{keys: map[string]string{"x": "k1"}})
	mdl, key, err := eng.Build(&entity.AIProvider{Name: "x", Kind: "openai"})
	if err != nil {
		t.Fatal(err)
	}
	if key != "k1" {
		t.Errorf("key=%q", key)
	}
	if mdl == nil {
		t.Fatal("model is nil")
	}
	if mdl.Name() != "gpt-4o-mini" {
		t.Errorf("default model not applied: %s", mdl.Name())
	}
}

func TestEngine_Build_OpenAICom(t *testing.T) {
	eng := aiengine.NewEngine(&fakeSecret{keys: map[string]string{}})
	mdl, _, err := eng.Build(&entity.AIProvider{Name: "x", Kind: "openai_compat", Model: "deepseek-chat"})
	if err != nil {
		t.Fatal(err)
	}
	if mdl == nil {
		t.Fatal("model is nil")
	}
	if mdl.Name() != "deepseek-chat" {
		t.Errorf("name=%s", mdl.Name())
	}
}

func TestEngine_Build_Anthropic(t *testing.T) {
	eng := aiengine.NewEngine(&fakeSecret{keys: map[string]string{}})
	mdl, _, err := eng.Build(&entity.AIProvider{Name: "x", Kind: "anthropic"})
	if err != nil {
		t.Fatal(err)
	}
	if mdl == nil {
		t.Fatal("model is nil")
	}
	if mdl.Name() != "claude-3-5-sonnet-20241022" {
		t.Errorf("default anthropic model not applied: %s", mdl.Name())
	}
}

// --- streaming semantics(集成到 blades 流上) ---

// recordingModel 记录 chat 入参,按预设回放 event 序列。
type recordingModel struct {
	name   string
	events []*blades.ModelResponse
	gotReq *blades.ModelRequest
}

func (m *recordingModel) Name() string { return m.name }

func (m *recordingModel) Generate(ctx context.Context, req *blades.ModelRequest) (*blades.ModelResponse, error) {
	m.gotReq = req
	return m.events[len(m.events)-1], nil
}

func (m *recordingModel) NewStreaming(ctx context.Context, req *blades.ModelRequest) blades.Generator[*blades.ModelResponse, error] {
	m.gotReq = req
	return func(yield func(*blades.ModelResponse, error) bool) {
		for _, ev := range m.events {
			if !yield(ev, nil) {
				return
			}
		}
	}
}

func TestEngine_StreamFlow(t *testing.T) {
	rm := &recordingModel{
		name: "gpt-4o-mini",
		events: []*blades.ModelResponse{
			{Message: blades.NewAssistantMessage(blades.StatusIncomplete)},
			{Message: blades.NewAssistantMessage(blades.StatusIncomplete)},
			{Message: blades.NewAssistantMessage(blades.StatusCompleted)},
		},
	}
	// 给第一个/第二个 message 加文本
	rm.events[0].Message.Parts = []blades.Part{blades.TextPart{Text: "hi"}}
	rm.events[1].Message.Parts = []blades.Part{blades.TextPart{Text: " there"}}
	rm.events[2].Message.Parts = []blades.Part{blades.TextPart{Text: "."}}

	eng := aiengine.NewEngine(&fakeSecret{keys: map[string]string{"x": "k"}})
	eng.Register("openai", func(cfg aiengine.Config, apiKey string) (blades.ModelProvider, error) {
		return rm, nil
	})

	mdl, key, err := eng.Build(&entity.AIProvider{Name: "x", Kind: "openai"})
	if err != nil {
		t.Fatal(err)
	}
	if key != "k" {
		t.Errorf("key=%q", key)
	}
	req := &blades.ModelRequest{Messages: []*blades.Message{blades.UserMessage("yo")}}
	stream := mdl.NewStreaming(context.Background(), req)
	var got string
	for resp, err := range stream {
		if err != nil {
			t.Fatal(err)
		}
		if resp == nil || resp.Message == nil {
			continue
		}
		got += resp.Message.Text()
	}
	if got != "hi there." {
		t.Errorf("text=%q", got)
	}
	if rm.gotReq == nil || len(rm.gotReq.Messages) != 1 {
		t.Errorf("model saw wrong req: %+v", rm.gotReq)
	}
}
