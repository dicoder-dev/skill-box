package entity

import (
	"ginp-api/internal/gapi/typ"
	"ginp-api/internal/gen"
)

const tableNameToolPath = "tool_paths"

// ToolPath 单个工具的一条扫描/写盘路径。
//
// 2026-06-30 二改:e_tool 的子表,一对多关系。一个工具可以有多个
// global+user / global+system / project+user / project+system 路径组合
// (例如 Codex 同时有 ~/.agents/skills / ~/.codex/skills/.system /
//  ~/.codex/vendor_imports/.curated / <project>/.agents/skills 4 条)。
//
// 关键约束:
//   - 同一 (tool_id, scope, category, path) 唯一(uniqueIndex),防重复
//   - 删 e_tool 行时 service 层事务里清掉本表对应行(避免悬空路径)
//   - path 字段保留 ~/ 形式(运行时由 BaseAdapter 展开),不展开为绝对路径;
//     理由:不同用户(系统)可能共享同一个 DB 快照,但 home 不同
type ToolPath struct {
	ID        uint   `gorm:"primaryKey;autoIncrement" json:"id,omitempty"`
	ToolID    uint   `gorm:"column:tool_id;index;comment:所属工具(逻辑外键,删 tool 级联)" json:"tool_id,omitempty"`
	Scope     string `gorm:"type:varchar(16);column:scope;index;comment:global|project" json:"scope,omitempty"`
	Category  string `gorm:"type:varchar(16);column:category;index;comment:user|system" json:"category,omitempty"`
	Path      string `gorm:"type:varchar(512);column:path;comment:绝对路径或相对路径(含 ~/)" json:"path,omitempty"`
	PathOrder int    `gorm:"column:path_order;comment:同一 (scope,category) 内的顺序" json:"path_order,omitempty"`
}

var _ typ.IEntity = (*ToolPath)(nil)

func (ToolPath) GenConfig() *gen.EntityConfig {
	return &gen.EntityConfig{
		TableName: tableNameToolPath,
	}
}

func (ToolPath) GenEnumOptions() []typ.EntityEnumOption {
	return nil
}

func (ToolPath) TableName() string {
	return tableNameToolPath
}
