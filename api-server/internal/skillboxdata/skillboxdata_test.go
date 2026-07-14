package skillboxdata

import (
	"encoding/json"
	"errors"
	"fmt"
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

// =====================================================================
// v2 单 conv 文件测试(2026-07-14 增)
// =====================================================================

// TestWriteReadConv_RoundTrip 写一条完整字段,读回字段一致。
func TestWriteReadConv_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	if err := Ensure(dir); err != nil {
		t.Fatalf("Ensure: %v", err)
	}
	item := HistoryItem{
		ID:       "conv_2026_test",
		Title:    "测试",
		Ts:       1700000000,
		Provider: "openai",
		Model:    "gpt-4o",
		Messages: json.RawMessage(`[{"id":"u1","role":"user","content":"hi"},{"id":"a1","role":"assistant","content":"hello","reason":"因为 ..."}]`),
	}
	if err := WriteConv(dir, item); err != nil {
		t.Fatalf("WriteConv: %v", err)
	}
	got, err := ReadConv(dir, "conv_2026_test")
	if err != nil {
		t.Fatalf("ReadConv: %v", err)
	}
	if got == nil {
		t.Fatal("ReadConv 返 nil")
	}
	if got.ID != item.ID || got.Title != item.Title || got.Ts != item.Ts {
		t.Errorf("field mismatch: %+v vs %+v", got, item)
	}
	// preview 自动从首条 assistant content 算
	if !strings.Contains(got.Preview, "hello") {
		t.Errorf("preview = %q, want contains 'hello'", got.Preview)
	}
}

// TestWriteConv_Upsert 同 conv_id 两次写,后者覆盖。
func TestWriteConv_Upsert(t *testing.T) {
	dir := t.TempDir()
	_ = Ensure(dir)
	a := HistoryItem{ID: "x", Title: "A", Ts: 100, Messages: json.RawMessage(`[{"role":"assistant","content":"first"}]`)}
	b := HistoryItem{ID: "x", Title: "B", Ts: 200, Messages: json.RawMessage(`[{"role":"assistant","content":"second"}]`)}
	if err := WriteConv(dir, a); err != nil {
		t.Fatal(err)
	}
	if err := WriteConv(dir, b); err != nil {
		t.Fatal(err)
	}
	got, _ := ReadConv(dir, "x")
	if got.Title != "B" {
		t.Errorf("upsert 失败: title = %q, want B", got.Title)
	}
	// 只剩一个文件
	ents, _ := os.ReadDir(filepath.Join(Dir(dir), HistoryDir))
	if len(ents) != 1 {
		t.Errorf("期望 1 个文件,实际 %d", len(ents))
	}
}

// TestWriteConv_TooLarge 超 MaxConvSize 返 ErrConvTooLarge + 盘外无残留。
func TestWriteConv_TooLarge(t *testing.T) {
	dir := t.TempDir()
	_ = Ensure(dir)
	// 单条 message 内容填到 ~2.5MB
	big := strings.Repeat("X", 2_500_000)
	item := HistoryItem{
		ID:       "too_big",
		Ts:       1,
		Messages: json.RawMessage(fmt.Sprintf(`[{"role":"user","content":%q}]`, big)),
	}
	err := WriteConv(dir, item)
	if !errors.Is(err, ErrConvTooLarge) {
		t.Errorf("err = %v, want ErrConvTooLarge", err)
	}
	// 盘外无文件
	if _, err := os.Stat(filepath.Join(Dir(dir), HistoryDir, "too_big.json")); err == nil {
		t.Errorf("不应创建文件,但还是创建了")
	}
}

// TestListConvs 列表:0/1/N / 坏 json 跳过 / ts desc / size>0。
func TestListConvs(t *testing.T) {
	dir := t.TempDir()

	// 目录不存在 → 返空
	got, err := ListConvs(dir)
	if err != nil {
		t.Fatalf("ListConvs 空目录: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("空目录列表非空: %d", len(got))
	}

	_ = Ensure(dir)
	// 写 3 条:ts 100/200/300
	for i, ts := range []int64{100, 300, 200} {
		_ = WriteConv(dir, HistoryItem{
			ID: fmt.Sprintf("c%d", i), Ts: ts,
			Messages: json.RawMessage(`[{"role":"assistant","content":"x"}]`),
		})
	}
	// 加一个坏 json
	badsDir := filepath.Join(Dir(dir), HistoryDir)
	if err := os.WriteFile(filepath.Join(badsDir, "garbage.json"), []byte("not json{"), 0o644); err != nil {
		t.Fatal(err)
	}
	// 一个非 json 后缀
	if err := os.WriteFile(filepath.Join(badsDir, "README"), []byte("h"), 0o644); err != nil {
		t.Fatal(err)
	}

	list, err := ListConvs(dir)
	if err != nil {
		t.Fatalf("ListConvs: %v", err)
	}
	if len(list) != 3 {
		t.Errorf("列表条数 = %d, want 3(坏文件被跳过)", len(list))
	}
	// ts desc:300 / 200 / 100
	for i := 0; i < len(list)-1; i++ {
		if list[i].Ts < list[i+1].Ts {
			t.Errorf("ts 不降序: %v", []int64{list[0].Ts, list[1].Ts, list[2].Ts})
		}
	}
	// size > 0
	for _, m := range list {
		if m.Size <= 0 {
			t.Errorf("size = %d, want > 0", m.Size)
		}
	}
}

// TestDeleteConv 幂等。
func TestDeleteConv(t *testing.T) {
	dir := t.TempDir()
	_ = Ensure(dir)
	_ = WriteConv(dir, HistoryItem{ID: "del_me", Ts: 1, Messages: json.RawMessage(`[{"role":"user","content":"x"}]`)})
	if err := DeleteConv(dir, "del_me"); err != nil {
		t.Fatalf("DeleteConv: %v", err)
	}
	if err := DeleteConv(dir, "del_me"); err != nil {
		// 不存在 → nil,幂等
		t.Errorf("DeleteConv (不存在): %v", err)
	}
	if err := DeleteConv(dir, "../etc"); err == nil {
		t.Errorf("非法 conv_id 应返 error")
	}
}

// TestSanitizeConvID 各种非法输入。
func TestSanitizeConvID(t *testing.T) {
	cases := []struct {
		in    string
		valid bool
	}{
		{"", false},
		{"valid_abc-123", true},
		{"../etc", false},
		{"a/b", false},
		{".json", false},
		{".", false},
		{"with space", false},
		{"with/slash", false},
		{"with.dot.x", false}, // 中间有点也算非法(只有开头 . 才拒?这里我们严格 —— 看实现)
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			_, err := sanitizeConvID(c.in)
			if c.valid && err != nil {
				t.Errorf("sanitizeConvID(%q) = %v, want valid", c.in, err)
			}
			if !c.valid && err == nil {
				t.Errorf("sanitizeConvID(%q) 应返 error", c.in)
			}
		})
	}
}
