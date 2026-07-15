#!/bin/bash
# build-dmg-both.sh — 一键产出 Skill-Box-arm64.dmg + Skill-Box-amd64.dmg
# 串行跑,避免 dmg-arm64 / dmg-amd64 共享 bin/Skill-Box.app task DAG 调度问题
# 用法:
#   ./scripts/build-dmg-both.sh
# 等价于:
#   wails3 task darwin:dmg-arm64 && wails3 task darwin:dmg-amd64
# 但 .app 在 task 间共享,用 wails3 task 链式跑第二个时 codesign 会失败
# (拿到第一个 raw cp 后的 .app,binary 仍是 arm64 + .cstemp 残留)
set -euo pipefail

BIN_DIR="bin"
APP_NAME="Skill-Box"

# PATH 兜底(wails3 在 $HOME/go/bin,但 task 子进程不一定继承)
case ":$PATH:" in
  *":$HOME/go/bin:"*) ;;
  *) export PATH="$HOME/go/bin:$PATH" ;;
esac

echo "=== 1/2 dmg-arm64 ==="
wails3 task darwin:dmg-arm64
echo ""
echo "=== 2/2 dmg-amd64 ==="
wails3 task darwin:dmg-amd64
echo ""

echo "✅ 两份 dmg 生成完成:"
ls -lh "${BIN_DIR}/${APP_NAME}-arm64.dmg" "${BIN_DIR}/${APP_NAME}-amd64.dmg" 2>&1 | head -3
