package skilltester_test

import (
	"strings"
	"testing"

	"ginp-api/internal/skilladapter"
	"ginp-api/internal/skilltester"
)

func okCanonical() skilladapter.Canonical {
	return skilladapter.Canonical{
		Manifest: skilladapter.Manifest{
			Name:        "demo",
			Version:     "0.1.0",
			Description: "this is a valid description for the demo skill that should pass lint",
			Triggers:    []string{"demo", "lint"},
		},
		Files: []skilladapter.File{{Path: "SKILL.md", Content: "# Demo\n\nbody content here"}},
	}
}

func TestLint_OK(t *testing.T) {
	r := skilltester.Lint(okCanonical())
	if r.Status != skilltester.StatusPassed {
		t.Errorf("expected passed, got %s (%s)", r.Status, r.Message)
	}
}

func TestLint_MissingName(t *testing.T) {
	c := okCanonical()
	c.Manifest.Name = ""
	r := skilltester.Lint(c)
	if r.Status != skilltester.StatusFailed {
		t.Errorf("expected failed, got %s", r.Status)
	}
	if !strings.Contains(r.Detail, "name_present") {
		t.Errorf("expected name_present in detail, got: %s", r.Detail)
	}
}

func TestLint_BadNameFormat(t *testing.T) {
	c := okCanonical()
	c.Manifest.Name = "Bad-Name" // 大写不允许
	r := skilltester.Lint(c)
	if r.Status != skilltester.StatusFailed {
		t.Errorf("expected failed, got %s", r.Status)
	}
}

func TestLint_ShortDescription(t *testing.T) {
	c := okCanonical()
	c.Manifest.Description = "short"
	r := skilltester.Lint(c)
	if r.Status != skilltester.StatusFailed {
		t.Errorf("expected failed, got %s", r.Status)
	}
}

func TestLint_DuplicateTriggers(t *testing.T) {
	c := okCanonical()
	c.Manifest.Triggers = []string{"demo", "demo", "lint"}
	r := skilltester.Lint(c)
	if r.Status != skilltester.StatusFailed {
		t.Errorf("expected failed, got %s", r.Status)
	}
}

func TestLint_NoTriggers(t *testing.T) {
	c := okCanonical()
	c.Manifest.Triggers = []string{}
	r := skilltester.Lint(c)
	if r.Status != skilltester.StatusFailed {
		t.Errorf("expected failed, got %s", r.Status)
	}
}

func TestLint_MissingSkillMD(t *testing.T) {
	c := okCanonical()
	c.Files = []skilladapter.File{{Path: "README.md", Content: "no skill md"}}
	r := skilltester.Lint(c)
	if r.Status != skilltester.StatusFailed {
		t.Errorf("expected failed, got %s", r.Status)
	}
}

func TestLint_BadPath(t *testing.T) {
	c := okCanonical()
	c.Files = []skilladapter.File{
		{Path: "SKILL.md", Content: "ok body"},
		{Path: "../etc/passwd", Content: "bad"},
	}
	r := skilltester.Lint(c)
	if r.Status != skilltester.StatusFailed {
		t.Errorf("expected failed, got %s", r.Status)
	}
}

func TestLint_HardcodedSecret(t *testing.T) {
	c := okCanonical()
	c.Files = []skilladapter.File{{Path: "SKILL.md", Content: "ok body\napi_key=sk-abcdef1234567890abcdef"}}
	r := skilltester.Lint(c)
	if r.Status != skilltester.StatusFailed {
		t.Errorf("expected failed, got %s", r.Status)
	}
	if !strings.Contains(r.Detail, "no_secrets") {
		t.Errorf("expected no_secrets in detail, got: %s", r.Detail)
	}
}

func TestRunScript_NoTest_Skipped(t *testing.T) {
	c := okCanonical()
	r := skilltester.RunScript(c, "", skilltester.Options{})
	if r.Status != skilltester.StatusSkipped {
		t.Errorf("expected skipped, got %s (%s)", r.Status, r.Message)
	}
}

func TestRunScript_BadCustomCommand(t *testing.T) {
	c := okCanonical()
	r := skilltester.RunScript(c, "", skilltester.Options{ScriptCommand: "rm -rf /; echo bad"})
	if r.Status != skilltester.StatusErrored {
		t.Errorf("expected errored, got %s", r.Status)
	}
}
