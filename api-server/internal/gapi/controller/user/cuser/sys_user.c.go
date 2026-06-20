package cuser

import (
	"ginp-api/internal/gapi/dto/comdto"
	"ginp-api/internal/gapi/entity"
	"ginp-api/internal/gapi/service/user/suser"

	"ginp-api/pkg/where"

	"ginp-api/pkg/ginp"
)

func FindByID(ctx *ginp.ContextPlus) {
	var params *comdto.ReqFindById
	if err := ctx.ShouldBindJSON(&params); err != nil {
		ctx.Fail("请求参数有误" + err.Error())
		return
	}
	info, err := suser.Model().FindOneById(params.ID)
	if err != nil {
		ctx.Fail(err.Error())
		return
	}
	ctx.SuccessData(info)
}

func Create(ctx *ginp.ContextPlus) {
	var params *entity.User
	if err := ctx.ShouldBindJSON(&params); err != nil {
		ctx.Fail("请求参数有误" + err.Error())
		return
	}
	//也可以自己创建并传入读写db: tables.NewUser(wdb,rdb)
	info, err := suser.Model().Create(params)
	if err != nil {
		ctx.Fail("创建失败" + err.Error())
		return
	}
	ctx.SuccessData(info)
}

func Update(ctx *ginp.ContextPlus) {
	var params *entity.User
	if err := ctx.ShouldBindJSON(&params); err != nil {
		ctx.Fail("请求参数有误" + err.Error())
		return
	}
	wheres := where.Format(where.OptEqual("id", params.ID))
	//也可以自己创建并传入读写db: tables.NewUser(wdb,rdb)
	err := suser.Model().Update(wheres, params)
	if err != nil {
		ctx.Fail("修改失败" + err.Error())
		return
	}
	ctx.Success()
}

func Delete(ctx *ginp.ContextPlus) {
	var params *comdto.ReqDelete
	if err := ctx.ShouldBindJSON(&params); err != nil {
		ctx.Fail("请求参数有误" + err.Error())
		return
	}

	//也可以自己创建并传入读写db: tables.NewUser(wdb,rdb)
	err := suser.Model().DeleteById(params.ID)
	if err != nil {
		ctx.Fail("删除失败" + err.Error())
		return
	}
	ctx.Success()
	return
}

func Search(ctx *ginp.ContextPlus) {
	var params *comdto.ReqSearch
	if err := ctx.ShouldBindJSON(&params); err != nil {
		ctx.Fail("请求参数有误" + err.Error())
		return
	}
	if where.Check(params.Wheres) != nil {
		ctx.Fail(where.Check(params.Wheres).Error())
		return
	}
	//也可以自己创建并传入读写db: tables.NewUser(wdb,rdb)
	list, total, err := suser.Model().FindList(params.Wheres, params.Extra)
	if err != nil {
		ctx.Fail("查询失败" + err.Error())
		return
	}

	resp := &comdto.RespSearch{
		List:     list,
		Total:    uint(total),
		PageNum:  uint(params.Extra.PageNum),
		PageSize: uint(params.Extra.PageSize),
	}
	ctx.SuccessData(resp)

}
