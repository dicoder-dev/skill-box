package router

//显式导入：来确保它的 init 函数被调用
import (
	_ "ginp-api/internal/gapi/controller/system/cdemotable"
	_ "ginp-api/internal/gapi/controller/system/cindex"
	_ "ginp-api/internal/gapi/controller/user/cuser"
	//{{placeholder_router_import}}//
	// 上面的占位符请不要动动，否则会导致生成工具无法自动替换
	//Please do not move the placeholders above, otherwise it will cause the generation tool to fail to replace them automatically
)
