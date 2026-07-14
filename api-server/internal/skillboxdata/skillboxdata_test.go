package skillboxdata

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestEnsure 创建 + 再调一次幂等(readme 不被覆盖)(2026-07-14 增)。
func TestEnsure(t *testing.T) {
	dir := t.TempDir()
	if err := Ensure(dir); err != nil {
		t.Fatalf("first Ensure: %v", err)
	}
	// 二次 Ensure 不应报错,且 readme 还在
	if err := Ensure(dir); err != nil {
		t.Fatalf("second Ensure: %v", err)
	}
	rm := filepath.Join(Dir(dir), ReadmeFile)
	if _, err := os.Stat(rm); err != nil {
		t.Fatalf("readme missing: %v", err)
	}
	// readme 不被覆盖:写入一个噪音,再 Ensure,内容应不变
	noise := []byte("# custom content\n")
	if err := os.WriteFile(rm, noise, 0o644); err != nil {
		t.Fatalf("write noise: %v", err)
	}
	if err := Ensure(dir); err != nil {
		t.Fatalf("Ensure after noise: %v", err)
	}
	got, _ := os.ReadFile(rm)
	if string(got) != string(noise) {
		t.Errorf("readme was overwritten by Ensure")
	}
}

// TestReadHistory_NotFound 不存在返空 History{Version: 1},不算 error。
func TestReadHistory_NotFound(t *testing.T) {
	dir := t.TempDir()
	h, err := ReadHistory(dir)
	if err != nil {
		t.Fatalf("ReadHistory: %v", err)
	}
	if h.Version != HistoryVersion {
		t.Errorf("Version = %d, want %d", h.Version, HistoryVersion)
	}
	if len(h.Items) != 0 {
		t.Errorf("Items len = %d, want 0", len(h.Items))
	}
}

// TestWriteHistory_Basic 写入几条小记录,读回字段一致。
func TestWriteHistory_Basic(t *testing.T) {
	dir := t.TempDir()
	items := []HistoryItem{
		{ID: "a", Title: "tA", Ts: 1000, Messages: json.RawMessage(`[{"role":"user","content":"hi"},{"role":"assistant","content":"hello world"}]`)},
		{ID: "b", Title: "tB", Ts: 2000, Provider: "openai", Model: "gpt-4o", Messages: json.RawMessage(`[{"role":"user","content":"foo"}]`)},
	}
	if err := WriteHistory(dir, items); err != nil {
		t.Fatalf("WriteHistory: %v", err)
	}
	h, err := ReadHistory(dir)
	if err != nil {
		t.Fatalf("ReadHistory: %v", err)
	}
	if len(h.Items) != 2 {
		t.Fatalf("Items len = %d, want 2", len(h.Items))
	}
	// preview 自动算
	if !strings.Contains(h.Items[0].Preview, "hello world") {
		t.Errorf("preview = %q, want contains 'hello world'", h.Items[0].Preview)
	}
}

// TestWriteHistory_Truncate 超 5MB 时按 ts 升序 FIFO 截断。
func TestWriteHistory_Truncate(t *testing.T) {
	dir := t.TempDir()
	// 构造 30 条大 messages(每条约 300KB),总 > 5MB
	mkBig := func(tag string) json.RawMessage {
		s := strings.Repeat("X", 300*1024) + tag
		return json.RawMessage(`[{"role":"assistant","content":"` + s + `"}]`)
	}
	var items []HistoryItem
	for i := 0; i < 30; i++ {
		items = append(items, HistoryItem{
			ID:    string(rune('a' + i%26)),
			Title: "x",
			Ts:    int64(1000 + i),
			Messages: mkBig(string(rune('a' + i%26))),
		})
	}
	if err := WriteHistory(dir, items); err != nil {
		t.Fatalf("WriteHistory: %v", err)
	}
	// 文件字节数应 <= 5MB
	path := filepath.Join(Dir(dir), HistoryFile)
	st, _ := os.Stat(path)
	if st.Size() > MaxHistorySize {
		t.Errorf("file size = %d, want <= %d", st.Size(), MaxHistorySize)
	}
	// 截断后至少剩 1 条
	h, _ := ReadHistory(dir)
	if len(h.Items) == 0 {
		t.Errorf("after truncate, no items left (want at least 1)")
	}
	// 剩下的应该是 ts 较大的(我们按 ts 升序删,留下的更"新")
	lastTs := int64(0)
	for _, it := range h.Items {
		if it.Ts < lastTs {
			t.Errorf("items not sorted by ts asc in surviving set")
		}
		lastTs = it.Ts
	}
}

// TestPreviewFromMessages preview 算法行为。
func TestPreviewFromMessages(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want string
	}{
		{"empty", "", ""},
		{"no assistant", `[{"role":"user","content":"hi"}]`, ""},
		{"first assistant", `[{"role":"user","content":"u"},{"role":"assistant","content":"hello"}]`, "hello"},
		{"trim whitespace", `[{"role":"assistant","content":"  hi there  "}]`, "hi there"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := previewFromMessages(json.RawMessage(c.raw))
			if got != c.want {
				t.Errorf("got %q, want %q", got, c.want)
			}
		})
	}
}

// TestDir 路径拼接正确。
func TestDir(t *testing.T) {
	got := Dir("/a/b")
	want := filepath.Join("/a/b", DirName)
	if got != want {
		t.Errorf("Dir() = %q, want %q", got, want)
	}
}
