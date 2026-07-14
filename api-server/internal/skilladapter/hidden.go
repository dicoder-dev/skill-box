package skilladapter

import (
	"path/filepath"
	"strings"
)

// HasHiddenSegment 报告相对路径 p 是否包含以 . 开头的段。
//
// 用于 copy 模式双保险 + AI prompt 喂数据前过滤,确保 .skill-box/、.git/ 等
// 隐藏路径不会:
//
//   - 被 BaseAdapter.Apply 写盘;
//   - 被 readDirFiles 加载进 Canonical.Files;
//   - 被 buildSkillMDForPrompt 拼到 AI prompt。
//
// 规则与 skillstore.walkFiles / listEmptyDirs 对齐,统一为"任一段以 . 开头"。
//
//   - "SKILL.md" / "examples/x.md" → false
//   - ".skill-box/history.json" / "examples/.cache/x" → true
func HasHiddenSegment(p string) bool {
	for _, seg := range strings.Split(filepath.ToSlash(p), "/") {
		if strings.HasPrefix(seg, ".") {
			return true
		}
	}
	return false
}
