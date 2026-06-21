package skillaudit_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"ginp-api/internal/skillaudit"
)

func TestBuildTag_OK(t *testing.T) {
	out, err := skillaudit.BuildTag(skillaudit.TagSnapshot{
		SkillID: 1,
		Tag:     "v1.0.0",
		Message: "release",
		Files: []skillaudit.FileSnap{
			{Path: "a.md", Content: "x"},
			{Path: "b.md", Content: "y"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.TagID != 0 {
		t.Errorf("TagID should be 0 (set by DB layer), got %d", out.TagID)
	}
	if out.Tag != "v1.0.0" || out.Message != "release" {
		t.Errorf("meta: %+v", out)
	}
	if len(out.Files) != 2 {
		t.Fatalf("files = %d, want 2", len(out.Files))
	}
	if out.Files[0].ContentHash != skillaudit.HashContent("x") {
		t.Errorf("hash mismatch")
	}
}

func TestBuildTag_EmptyFiles_Err(t *testing.T) {
	_, err := skillaudit.BuildTag(skillaudit.TagSnapshot{SkillID: 1, Tag: "v1", Files: nil})
	if !errors.Is(err, skillaudit.ErrEmptyFiles) {
		t.Errorf("err = %v, want ErrEmptyFiles", err)
	}
}

func TestBuildTag_EmptyPath_Err(t *testing.T) {
	_, err := skillaudit.BuildTag(skillaudit.TagSnapshot{
		SkillID: 1, Tag: "v1",
		Files: []skillaudit.FileSnap{{Path: "", Content: "x"}},
	})
	if !errors.Is(err, skillaudit.ErrEmptyFiles) {
		t.Errorf("err = %v, want ErrEmptyFiles", err)
	}
}

func TestBuildTag_InvalidTag_Err(t *testing.T) {
	_, err := skillaudit.BuildTag(skillaudit.TagSnapshot{
		SkillID: 1, Tag: "",
		Files: []skillaudit.FileSnap{{Path: "a", Content: "x"}},
	})
	if !errors.Is(err, skillaudit.ErrEmptyTag) {
		t.Errorf("err = %v, want ErrEmptyTag", err)
	}
}

func TestImplicitPreRollbackTag_Format(t *testing.T) {
	ts := time.Date(2026, 1, 2, 15, 4, 5, 0, time.UTC)
	got := skillaudit.ImplicitPreRollbackTag(ts)
	want := "_pre_rollback_20260102T150405"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if !strings.HasPrefix(got, "_pre_rollback_") {
		t.Errorf("missing prefix: %q", got)
	}
}

func TestFilesFromTagged_RoundTrip(t *testing.T) {
	tagged := []skillaudit.TaggedFile{
		{Path: "a", Content: "1", ContentHash: "h1"},
		{Path: "b", Content: "2", ContentHash: "h2"},
	}
	back := skillaudit.FilesFromTagged(tagged)
	if len(back) != 2 {
		t.Fatalf("len = %d", len(back))
	}
	if back[0].Path != "a" || back[0].Content != "1" {
		t.Errorf("round-trip lost data: %+v", back[0])
	}
}
