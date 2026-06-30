package mtool

// 字段常量(跟 entity.Tool 字段一一对应,业务层用 NewField / NewWhWhere 引用避免散落字面量)。
const (
	FieldID          = "id"
	FieldToolID      = "tool_id"
	FieldDisplayName = "display_name"
	FieldMdiIcon     = "mdi_icon"
	FieldMaturity    = "maturity"
	FieldNote        = "note"
	FieldIsSystem    = "is_system"
	FieldEnabled     = "enabled"
	FieldSortOrder   = "sort_order"
	FieldCreatedAt   = "created_at"
	FieldUpdatedAt   = "updated_at"
)