// fsutil_test.go - 单元测试:InspectProject / slugify。
//
// 这两个能力是"导入项目"流程的核心启发逻辑,需要保证:
//   - 正常目录:能取到 basename 作为 name,slugify 后作为 alias
//   - 路径不存在 / 不是目录:返 error
//   - slugify 规则:小写 + 非字母数字折叠成单个 '-' + trim 收尾 '-'
package fsutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSlugify(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"My App", "my-app"},
		{"foo_bar.baz", "foo-bar-baz"},
		{"Hello World!!", "hello-world"},
		{"你好-world", "world"},
		{"---", ""},
		{"  ", ""},
		{"a", "a"},
		{"A1B2C3", "a1b2c3"},
		{"__foo__", "foo"},
		{"foo..bar", "foo-bar"},
	}
	for _, c := range cases {
		got := slugify(c.in)
		if got != c.want {
			t.Errorf("slugify(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestInspectProject_OK(t *testing.T) {
	// 临时建一个目录,basename 已知
	dir := t.TempDir()
	// 真实 tmpdir 的最后一段 basename 在不同系统上都非空
	base := filepath.Base(dir)
	hint, err := InspectProject(dir)
	if err != nil {
		t.Fatalf("InspectProject(%q) err: %v", dir, err)
	}
	if hint.Name != base {
		t.Errorf("Name = %q, want %q", hint.Name, base)
	}
	if hint.Alias == "" {
		t.Errorf("Alias should be non-empty for normal dir name %q", base)
	}
}

func TestInspectProject_NotDir(t *testing.T) {
	// 拿一个真实文件(测试源码自身)当作非目录输入
	dir := t.TempDir()
	file := filepath.Join(dir, "regular.txt")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if _, err := InspectProject(file); err == nil {
		t.Errorf("InspectProject on a file should return error")
	}
}

func TestInspectProject_Nonexistent(t *testing.T) {
	if _, err := InspectProject("/this/path/should/not/exist/skill-box-test-xyz"); err == nil {
		t.Errorf("InspectProject on nonexistent path should return error")
	}
}
