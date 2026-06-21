package skillpkg

import (
	"archive/zip"
	"bytes"
	"errors"
	"strings"
	"testing"

	"ginp-api/internal/skilladapter"
)

// fakeProvider 模拟 sskill,按 (name,version) 查 canonical;查不到 ok=false。
type fakeProvider struct {
	store  map[string]skilladapter.Canonical
	errors map[string]error // key="name@version" 强制返错
}

func (f *fakeProvider) LoadCanonical(scope string, projectID uint, name, version string) (skilladapter.Canonical, bool, error) {
	key := name + "@" + version
	if e, ok := f.errors[key]; ok {
		return skilladapter.Canonical{}, false, e
	}
	c, ok := f.store[key]
	return c, ok, nil
}

func TestBuildBytes_Empty(t *testing.T) {
	_, _, err := BuildBytes(ExportRequest{}, &fakeProvider{store: map[string]skilladapter.Canonical{}})
	if !errors.Is(err, ErrEmptySkills) {
		t.Fatalf("want ErrEmptySkills, got %v", err)
	}
}

func TestBuildBytes_OneSkill_OK(t *testing.T) {
	can := skilladapter.Canonical{
		Manifest: skilladapter.Manifest{
			Name: "review-pr", Version: "0.1.0", Description: "review a PR",
			Triggers: []string{"review pr", "code review"},
		},
		Files: []skilladapter.File{
			{Path: "SKILL.md", Content: "# review-pr\n\nThis is a skill."},
			{Path: "examples/run.sh", Content: "#!/bin/sh\necho hi"},
		},
	}
	p := &fakeProvider{store: map[string]skilladapter.Canonical{"review-pr@0.1.0": can}}

	b, fails, err := BuildBytes(ExportRequest{
		Skills:     []SkillRef{{Scope: "global", Name: "review-pr", Version: "0.1.0"}},
		SourceApp:  "skillbox",
		SourceDesc: "unit test",
	}, p)
	if err != nil {
		t.Fatalf("BuildBytes: %v", err)
	}
	if len(fails) > 0 {
		t.Fatalf("unexpected failures: %v", fails)
	}
	if len(b) == 0 {
		t.Fatal("empty zip")
	}

	// 验证 zip 能读
	zr, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		t.Fatalf("open zip: %v", err)
	}
	got := map[string]string{}
	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			t.Fatalf("open entry: %v", err)
		}
		buf := new(bytes.Buffer)
		if _, err := buf.ReadFrom(rc); err != nil {
			t.Fatalf("read entry: %v", err)
		}
		rc.Close()
		got[f.Name] = buf.String()
	}

	// 必有 manifest.json
	mf, ok := got["manifest.json"]
	if !ok {
		t.Fatal("manifest.json missing")
	}
	if !strings.Contains(mf, `"pkg_format": "skillbox.v1"`) {
		t.Errorf("manifest pkg_format wrong: %s", mf)
	}
	if !strings.Contains(mf, `"name": "review-pr"`) {
		t.Errorf("manifest skill index wrong: %s", mf)
	}
	// 必有 skill.yaml + SKILL.md + examples/run.sh
	for _, want := range []string{
		"skills/review-pr@0.1.0/skill.yaml",
		"skills/review-pr@0.1.0/SKILL.md",
		"skills/review-pr@0.1.0/examples/run.sh",
	} {
		if _, ok := got[want]; !ok {
			t.Errorf("missing entry: %s", want)
		}
	}
	// 验证 skill.yaml 内容
	yaml := got["skills/review-pr@0.1.0/skill.yaml"]
	if !strings.Contains(yaml, "name: review-pr") {
		t.Errorf("yaml missing name: %s", yaml)
	}
	if !strings.Contains(yaml, "triggers:") {
		t.Errorf("yaml missing triggers: %s", yaml)
	}
}

func TestBuildBytes_PartialFailure(t *testing.T) {
	p := &fakeProvider{
		store: map[string]skilladapter.Canonical{
			"a@0.1.0": {
				Manifest: skilladapter.Manifest{Name: "a", Version: "0.1.0", Description: "long enough desc."},
				Files:    []skilladapter.File{{Path: "SKILL.md", Content: "x"}},
			},
		},
		errors: map[string]error{"b@0.1.0": errors.New("disk fail")},
	}
	b, fails, err := BuildBytes(ExportRequest{
		Skills: []SkillRef{
			{Scope: "global", Name: "a", Version: "0.1.0"},
			{Scope: "global", Name: "b", Version: "0.1.0"},
		},
	}, p)
	if err != nil {
		t.Fatalf("BuildBytes: %v", err)
	}
	if len(fails) != 1 {
		t.Fatalf("want 1 failure, got %v", fails)
	}
	if !strings.Contains(fails[0], "b@0.1.0") {
		t.Errorf("failure should mention b: %s", fails[0])
	}
	if len(b) == 0 {
		t.Fatal("partial should still produce zip")
	}
}

func TestBuildBytes_AllFail(t *testing.T) {
	p := &fakeProvider{
		store: map[string]skilladapter.Canonical{},
		errors: map[string]error{"x@1": errors.New("nope")},
	}
	_, fails, err := BuildBytes(ExportRequest{
		Skills: []SkillRef{{Scope: "global", Name: "x", Version: "1"}},
	}, p)
	if err == nil {
		t.Fatal("want err when all fail")
	}
	if len(fails) == 0 {
		t.Fatal("want failures list")
	}
}
