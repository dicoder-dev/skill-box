package entity

import "ginp-api/internal/gapi/typ"

func (User) GenEnumOptions() []typ.EntityEnumOption {
	return []typ.EntityEnumOption{
		{
			FieldName: "status",
			Options: map[string]string{
				"1": "正常",
				"2": "禁用",
				"3": "注销",
			},
		},
	}
}
