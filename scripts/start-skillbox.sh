#!/bin/bash
# start-skillbox.sh — macOS 26 Tahoe 上「双击 .app 闪退」的临时拉起脚本
#
# 背景:
#   dmg 是 ad-hoc 签名的 Skill-Box.app。macOS 26 Tahoe 的 amfi 对
#   LaunchServices / amfi 派发链上的 ad-hoc binary 报 Code=-423,
#   open / Finder 双击进程会被静默杀掉,没有任何 GUI 提示。
#
# 解法(实测 2026-07-15 通过):
#   launchctl asuser <uid> <binary> 走 launchd 直派发(launchedByLS=0),
#   不触发 amfi 校验,binary 能正常起来。
#
# 用法:
#   bash scripts/start-skillbox.sh            # 默认 /Applications/Skill-Box.app
#   bash scripts/start-skillbox.sh --detach   # 用 nohup + disown 脱离当前 shell
#
# 注意:
#   - 不要把这脚本塞进 dmg(用户原话:不想要 install.sh)
#   - 想彻底"双击即开"必须 Apple Developer ID + notarize($99/年),
#     docs/project/desktop/dmg-分发说明.md 末尾有完整配置
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