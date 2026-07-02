#!/bin/bash
# run-wails - 交互式启动 wails3 任务
# 用法: ./run-wails
# 选项:
#   1) dev   -> wails3 task dev
#   2) web   -> wails3 task web
#   3) build -> wails3 task build
# 默认(直接回车) -> 1) dev
# 启动前会先调用 ./kill_port.sh 释放 9245 端口

set -e

PORT=9245

# 进入脚本所在目录,确保相对路径生效
cd "$(dirname "$0")"

echo "🚀 run-wails 启动器"
echo "─────────────────────────────────"
echo "请选择要执行的任务:"
echo "  1) dev   (wails3 task dev)"
echo "  2) web   (wails3 task web)"
echo "  3) build (wails3 task build)"
echo "─────────────────────────────────"
read -r -p "请输入选项 [1/2/3] (默认 1): " CHOICE

# 默认值: 空输入 -> 1
if [ -z "$CHOICE" ]; then
  CHOICE=1
fi

case "$CHOICE" in
  1)
    TASK="dev"
    ;;
  2)
    TASK="web"
    ;;
  3)
    TASK="build"
    ;;
  *)
    echo "❌ 无效选项: $CHOICE (仅支持 1/2/3)"
    exit 1
    ;;
esac

echo "✅ 已选择任务: ${TASK} (wails3 task ${TASK})"

# 先释放端口
if [ -f "./kill_port.sh" ]; then
  echo "🧹 准备释放端口 ${PORT} ..."
  bash ./kill_port.sh "${PORT}"
else
  echo "⚠️  未找到 ./kill_port.sh,跳过端口清理"
fi

echo "▶️  执行: wails3 task ${TASK}"
exec wails3 task "${TASK}"
