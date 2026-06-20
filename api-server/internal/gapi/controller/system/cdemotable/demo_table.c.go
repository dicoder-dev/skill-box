package cdemotable

import (
	"ginp-api/internal/gapi/dto/comdto"
	"ginp-api/internal/gapi/entity"
	"ginp-api/internal/gapi/service/system/sdemotable"
	"ginp-api/pkg/where"

	"ginp-api/pkg/ginp"
)

func FindByID(c *ginp.ContextPlus, params *comdto.ReqFindById) {
	info, err := sdemotable.Model().FindOneById(params.ID)
	if err != nil {
		c.Fail(err.Error())
		return
	}
	c.SuccessData(info)
}

func Create(c *ginp.ContextPlus, params *entity.DemoTable) {
	//也可以自己创建并传入读写db: tables.NewDemoTable(wdb,rdb)
	info, err := sdemotable.Model().Create(params)
	if err != nil {
		c.Fail("创建失败" + err.Error())
		return
	}
	c.SuccessData(info)
}

func Update(c *ginp.ContextPlus, params *entity.DemoTable) {
	wheres := where.Format(where.OptEqual("id", params.ID))
	//也可以自己创建并传入读写db: tables.NewDemoTable(wdb,rdb)
	err := sdemotable.Model().Update(wheres, params)
	if err != nil {
		c.Fail("修改失败" + err.Error())
		return
	}
	c.Success()
}

func Delete(c *ginp.ContextPlus, params *comdto.ReqDelete) {
	//也可以自己创建并传入读写db: tables.NewDemoTable(wdb,rdb)
	err := sdemotable.Model().DeleteById(params.ID)
	if err != nil {
		c.Fail("删除失败" + err.Error())
		return
	}
	c.Success()
	return
}

func Search(c *ginp.ContextPlus, params *comdto.ReqSearch) {
	if where.Check(params.Wheres) != nil {
		c.Fail(where.Check(params.Wheres).Error())
		return
	}
	//也可以自己创建并传入读写db: tables.NewDemoTable(wdb,rdb)
	list, total, err := sdemotable.Model().FindList(params.Wheres, params.Extra)
	if err != nil {
		c.Fail("查询失败" + err.Error())
		return
	}

	resp := &comdto.RespSearch{
		List:     list,
		Total:    uint(total),
		PageNum:  uint(params.Extra.PageNum),
		PageSize: uint(params.Extra.PageSize),
	}
	c.SuccessData(resp)
}