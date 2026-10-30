package entity

import (
	"time"

	"ginp-api/internal/gapi/typ"
	"ginp-api/internal/gen"
)

const tableNameTool = "tools"

// Tool 单个 AI 编程工具的元数据(前端可编辑)。
//
// 2026-06-30 二改:此表替代 internal/skilladapter/toolspecs/specs/*.yaml,
// 工具元数据从"编译期内嵌"变成"运行时 DB 存" + "前端可改"。
//
// 关键约束:
//   - tool_id 全局唯一(uniqueIndex),业务上不可改
//   - is_system=true 的行(seed 出的 9 个默认工具):tool_id / is_system 不可改,
//     整行不可删;display_name / mdi_icon / maturity / note / enabled / paths 可改
//   - is_system=false 的行(用户新建):用户自由改 / 删
//   - 路径细节见 e_tool_path(子表,一对多)
type Tool struct {
	ID          uint      `gorm:"primaryKey;autoIncrement" json:"id,omitempty"`
	ToolID      string    `gorm:"type:varchar(32);column:tool_id;uniqueIndex;comment:canonical 工具 ID(全局唯一)" json:"tool_id,omitempty"`
	DisplayName string    `gorm:"type:varchar(64);column:display_name;comment:UI 显示名" json:"display_name,omitempty"`
	MdiIcon     string    `gorm:"type:varchar(64);column:mdi_icon;comment:前端 mdi 图标(mdi:xxx);与 icon_file 二选一" json:"mdi_icon,omitempty"`
	// IconFile 用户上传/前端嵌入的自定义图标文件名(纯 basename,如 claude.png)。
	// 空字符串 = 用 mdi_icon。为避免路径穿越,前端只接受 basename,
	// 后端读文件时拼到 ~/.skill-box/tool-icons/<basename> 兜底校验。
	IconFile    string    `gorm:"type:varchar(128);column:icon_file;comment:自定义图标文件名(basename),存于 ~/.skill-box/tool-icons/;空则用 mdi_icon" json:"icon_file,omitempty"`
	Maturity    string    `gorm:"type:varchar(16);column:maturity;comment:stable|experimental|deprecated" json:"maturity,omitempty"`
	Note        string    `gorm:"type:text;column:note;comment:自由文本说明,前端不展示" json:"note,omitempty"`
	IsSystem    bool      `gorm:"column:is_system;index;comment:系统工具:tool_id 不可改,行不可删" json:"is_system,omitempty"`
	Enabled     bool      `gorm:"column:enabled;index;comment:工具启用开关;false 时 adapter 不注册" json:"enabled,omitempty"`
	SortOrder   int       `gorm:"column:sort_order;comment:列表展示顺序,数字越小越前" json:"sort_order,omitempty"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at,omitempty"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updated_at,omitempty"`
}

var _ typ.IEntity = (*Tool)(nil)

func (Tool) GenConfig() *gen.EntityConfig {
	return &gen.EntityConfig{
		TableName: tableNameTool,
	}
}

func (Tool) GenEnumOptions() []typ.EntityEnumOption {
	return nil
}

func (Tool) TableName() string {
	return tableNameTool
}
