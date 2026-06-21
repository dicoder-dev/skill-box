package entity

import (
    "ginp-api/internal/gapi/typ"
    "ginp-api/internal/gen"
)

const tableNameOnboardingState = "onboarding_states"

// OnboardingState 见 docs/project/需求规划.md 第 6 节。
type OnboardingState struct {
    ID             uint       `gorm:"primaryKey;autoIncrement" json:"id,omitempty"`
    Key            string     `gorm:"type:varchar(64);column:key;comment:键名;uniqueIndex:idx_onboarding_key" json:"key,omitempty"`
    Value          string     `gorm:"type:text;column:value;comment:值" json:"value,omitempty"`
}

var _ typ.IEntity = (*OnboardingState)(nil)

func (OnboardingState) GenConfig() *gen.EntityConfig {
	return &gen.EntityConfig{
		TableName: tableNameOnboardingState,
	}
}

func (OnboardingState) GenEnumOptions() []typ.EntityEnumOption {
	return nil
}

func (OnboardingState) TableName() string {
	return tableNameOnboardingState
}
