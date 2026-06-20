#!/bin/bash

# 清理Go模块缓存
echo "清理Go模块缓存..."
go clean -modcache

# 删除vendor目录以确保获取最新代码
echo "删除vendor目录..."
rm -rf vendor/github.com/DicoderCn/ginp

# 更新指定ginp包到最新版本
echo "更新ginp包..."
GOPROXY=direct go get -u -v github.com/DicoderCn/ginp

# 执行vendor操作
echo "执行go mod vendor..."
go mod vendor

echo "依赖更新完成"