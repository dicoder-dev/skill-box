#!/bin/bash
# start-skillbox--dev.sh — 【DEV ONLY】开发者本地 debug 用,不是用户解法
#
# ★ 用户解法不是这个脚本
# dmg 分发给终端用户后,「双击闪退」的解法是 Apple GUI 路径:
#   系统设置 → 隐私与安全性 → 滚到底 → 点「仍要打开」→ 再双击
# 这是 Apple 给非 Developer ID 应用的唯一免费 fallback。
#
# ★ 这个脚本是干什么的
# 开发期 / 自测时,想绕过 GUI 手动点「仍要打开」,用 launchctl asuser 直派发
# 拉起 binary(走 launchd 不走 LaunchServices,绕开 amfi Code=-423)。
#
# 位置刻意放在 scripts/dev/ 而不是 scripts/,文件名带 --dev 后缀,
# 防止以后误以为是用户面向的功能。
#
# 用法:
#   bash scripts/dev/start-skillbox--dev.sh
#   bash scripts/dev/start-skillbox--dev.sh --detach   # nohup + disown
#
# 注意:
#   - 不进 dmg(用户原话:不想要 install.sh)
#   - 用户文档(docs/project/desktop/dmg-分发说明.md)不引用本脚本
#   - 想彻底「双击即开」必须 Apple Developer ID + notarize($99/年)
set -euo pipefail

APP_PATH="/Applications/Skill-Box.app"
BIN_PATH="${APP_PATH}/Contents/MacOS/Skill-Box"
LOG_PATH="/tmp/skillbox.log"
DETACH=0

# ---- 参数 ----
while [ $# -gt 0 ]; do
  case "$1" in
    --detach) DETACH=1; shift ;;
    -h|--help) sed -n '2,20p' "$0"; exit 0 ;;
    *) echo "❌ 未知参数: $1"; exit 1 ;;
  esac
done

# ---- 守卫 ----
if [ "$(uname -s)" != "Darwin" ]; then
  echo "❌ 仅 macOS 支持,当前: $(uname -s)"
  exit 1
fi

if [ ! -x "$BIN_PATH" ]; then
  echo "❌ 找不到 binary: $BIN_PATH"
  echo "   请先把 Skill-Box.app 拖到 /Applications"
  exit 1
fi

# ---- 如果已经在跑,提示并退出 ----
if lsof -nP -iTCP:8082 -sTCP:LISTEN 2>/dev/null | grep -q "$BIN_PATH\|Skill-Box"; then
  echo "⚠️  8082 已经被 Skill-Box 占用,先 kill 旧进程:"
  PIDS=$(lsof -nP -iTCP:8082 -sTCP:LISTEN -t 2>/dev/null)
  echo "   PIDs: $PIDS"
  echo "   跑: kill $PIDS   再重跑本脚本"
  exit 1
fi

# ---- 拉起 ----
echo "🚀 launchctl asuser 拉起 $BIN_PATH"
echo "   日志: $LOG_PATH"
if [ "$DETACH" -eq 1 ]; then
  nohup launchctl asuser "$(id -u)" "$BIN_PATH" > "$LOG_PATH" 2>&1 &
  disown
else
  launchctl asuser "$(id -u)" "$BIN_PATH" > "$LOG_PATH" 2>&1 &
fi

# ---- 等 3 秒看进程 / 端口 ----
sleep 3
if lsof -nP -iTCP:8082 -sTCP:LISTEN 2>/dev/null | grep -q Skill-Box; then
  echo "✅ 8082 LISTEN — Skill-Box 已起来"
  echo "   浏览器开 http://127.0.0.1:8082 看前端"
  echo "   实时日志: tail -f $LOG_PATH"
else
  echo "❌ 没起来,看日志:"
  tail -30 "$LOG_PATH" 2>&1 || true
  exit 1
fi