package sdemotable

import (
	"ginp-api/internal/gapi/model/system/mdemotable"
	"ginp-api/internal/db/mysql"
)

var DemoTable *mdemotable.Model

func Model() *mdemotable.Model {
	if DemoTable == nil {
		dbRead := mysql.GetReadDb()
		dbWrite := mysql.GetWriteDb()
		DemoTable = mdemotable.NewModel(dbRead, dbWrite)
	}
	return DemoTable
}
