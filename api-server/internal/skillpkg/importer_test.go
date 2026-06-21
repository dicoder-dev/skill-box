package skillpkg

import (
	"errors"
	"sync"
	"testing"

	"ginp-api/internal/skilladapter"
)

type fakeInstaller struct {
	mu        sync.Mutex
	installed []skilladapter.Canonical
	failKey   string // 命中即返错
}

func (f *fakeInstaller) InstallCanonical(scope string, projectID uint, c skilladapter.Canonical, source string) (uint, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if c.Manifest.Name+"@"+c.Manifest.Version == f.failKey {
		return 0, errors.New("install refused")
	}
	f.installed = append(f.installed, c)
	return uint(len(f.installed)), nil
}

func newTestPackage(t *testing.T) []byte {
	t.Helper()
	p := &fakeProvider{store: map[string]skilladapter.Canonical{
		"alpha@0.1.0": {
			Manifest: skilladapter.Manifest{
				Name: "alpha", Version: "0.1.0", Description: "an alpha skill",
				Triggers: []string{"alpha"},
			},
			Files: []skilladapter.File{{Path: "SKILL.md", Content: "alpha body"}},
		},
		"beta@1.0.0": {
			Manifest: skilladapter.Manifest{
				Name: "beta", Version: "1.0.0", Description: "a beta skill",
				Triggers: []string{"beta"},
			},
			Files: []skilladapter.File{
				{Path: "SKILL.md", Content: "beta body"},
				{Path: "extra.md", Content: "more"},
			},
		},
	}}
	b, _, err := BuildBytes(ExportRequest{
		Skills: []SkillRef{
			{Scope: "global", Name: "alpha", Version: "0.1.0"},
			{Scope: "global", Name: "beta", Version: "1.0.0"},
		},
	}, p)
	if err != nil {
		t.Fatalf("BuildBytes: %v", err)
	}
	return b
}

func TestParseManifest_Ok(t *testing.T) {
	b := newTestPackage(t)
	mf, err := ParseManifest(b)
	if err != nil {
		t.Fatalf("ParseManifest: %v", err)
	}
	if mf.PkgFormat != "skillbox.v1" {
		t.Errorf("pkg_format: %s", mf.PkgFormat)
	}
	if len(mf.Skills) != 2 {
		t.Errorf("want 2 skills, got %d", len(mf.Skills))
	}
}

func TestParseManifest_NotZip(t *testing.T) {
	_, err := ParseManifest([]byte("not a zip"))
	if !errors.Is(err, ErrInvalidManifest) && err == nil {
		t.Fatalf("want err, got %v", err)
	}
}

func TestParseManifest_NoManifest(t *testing.T) {
	// 构造一个不含 manifest.json 的 zip
	// 直接调用 Install 走 parseAll 路径
	_, _, err := (&Importer{Provider: &fakeInstaller{}}).parseAll([]byte("not a zip"))
	if err == nil {
		t.Fatal("want err on non-zip")
	}
}

func TestInstall_AllOK(t *testing.T) {
	b := newTestPackage(t)
	inst := &fakeInstaller{}
	out, err := NewImporter(inst).Install(b, ImportRequest{TargetScope: "global"})
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	if out.Total != 2 || out.OK != 2 || out.Failed != 0 {
		t.Errorf("bad counts: %+v", out)
	}
	if len(inst.installed) != 2 {
		t.Errorf("installer not called twice: %d", len(inst.installed))
	}
}

func TestInstall_Selective(t *testing.T) {
	b := newTestPackage(t)
	inst := &fakeInstaller{}
	out, err := NewImporter(inst).Install(b, ImportRequest{
		TargetScope: "global",
		Skills:      []ImportSkillEntry{{Key: "alpha@0.1.0"}},
	})
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	if out.Total != 1 || out.OK != 1 {
		t.Errorf("want 1 ok, got %+v", out)
	}
}

func TestInstall_PartialFail(t *testing.T) {
	b := newTestPackage(t)
	inst := &fakeInstaller{failKey: "alpha@0.1.0"}
	out, err := NewImporter(inst).Install(b, ImportRequest{TargetScope: "global"})
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	if out.OK != 1 || out.Failed != 1 {
		t.Errorf("want 1/1, got %+v", out)
	}
	// 检查 alpha 失败、beta 成功
	got := map[string]bool{}
	for _, it := range out.Items {
		got[it.Key] = it.OK
	}
	if got["alpha@0.1.0"] {
		t.Error("alpha should fail")
	}
	if !got["beta@1.0.0"] {
		t.Error("beta should pass")
	}
}

func TestInstall_UnknownKey(t *testing.T) {
	b := newTestPackage(t)
	inst := &fakeInstaller{}
	out, err := NewImporter(inst).Install(b, ImportRequest{
		TargetScope: "global",
		Skills:      []ImportSkillEntry{{Key: "ghost@9"}},
	})
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	if out.OK != 0 || out.Failed != 1 {
		t.Errorf("want 0/1, got %+v", out)
	}
}

func TestInstall_BadScope(t *testing.T) {
	b := newTestPackage(t)
	_, err := NewImporter(&fakeInstaller{}).Install(b, ImportRequest{TargetScope: "oops"})
	if err == nil {
		t.Fatal("want err on bad scope")
	}
}

func TestInstall_ProjectRequiresID(t *testing.T) {
	b := newTestPackage(t)
	_, err := NewImporter(&fakeInstaller{}).Install(b, ImportRequest{TargetScope: "project", ProjectID: 0})
	if err == nil {
		t.Fatal("want err when project_id missing")
	}
}
