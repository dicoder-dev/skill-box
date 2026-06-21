// Package skillaudit 提供"打 tag / diff / rollback"的纯计算核心。
//
// 设计要点(见 docs/project/需求规划.md 第 4.1.9 + 6.4-6.5 节):
//   - Tag = 用户对某 skill 在某时间点的内容快照(文件 + content hash)
//   - 文件由 caller 传入(从 skillstore.Load 拿);本包不直接读盘
//   - Diff 是文件级 + 行级两段式;文件级先按 path 对齐,行级走 LCS
//   - Rollback 由 caller 负责:本包只生成"目标文件集",不直接写盘
package skillaudit

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

// 业务错误(sentinel),service 层 wrap 业务语义。
var (
	ErrEmptyFiles    = errors.New("skillaudit: empty files")
	ErrEmptyTag      = errors.New("skillaudit: empty tag name")
	ErrInvalidTag    = errors.New("skillaudit: invalid tag name")
	ErrIdentical      = errors.New("skillaudit: identical files")
)

// FileSnap 一次 tag / diff 里的一个文件视图。
type FileSnap struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

// HashContent 算 content 的 SHA-256(用 hex 编码 64 字符)。
func HashContent(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}

// ValidateTag 校验 tag 名(semver-like / 普通短名)。
// 允许:字母/数字/点/下划线/连字符/单斜线(子路径);长度 1~64;不为纯 . 也不为 ..
func ValidateTag(tag string) error {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return ErrEmptyTag
	}
	if len(tag) > 64 {
		return fmt.Errorf("%w: too long (%d)", ErrInvalidTag, len(tag))
	}
	if tag == "." || tag == ".." {
		return fmt.Errorf("%w: %q", ErrInvalidTag, tag)
	}
	for _, r := range tag {
		if r >= 'a' && r <= 'z' {
			continue
		}
		if r >= 'A' && r <= 'Z' {
			continue
		}
		if r >= '0' && r <= '9' {
			continue
		}
		if r == '.' || r == '-' || r == '_' || r == '/' {
			continue
		}
		return fmt.Errorf("%w: invalid char %q", ErrInvalidTag, r)
	}
	return nil
}

// FileMap 文件 path -> content,内部排序用。
func FileMap(files []FileSnap) map[string]string {
	out := make(map[string]string, len(files))
	for _, f := range files {
		out[f.Path] = f.Content
	}
	return out
}

// SnapFileMaps 把"两个 FileSnap 切片"变成 path → (left, right) 视图。
// 排序按 path 字典序稳定。
func SnapFileMaps(left, right []FileSnap) (mapL, mapR map[string]string, allPaths []string) {
	mapL = FileMap(left)
	mapR = FileMap(right)
	set := make(map[string]struct{}, len(mapL)+len(mapR))
	for p := range mapL {
		set[p] = struct{}{}
	}
	for p := range mapR {
		set[p] = struct{}{}
	}
	allPaths = make([]string, 0, len(set))
	for p := range set {
		allPaths = append(allPaths, p)
	}
	sortStrings(allPaths)
	return
}

func sortStrings(s []string) {
	// 走 sort 包避免额外的 import 噪声
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j-1] > s[j]; j-- {
			s[j-1], s[j] = s[j], s[j-1]
		}
	}
}

// FileDiff 一对文件比较的产物。
type FileDiff struct {
	Path      string     `json:"path"`
	Kind      string     `json:"kind"` // added / removed / modified / unchanged
	LeftHash  string     `json:"left_hash,omitempty"`
	RightHash string     `json:"right_hash,omitempty"`
	Lines     []DiffLine `json:"lines,omitempty"`
}

// DiffLine 单行 diff 结果。
type DiffLine struct {
	Kind    string `json:"kind"` // context / added / removed
	LeftNo  int    `json:"left_no,omitempty"`
	RightNo int    `json:"right_no,omitempty"`
	Text    string `json:"text"`
}

// Diff 两组文件按 path 对齐,产出 FileDiff 列表。
// 排序:added → removed → modified → unchanged;每类内 path 字典序。
func Diff(left, right []FileSnap) []FileDiff {
	mapL, mapR, paths := SnapFileMaps(left, right)
	out := make([]FileDiff, 0, len(paths))
	for _, p := range paths {
		l, lok := mapL[p]
		r, rok := mapR[p]
		fd := FileDiff{Path: p, LeftHash: HashContent(l), RightHash: HashContent(r)}
		switch {
		case !lok && rok:
			fd.Kind = "added"
			fd.Lines = linesAdded(r)
		case lok && !rok:
			fd.Kind = "removed"
			fd.Lines = linesRemoved(l)
		case lok && rok && l != r:
			fd.Kind = "modified"
			fd.Lines = LinesDiff(l, r)
		default:
			fd.Kind = "unchanged"
		}
		out = append(out, fd)
	}
	// 排序
	sortFileDiffs(out)
	return out
}

func sortFileDiffs(ds []FileDiff) {
	// 稳定排序:按 Kind 优先级 + Path 字典序
	priority := map[string]int{"added": 0, "removed": 1, "modified": 2, "unchanged": 3}
	// 插入排序
	for i := 1; i < len(ds); i++ {
		for j := i; j > 0; j-- {
			if lessFileDiff(ds[j], ds[j-1], priority) {
				ds[j-1], ds[j] = ds[j], ds[j-1]
			} else {
				break
			}
		}
	}
}

func lessFileDiff(a, b FileDiff, pri map[string]int) bool {
	pa, pb := pri[a.Kind], pri[b.Kind]
	if pa != pb {
		return pa < pb
	}
	return a.Path < b.Path
}

func linesAdded(s string) []DiffLine {
	lines := splitLines(s)
	out := make([]DiffLine, 0, len(lines))
	for i, ln := range lines {
		out = append(out, DiffLine{Kind: "added", RightNo: i + 1, Text: ln})
	}
	return out
}

func linesRemoved(s string) []DiffLine {
	lines := splitLines(s)
	out := make([]DiffLine, 0, len(lines))
	for i, ln := range lines {
		out = append(out, DiffLine{Kind: "removed", LeftNo: i + 1, Text: ln})
	}
	return out
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}
