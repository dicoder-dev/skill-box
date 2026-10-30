package entity

import (
	"ginp-api/internal/gapi/typ"
	"ginp-api/internal/gen"
)

// tableNameAgentPath DB 表名(保留历史值 "tool_paths",理由同 tableNameAgent)。
const tableNameAgentPath = "tool_paths"

// AgentPath 单个 Agent 的一条扫描/写盘路径。
//
// 2026-07-04 改:每个 Agent 的每个 (scope, category) 最多只允许 1 条 path,
// DB 层加 uniqueIndex(tool_id, scope, category) 兜底。前端编辑弹窗也
// 同步改成 4 格固定布局(global/project × user/system 各 1 条)。
// 删 tools 行时 service 层事务里清掉本表对应行(避免悬空路径)。
//
// 关键约束:
//   - (tool_id, scope, category) 三元组唯一;同 (scope, category) 不允许多条
//   - path 字段保留 ~/ 形式(运行时由 BaseAdapter 展开),不展开为绝对路径;
//     理由:不同用户(系统)可能共享同一个 DB 快照,但 home 不同
//   - path_order 在单 path 模型下退化为恒为 0 的顺序字段,保留兼容老行
type AgentPath struct {
	ID        uint   `gorm:"primaryKey;autoIncrement" json:"id,omitempty"`
	AgentID   uint   `gorm:"column:tool_id;uniqueIndex:uniq_tool_scope_category;comment:所属 Agent(逻辑外键,删 agent 级联)" json:"agent_id,omitempty"`
	Scope     string `gorm:"type:varchar(16);column:scope;uniqueIndex:uniq_tool_scope_category;comment:global|project" json:"scope,omitempty"`
	Category  string `gorm:"type:varchar(16);column:category;uniqueIndex:uniq_tool_scope_category;comment:user|system" json:"category,omitempty"`
	Path      string `gorm:"type:varchar(512);column:path;comment:绝对路径或相对路径(含 ~/)" json:"path,omitempty"`
	PathOrder int    `gorm:"column:path_order;comment:同一 (scope,category) 内的顺序(单 path 模型下恒为 0)" json:"path_order,omitempty"`
}

var _ typ.IEntity = (*AgentPath)(nil)

func (AgentPath) GenConfig() *gen.EntityConfig {
	return &gen.EntityConfig{
		TableName: tableNameAgentPath,
	}
}

func (AgentPath) GenEnumOptions() []typ.EntityEnumOption {
	return nil
}

func (AgentPath) TableName() string {
	return tableNameAgentPath
}
