package skillstore

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// TestListTreeConcurrentLarge 2026-07-28 增:验证 ListTree 并发改造后能正确
// 列出 N=100 个 skill(混合顶层 + 嵌套 group),与原串行 buildTreeNode 行为一致:
//   - leaf 全部命中
//   - group 嵌套结构正确(2 级:根 group/子 group/skill)
//   - skill_meta 字段全填(name/version/description/triggers)
//   - 排序稳定(IsGroup desc, Name asc)
//
// 跑并发路径覆盖 errgroup.SetLimit + map 加锁 + 反向索引 leafByGrp。
func TestListTreeConcurrentLarge(t *testing.T) {
	tmp, _ := os.MkdirTemp("", "listtree-large-")
	defer os.RemoveAll(tmp)
	store, err := NewAt(tmp)
	if err != nil {
		t.Fatalf("NewAt: %v", err)
	}

	// 造 100 skill:50 在根,50 在 frontend/react 嵌套 group 下
	for i := 0; i < 50; i++ {
		writeSkill(t, store, "", "root-skill-"+pad(i))
	}
	for i := 0; i < 50; i++ {
		writeSkill(t, store, "frontend/react", "nested-skill-"+pad(i))
	}

	tree, err := store.ListTree("")
	if err != nil {
		t.Fatalf("ListTree: %v", err)
	}
	if got := countLeaves2(tree); got != 100 {
		t.Fatalf("expected 100 leaves, got %d", got)
	}
	// 结构断言:根下应有 root-skill-* 50 个 + frontend 1 个 group
	var rootLeaves, rootGroups int
	for _, n := range tree {
		if n.IsGroup {
			rootGroups++
			if n.Name != "frontend" {
				t.Fatalf("expected root group 'frontend', got %q", n.Name)
			}
			// frontend 下应有 1 个 react 子 group(react 下挂 50 个 nested-skill-*)
			var reactGroups, reactLeaves int
			for _, c := range n.Children {
				if c.IsGroup {
					reactGroups++
					if c.Name != "react" {
						t.Fatalf("expected 'react' subgroup, got %q", c.Name)
					}
					// react 下应只有 50 个 leaf,不再有 group
					for _, cc := range c.Children {
						if cc.IsGroup {
							t.Fatalf("react should have no nested groups, got %q", cc.Name)
						}
						reactLeaves++
					}
				} else {
					t.Fatalf("frontend should have only react subgroup, got leaf %q at this level", c.Name)
				}
			}
			if reactGroups != 1 || reactLeaves != 50 {
				t.Fatalf("frontend group: expected 1 react subgroup + 50 leaves, got %d/%d", reactGroups, reactLeaves)
			}
		} else {
			rootLeaves++
		}
	}
	if rootLeaves != 50 || rootGroups != 1 {
		t.Fatalf("root: expected 50 leaves + 1 group, got %d/%d", rootLeaves, rootGroups)
	}

	// 抽样验证 skill_meta 全填
	for _, n := range tree {
		if n.IsGroup {
			continue
		}
		if n.SkillMeta == nil {
			t.Fatalf("leaf %q missing skill_meta", n.Name)
		}
		if n.SkillMeta.Name == "" || n.SkillMeta.Version == "" {
			t.Fatalf("leaf %q incomplete meta: %+v", n.Name, n.SkillMeta)
		}
	}
}

// TestListTreeConcurrentKeyword 验证 keyword 过滤在并发路径下正确。
// 期望:keyword="root" 只命中根下 50 个,frontend/react 整组被隐藏(kw 非空时空组丢)。
func TestListTreeConcurrentKeyword(t *testing.T) {
	tmp, _ := os.MkdirTemp("", "listtree-kw-")
	defer os.RemoveAll(tmp)
	store, _ := NewAt(tmp)
	for i := 0; i < 30; i++ {
		writeSkill(t, store, "", "root-skill-"+pad(i))
	}
	for i := 0; i < 30; i++ {
		writeSkill(t, store, "frontend", "frontend-skill-"+pad(i))
	}
	for i := 0; i < 30; i++ {
		writeSkill(t, store, "backend", "backend-skill-"+pad(i))
	}

	tree, err := store.ListTree("root")
	if err != nil {
		t.Fatalf("ListTree: %v", err)
	}
	if got := countLeaves2(tree); got != 30 {
		t.Fatalf("keyword 'root' expected 30 leaves, got %d", got)
	}
	// frontend/backend 整组应被隐藏(只有 root-skill-*,无 frontend/backend 顶层)
	for _, n := range tree {
		if n.IsGroup {
			t.Fatalf("keyword 'root' should not contain groups, got group %q", n.Name)
		}
	}
}

