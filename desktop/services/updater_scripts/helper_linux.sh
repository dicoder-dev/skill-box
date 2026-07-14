#!/usr/bin/env bash
# Linux Skill Box 升级替身脚本 — AppImage 或裸二进制模式。
#
# 用法:bash helper_linux.sh <DEST> <TARGET_INSTALL_DIR> <OS> <ARCH> <OLD_VERSION>
#
# DEST     : 已下载的 .AppImage / tar.gz / 裸二进制 路径
# TARGET   : ~/.local/bin/skill-box(默认)或留空走 DEST 替换自己
# OLD      : 升级前版本,作为 SKILLBOX_UPDATER_FROM 写到 env 给新进程

set -e
DEST="$1"
TARGET="$2"
OS_VAL="$3"
ARCH_VAL="$4"
OLD_VERSION="$5"

if [ -z "$DEST" ]; then
  echo "usage: helper_linux.sh <dest> [target_install_dir] [os] [arch] [old_version]" >&2
  exit 64
fi

# 等父进程退出
sleep 2

if [ -z "$TARGET" ]; then
  # 自替换模式:DEST 直接替换当前 skill-box 进程,但 fork+wait 已经退出,这里直接 install 自身
  SELF="$(ps -o comm= $$ 2>/dev/null || echo skill-box)"
  TARGET="$SELF"
fi

# 自包含容错
cleanup() {
  if [ -e "${TARGET}.bak" ]; then
    mv -f "${TARGET}.bak" "$TARGET"
  fi
}
trap cleanup ERR EXIT

case "$DEST" in
  *.AppImage|*.appimage)
    # AppImage 模式:DEST 自身是可执行的,install 到目标
    install -m 0755 "$DEST" "${TARGET}.new"
    # 同时保留旧(回滚用)
    if [ -e "$TARGET" ]; then cp -p "$TARGET" "${TARGET}.bak"; fi
    mv -f "${TARGET}.new" "$TARGET"
    ;;
  *.tar.gz|*.tgz)
    WORK="$(mktemp -d)"
    tar -xzf "$DEST" -C "$WORK"
    NEW_BIN="$(find "$WORK" -maxdepth 4 -type f -name 'skill-box' | head -n 1)"
    if [ -z "$NEW_BIN" ]; then
      echo "linux helper: no skill-box binary in $DEST" >&2
      exit 70
    fi
    if [ -e "$TARGET" ]; then cp -p "$TARGET" "${TARGET}.bak"; fi
    install -m 0755 "$NEW_BIN" "${TARGET}.new"
    mv -f "${TARGET}.new" "$TARGET"
    rm -rf "$WORK"
    ;;
  *)
    # 裸二进制模式
    if [ -e "$TARGET" ]; then cp -p "$TARGET" "${TARGET}.bak"; fi
    install -m 0755 "$DEST" "${TARGET}.new"
    mv -f "${TARGET}.new" "$TARGET"
    ;;
esac

# 启动新二进制
if [ -e "${TARGET}.bak" ]; then rm -f "${TARGET}.bak"; fi
trap - ERR EXIT

SKILLBOX_UPDATER_FROM="$OLD_VERSION" SKILLBOX_PLATFORM_OVERRIDE=linux nohup "$TARGET" >/dev/null 2>&1 &
exit 0
