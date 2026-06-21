package entity

import (
    "time"
    "ginp-api/internal/gapi/typ"
    "ginp-api/internal/gen"
)

const tableNameAuditLog = "audit_logs"

// AuditLog 见 docs/project/需求规划.md 第 6 节。
type AuditLog struct {
    ID             uint       `gorm:"primaryKey;autoIncrement" json:"id,omitempty"`
    Actor          string     `gorm:"type:varchar(64);column:actor;index;comment:操作者" json:"actor,omitempty"`
    Action         string     `gorm:"type:varchar(32);column:action;index;comment:操作类型" json:"action,omitempty"`
    TargetType     string     `gorm:"type:varchar(32);column:target_type;index;comment:目标类型" json:"target_type,omitempty"`
    TargetID       uint       `gorm:"column:target_id;comment:目标 ID" json:"target_id,omitempty"`
    Payload        string     `gorm:"type:text;column:payload;comment:上下文 JSON" json:"payload,omitempty"`
    CreatedAt      time.Time  `gorm:"autoCreateTime" json:"created_at,omitempty"`
}

var _ typ.IEntity = (*AuditLog)(nil)

func (AuditLog) GenConfig() *gen.EntityConfig {
	return &gen.EntityConfig{
		TableName: tableNameAuditLog,
	}
}

func (AuditLog) GenEnumOptions() []typ.EntityEnumOption {
	return nil
}

func (AuditLog) TableName() string {
	return tableNameAuditLog
}
