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

# 释放端口:查找并杀掉占用 PORT 的进程
echo "🧹 准备释放端口 ${PORT} ..."

# 优先使用 ./kill_port.sh(如果存在)
if [ -f "./kill_port.sh" ]; then
  bash ./kill_port.sh "${PORT}"
else
  # 内置兜底逻辑:直接查找占用端口的进程并 kill
  PIDS=""
  # 兼容 lsof (macOS/Linux)
  if command -v lsof >/dev/null 2>&1; then
    PIDS=$(lsof -ti tcp:"${PORT}" 2>/dev/null || true)
  fi

  # 兜底:兼容 ss / fuser
  if [ -z "$PIDS" ] && command -v fuser >/dev/null 2>&1; then
    PIDS=$(fuser "${PORT}/tcp" 2>/dev/null | tr -d ' ' || true)
  fi

  if [ -n "$PIDS" ]; then
    echo "🔍 端口 ${PORT} 被以下进程占用: ${PIDS}"
    for PID in ${PIDS}; do
      # 跳过当前 shell 自身及父进程
      if [ "${PID}" != "$$" ] && [ "${PID}" != "${PPID}" ]; then
        # 取进程名用于日志
        PNAME=$(ps -p "${PID}" -o comm= 2>/dev/null || echo "unknown")
        echo "💀 杀掉进程 ${PID} (${PNAME})"
        kill -9 "${PID}" 2>/dev/null || true
      fi
    done
    # 等待端口真正释放
    sleep 1
    echo "✅ 端口 ${PORT} 已释放"
  else
    echo "ℹ️  端口 ${PORT} 未被占用,无需清理"
  fi
fi

echo "▶️  执行: wails3 task ${TASK}"
exec wails3 task "${TASK}"
