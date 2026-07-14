package skilladapter

import "testing"

// TestHasHiddenSegment 验证隐藏目录/文件的过滤规则(2026-07-14 增)。
// 规则与 skillstore.walkFiles / listEmptyDirs 对齐:任一段以 . 开头即视为 hidden。
func TestHasHiddenSegment(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		// 正向(常见 skill 内容)
		{"SKILL.md", false},
		{"examples/x.md", false},
		{"deep/nested/file.json", false},
		{"a", false},

		// 反向(隐藏路径)
		{".skill-box/readme.md", true},
		{".skill-box/history.json", true},
		{".git/config", true},
		{".DS_Store", true},
		{"examples/.cache/x", true},  // 嵌套的隐藏段
		{"a/.b", true},
		{"./foo", true},             // 前导 ./
		{"a/./b", true},             // 中间 . 段

		// 边界
		{"", false},                 // 空字符串不视为隐藏
		{".", true},                 // 当前目录段
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			if got := HasHiddenSegment(c.in); got != c.want {
				t.Errorf("HasHiddenSegment(%q) = %v, want %v", c.in, got, c.want)
			}
		})
	}
}
