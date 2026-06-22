package skillbundle

import (
	"regexp"
	"strings"
	"testing"

	"ginp-api/internal/skilladapter"
)

func TestLoadAll_Succeeds(t *testing.T) {
	all, err := LoadAll()
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}
	if got, want := len(all), len(All); got != want {
		t.Fatalf("len = %d, want %d", got, want)
	}
}

func TestLoadAll_PreservesOrder(t *testing.T) {
	all, err := LoadAll()
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}
	for i, c := range all {
		if c.Manifest.Name == "" {
			t.Fatalf("idx %d: empty name", i)
		}
		// 顺序按 All 中的 Key 推;但 manifest.Name 是 normalize 后的,
		// Key 通常已经是合法 name(纯 ascii + '-'),所以应该相等
		if c.Manifest.Name != All[i].Key {
			t.Errorf("idx %d: name=%q, want %q", i, c.Manifest.Name, All[i].Key)
		}
	}
}

func TestLoadAll_ManifestValid(t *testing.T) {
	all, err := LoadAll()
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}
	nameRE := regexp.MustCompile(`^[a-z][a-z0-9-]{1,63}$`)
	versionRE := regexp.MustCompile(`^v?\d+\.\d+\.\d+([-+].+)?$`)
	for _, c := range all {
		m := c.Manifest
		if !nameRE.MatchString(m.Name) {
			t.Errorf("%s: name %q 不合法", m.Name, m.Name)
		}
		if !versionRE.MatchString(m.Version) {
			t.Errorf("%s: version %q 不合法", m.Name, m.Version)
		}
		if l := len(m.Description); l < 10 || l > 500 {
			t.Errorf("%s: description 长度 %d 超出 [10,500]", m.Name, l)
		}
		if len(m.Triggers) < 1 || len(m.Triggers) > 10 {
			t.Errorf("%s: triggers 数 %d 超出 [1,10]", m.Name, len(m.Triggers))
		}
		if len(m.TargetTools) == 0 {
			t.Errorf("%s: target_tools 为空", m.Name)
		}
		// target_tools 必须是 AllTools 的子集
		allowed := map[string]bool{}
		for _, id := range skilladapter.AllTools {
			allowed[id] = true
		}
		for _, tool := range m.TargetTools {
			if !allowed[tool] {
				t.Errorf("%s: target_tools 含未知工具 %q", m.Name, tool)
			}
		}
	}
}

func TestLoadAll_HasSKILLMd(t *testing.T) {
	all, err := LoadAll()
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}
	for _, c := range all {
		found := false
		for _, f := range c.Files {
			if f.Path == "SKILL.md" {
				found = true
				if !strings.HasPrefix(f.Content, "---") {
					t.Errorf("%s: SKILL.md 缺少 frontmatter", c.Manifest.Name)
				}
				if !strings.Contains(f.Content, c.Manifest.Name) {
					t.Errorf("%s: SKILL.md body 未引用 name", c.Manifest.Name)
				}
				break
			}
		}
		if !found {
			t.Errorf("%s: Files 缺少 SKILL.md", c.Manifest.Name)
		}
	}
}

func TestLoadAll_ExamplesLoaded(t *testing.T) {
	all, err := LoadAll()
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}
	for _, c := range all {
		// 至少要有一个 examples/*.sh(设计如此)
		count := 0
		for _, f := range c.Files {
			if strings.HasPrefix(f.Path, "examples/") && strings.HasSuffix(f.Path, ".sh") {
				count++
				if !strings.HasPrefix(f.Content, "#!/") {
					t.Errorf("%s: %s 不是可执行脚本", c.Manifest.Name, f.Path)
				}
			}
		}
		if count == 0 {
			t.Errorf("%s: examples/ 下没有任何 .sh", c.Manifest.Name)
		}
	}
}

func TestLoadOne_NotFound(t *testing.T) {
	_, err := LoadOne("presets/__nope__")
	if err == nil {
		t.Fatal("expected error for non-existent preset, got nil")
	}
}

func TestKeys_Stable(t *testing.T) {
	keys := Keys()
	if len(keys) != len(All) {
		t.Fatalf("Keys len = %d, want %d", len(keys), len(All))
	}
	// 不能重复
	seen := map[string]bool{}
	for _, k := range keys {
		if seen[k] {
			t.Errorf("duplicate key %q", k)
		}
		seen[k] = true
	}
}