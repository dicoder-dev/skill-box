package toolspecs

import (
	"embed"
	"fmt"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// specsFS 嵌入 specs/ 目录的全部 .yaml 文件。
//
// 2026-06-30:yaml 文件不能是 symlink(//go:embed 不支持符号链接,见
// docs/agent/memory 项目里记录的踩坑),所以本目录只放实体文件,后续
// 若想从外部挂载,要走配置覆盖而非符号链接。
//
//go:embed all:specs
var specsFS embed.FS

// LoadAll 一次性加载全部内嵌 spec,返回按 tool_id 排序的列表。
//
// 失败语义:启动期硬错误,panic。spec 文件的合法性必须保证:任何不合法的
// spec 都是代码 bug,不能让生产服务起来后再报。
func LoadAll() ([]*ToolSpec, error) {
	entries, err := specsFS.ReadDir("specs")
	if err != nil {
		return nil, fmt.Errorf("toolspecs: read specs dir: %w", err)
	}
	out := make([]*ToolSpec, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") {
			continue
		}
		spec, err := loadOne("specs/" + name)
		if err != nil {
			return nil, fmt.Errorf("toolspecs: load %s: %w", name, err)
		}
		out = append(out, spec)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ToolID < out[j].ToolID })

	// 二次校验:tool_id 全局唯一
	seen := make(map[string]bool, len(out))
	for _, s := range out {
		if seen[s.ToolID] {
			return nil, fmt.Errorf("toolspecs: duplicate tool_id %q", s.ToolID)
		}
		seen[s.ToolID] = true
	}
	return out, nil
}

func loadOne(path string) (*ToolSpec, error) {
	data, err := specsFS.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}
	var s ToolSpec
	if err := yaml.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("unmarshal yaml: %w", err)
	}
	if err := s.Validate(); err != nil {
		return nil, err
	}
	return &s, nil
}
