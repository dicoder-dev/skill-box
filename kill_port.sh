#!/bin/bash
# kill_port.sh - 杀掉占用指定端口的进程
# 用法: ./kill_port.sh [port]  默认 9245

PORT="${1:-9245}"

echo "🔍 正在查找占用端口 ${PORT} 的进程..."

# 查找占用端口的进程 PID（兼容 macOS 和 Linux）
PIDS=$(lsof -ti tcp:${PORT} 2>/dev/null)

if [ -z "$PIDS" ]; then
  echo "✅ 端口 ${PORT} 空闲，没有进程占用"
  exit 0
fi

echo "🎯 找到以下进程占用端口 ${PORT}:"
for PID in $PIDS; do
  # 尝试获取进程信息
  if [ -d "/proc/${PID}" ]; then
    # Linux
    CMD=$(cat /proc/${PID}/comm 2>/dev/null || echo "未知")
  else
    # macOS
    CMD=$(ps -p ${PID} -o comm= 2>/dev/null || echo "未知")
  fi
  echo "   PID=${PID}  CMD=${CMD}"
done

echo "💀 正在 kill 这些进程..."
for PID in $PIDS; do
  kill -9 ${PID} 2>/dev/null && echo "   ✅ killed PID=${PID}" || echo "   ❌ kill PID=${PID} 失败"
done

# 再次验证
sleep 1
REMAINING=$(lsof -ti tcp:${PORT} 2>/dev/null)
if [ -z "$REMAINING" ]; then
  echo "🎉 端口 ${PORT} 已释放"
else
  echo "⚠️  仍有进程占用: ${REMAINING}"
  exit 1
fi