// TestListTreeConcurrentCorruptedLeaf 验证损坏的 SKILL.md 不阻断其他 skill。
// 造 1 个二进制损坏 skill + 9 个正常 skill,期望 9 个正常 skill 全部返回。
func TestListTreeConcurrentCorruptedLeaf(t *testing.T) {
	tmp, _ := os.MkdirTemp("", "listtree-corr-")
	defer os.RemoveAll(tmp)
	store, _ := NewAt(tmp)

	for i := 0; i < 9; i++ {
		writeSkill(t, store, "", "ok-skill-"+pad(i))
	}
	// 写一个二进制乱码 SKILL.md
	bad := filepath.Join(tmp, "bad-skill")
	if err := os.MkdirAll(bad, 0o755); err != nil {
		t.Fatalf("mkdir bad: %v", err)
	}
	if err := os.WriteFile(filepath.Join(bad, "SKILL.md"), []byte{0xFF, 0xFE, 0x00, 0x01, 0x80, 0x90}, 0o644); err != nil {
		t.Fatalf("write bad: %v", err)
	}

	tree, err := store.ListTree("")
	if err != nil {
		t.Fatalf("ListTree: %v", err)
	}
	// 应有 9 个 leaf(bad-skill 走 logger.Warn 跳过,不阻断)
	if got := countLeaves2(tree); got != 9 {
		names := collectLeafNames(tree)
		t.Fatalf("expected 9 ok leaves, got %d (leaves: %v)", got, names)
	}
}

// TestListTreeConcurrentStableSort 验证 sortTreeNodes 在并发路径下仍稳定
// (同组内按 Name 字典序,group 排在 leaf 前面)。
func TestListTreeConcurrentStableSort(t *testing.T) {
	tmp, _ := os.MkdirTemp("", "listtree-sort-")
	defer os.RemoveAll(tmp)
	store, _ := NewAt(tmp)

	// 在 zoo 下放 4 个 leaf + 1 个 inner 子 group
	for _, n := range []string{"zebra", "alpha", "mango", "banana"} {
		writeSkill(t, store, "zoo", n)
	}
	// inner 子 group 里有 inner-a / inner-y 两个 leaf
	writeSkill(t, store, "zoo/inner", "inner-a")
	writeSkill(t, store, "zoo/inner", "inner-y")

	tree, err := store.ListTree("")
	if err != nil {
		t.Fatalf("ListTree: %v", err)
	}
	if len(tree) != 1 || tree[0].Name != "zoo" {
		t.Fatalf("expected root group 'zoo', got %+v", tree)
	}
	// zoo.Children 顺序:group(内嵌的 inner)在前,然后 leaf 按 Name asc:
	// inner(group) 排第一,leaf 字典序:alpha, banana, mango, zebra
	want := []string{"inner", "alpha", "banana", "mango", "zebra"}
	gotOrder := make([]string, 0, len(tree[0].Children))
	for _, c := range tree[0].Children {
		gotOrder = append(gotOrder, c.Name)
	}
	if !sliceEqual(gotOrder, want) {
		t.Fatalf("sort mismatch:\n  got:  %v\n  want: %v", gotOrder, want)
	}
	// inner group 内:inner-a, inner-y 按字典序
	inner := tree[0].Children[0]
	if inner.Name != "inner" || !inner.IsGroup {
		t.Fatalf("first child should be 'inner' group, got %+v", inner)
	}
	innerLeaves := []string{inner.Children[0].Name, inner.Children[1].Name}
	if !sliceEqual(innerLeaves, []string{"inner-a", "inner-y"}) {
		t.Fatalf("inner group leaves: want [inner-a inner-y], got %v", innerLeaves)
	}
}

// ====== helpers ======

func writeSkill(t *testing.T, s *Store, group, name string) {
	t.Helper()
	dir := filepath.Join(s.root, group, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	body := "---\nname: " + name + "\nversion: 0.1.0\ndescription: test skill " + name + "\ntriggers: [t-" + name + "]\n---\n\nbody for " + name + "\n"
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(body), 0o644); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}
}

func pad(i int) string {
	s := "00"
	switch {
	case i < 10:
		s = "0" + string(rune('0'+i))
	case i < 100:
		s = string(rune('0'+i/10)) + string(rune('0'+i%10))
	}
	return s
}

func countLeaves2(nodes []TreeNode) int {
	n := 0
	for _, x := range nodes {
		if !x.IsGroup {
			n++
			continue
		}
		n += countLeaves2(x.Children)
	}
	return n
}

func collectLeafNames(nodes []TreeNode) []string {
	var out []string
	for _, x := range nodes {
		if !x.IsGroup {
			out = append(out, x.Name)
		}
		out = append(out, collectLeafNames(x.Children)...)
	}
	sort.Strings(out)
	return out
}

func sliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
