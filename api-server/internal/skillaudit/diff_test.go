package skillaudit_test

import (
	"testing"

	"ginp-api/internal/skillaudit"
)

func TestDiff_AddedRemoved(t *testing.T) {
	left := []skillaudit.FileSnap{{Path: "a.md", Content: "x"}}
	right := []skillaudit.FileSnap{{Path: "b.md", Content: "y"}}
	d := skillaudit.Diff(left, right)
	if len(d) != 2 {
		t.Fatalf("len = %d, want 2 (%+v)", len(d), d)
	}
	// 排序:added 在前
	if d[0].Kind != "added" || d[0].Path != "b.md" {
		t.Errorf("d[0] = %+v, want added b.md", d[0])
	}
	if d[1].Kind != "removed" || d[1].Path != "a.md" {
		t.Errorf("d[1] = %+v, want removed a.md", d[1])
	}
}

func TestDiff_Modified(t *testing.T) {
	left := []skillaudit.FileSnap{{Path: "a.md", Content: "line1\nline2\nline3"}}
	right := []skillaudit.FileSnap{{Path: "a.md", Content: "line1\nLINE2\nline3\nline4"}}
	d := skillaudit.Diff(left, right)
	if len(d) != 1 {
		t.Fatalf("len = %d, want 1", len(d))
	}
	if d[0].Kind != "modified" {
		t.Errorf("kind = %s, want modified", d[0].Kind)
	}
	if d[0].LeftHash == d[0].RightHash {
		t.Errorf("hash should differ")
	}
	// 行级 diff 至少应该有 added + removed
	kinds := map[string]int{}
	for _, ln := range d[0].Lines {
		kinds[ln.Kind]++
	}
	if kinds["removed"] == 0 || kinds["added"] == 0 {
		t.Errorf("expected added+removed in lines, got %+v", kinds)
	}
}

func TestDiff_Unchanged(t *testing.T) {
	left := []skillaudit.FileSnap{{Path: "a.md", Content: "x"}}
	right := []skillaudit.FileSnap{{Path: "a.md", Content: "x"}}
	d := skillaudit.Diff(left, right)
	if len(d) != 1 || d[0].Kind != "unchanged" {
		t.Errorf("want unchanged, got %+v", d)
	}
}

func TestDiff_EmptyLeft(t *testing.T) {
	right := []skillaudit.FileSnap{{Path: "a.md", Content: "x"}}
	d := skillaudit.Diff(nil, right)
	if len(d) != 1 || d[0].Kind != "added" {
		t.Errorf("want added, got %+v", d)
	}
}

func TestDiff_EmptyRight(t *testing.T) {
	left := []skillaudit.FileSnap{{Path: "a.md", Content: "x"}}
	d := skillaudit.Diff(left, nil)
	if len(d) != 1 || d[0].Kind != "removed" {
		t.Errorf("want removed, got %+v", d)
	}
}

func TestDiff_SortedByKindPriority(t *testing.T) {
	// 准备混合:有 unchanged / modified / added / removed
	left := []skillaudit.FileSnap{
		{Path: "keep.md", Content: "same"},
		{Path: "mod.md", Content: "old"},
		{Path: "rm.md", Content: "gone"},
	}
	right := []skillaudit.FileSnap{
		{Path: "keep.md", Content: "same"},
		{Path: "mod.md", Content: "new"},
		{Path: "add.md", Content: "fresh"},
	}
	d := skillaudit.Diff(left, right)
	// 排序:add → rm → mod → keep
	wantOrder := []string{"added", "removed", "modified", "unchanged"}
	if len(d) != 4 {
		t.Fatalf("len = %d, want 4 (%+v)", len(d), d)
	}
	for i, want := range wantOrder {
		if d[i].Kind != want {
			t.Errorf("d[%d].Kind = %s, want %s (path=%s)", i, d[i].Kind, want, d[i].Path)
		}
	}
}

func TestLinesDiff_ContextCollapse(t *testing.T) {
	// 100 行 unchanged + 1 行 removed + 100 行 unchanged → context 应被折叠
	left := ""
	right := ""
	for i := 0; i < 100; i++ {
		left += "ctx\n"
		right += "ctx\n"
	}
	left += "oldline\n"
	right += ""
	for i := 0; i < 100; i++ {
		left += "ctx2\n"
		right += "ctx2\n"
	}
	dl := skillaudit.LinesDiff(left, right)
	kinds := map[string]int{}
	for _, l := range dl {
		kinds[l.Kind]++
	}
	if kinds["removed"] != 1 {
		t.Errorf("removed count = %d, want 1", kinds["removed"])
	}
	// context 应远小于 200(已被折叠)
	if kinds["context"] > 10 {
		t.Errorf("context count = %d, want < 10 (collapsed)", kinds["context"])
	}
}

func TestLinesDiff_AddedAndRemoved(t *testing.T) {
	dl := skillaudit.LinesDiff("a\nb\nc", "a\nx\nc\nd")
	kinds := map[string]int{}
	for _, l := range dl {
		kinds[l.Kind]++
	}
	if kinds["added"] < 2 {
		t.Errorf("added count = %d, want >= 2 (x + d)", kinds["added"])
	}
	if kinds["removed"] < 1 {
		t.Errorf("removed count = %d, want >= 1 (b)", kinds["removed"])
	}
}

func TestLinesDiff_Empty(t *testing.T) {
	if dl := skillaudit.LinesDiff("", ""); len(dl) != 0 {
		t.Errorf("empty → empty, got %d", len(dl))
	}
	if dl := skillaudit.LinesDiff("", "x"); len(dl) != 1 || dl[0].Kind != "added" {
		t.Errorf("empty left + 1 line: %+v", dl)
	}
}
