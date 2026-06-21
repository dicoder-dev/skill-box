package skillaudit_test

import (
	"errors"
	"strings"
	"testing"

	"ginp-api/internal/skillaudit"
)

func TestValidateTag_Valid(t *testing.T) {
	cases := []string{
		"v1.0.0", "v0.1", "release-1.0", "stable", "patch_3",
		"feature/auth", "sub/dir/v1", "_pre_rollback_20260101T120000", "0.1.0",
	}
	for _, c := range cases {
		if err := skillaudit.ValidateTag(c); err != nil {
			t.Errorf("ValidateTag(%q) = %v, want nil", c, err)
		}
	}
}

func TestValidateTag_Invalid(t *testing.T) {
	cases := []struct {
		tag string
		why string
	}{
		{"", "empty"},
		{"   ", "empty after trim"},
		{".", "current dir"},
		{"..", "parent dir"},
		{"a b", "space"},
		{"a*b", "asterisk"},
		{"a\\b", "backslash"},
		{strings.Repeat("a", 65), "too long"},
	}
	for _, c := range cases {
		if err := skillaudit.ValidateTag(c.tag); err == nil {
			t.Errorf("ValidateTag(%q) = nil, want error (%s)", c.tag, c.why)
		} else if !errors.Is(err, skillaudit.ErrInvalidTag) && !errors.Is(err, skillaudit.ErrEmptyTag) {
			t.Errorf("ValidateTag(%q) = %v, want ErrInvalidTag/ErrEmptyTag", c.tag, err)
		}
	}
}

func TestHashContent_Stable(t *testing.T) {
	h1 := skillaudit.HashContent("hello")
	h2 := skillaudit.HashContent("hello")
	h3 := skillaudit.HashContent("world")
	if h1 != h2 {
		t.Errorf("same content → different hash: %s vs %s", h1, h2)
	}
	if h1 == h3 {
		t.Errorf("different content → same hash")
	}
	if len(h1) != 64 {
		t.Errorf("hash length = %d, want 64 (SHA-256 hex)", len(h1))
	}
}

func TestFileMap(t *testing.T) {
	files := []skillaudit.FileSnap{
		{Path: "a.md", Content: "1"},
		{Path: "b.md", Content: "2"},
	}
	m := skillaudit.FileMap(files)
	if m["a.md"] != "1" || m["b.md"] != "2" {
		t.Errorf("FileMap = %+v", m)
	}
}

func TestSnapFileMaps_UnionPaths(t *testing.T) {
	left := []skillaudit.FileSnap{{Path: "a", Content: "1"}, {Path: "b", Content: "2"}}
	right := []skillaudit.FileSnap{{Path: "b", Content: "22"}, {Path: "c", Content: "3"}}
	mapL, mapR, paths := skillaudit.SnapFileMaps(left, right)
	if len(paths) != 3 {
		t.Errorf("paths len = %d, want 3 (%+v)", len(paths), paths)
	}
	if mapL["a"] != "1" || mapR["a"] != "" {
		t.Errorf("a in left only: %+v %+v", mapL, mapR)
	}
	if mapL["c"] != "" || mapR["c"] != "3" {
		t.Errorf("c in right only: %+v %+v", mapL, mapR)
	}
}
