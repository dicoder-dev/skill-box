#!/usr/bin/env bash
# macOS Skill Box 升级替身脚本 — 由 desktop/services/updater_helper.go 在 desktop app Quit 后 fork。
#
# 用法:bash helper_darwin.sh <DEST_ZIP_OR_DMG> <TARGET_INSTALL_DIR> <OS> <ARCH> <OLD_VERSION>
# 标准触发时序详见 ssupdater.SpawnOrder。
#
# - DEST  : 已下载的 .zip / .dmg 路径(Downloader.Download 返的本地路径)
# - TARGET: 例如 /Applications/SkillBox.app
# - OLD   : 升级前的版本,作为 SKILLBOX_UPDATER_FROM 写到 env 给新进程

set -e
DEST="$1"
TARGET="$2"
OLD_VERSION="$5"

if [ -z "$DEST" ] || [ -z "$TARGET" ]; then
  echo "usage: helper_darwin.sh <dest> <target_install_dir> <os> <arch> <old_version>" >&2
  exit 64
fi

# 等 2s 让父进程释放 socket / 端口
sleep 2

# 计算备份名(若已存在 .bak 则覆盖,bak 是覆盖式备份,旧的会丢)
BAK="${TARGET}.bak"

cleanup_partial() {
  # 若已解出 .new 失败,把 .bak 还原
  if [ -d "$BAK" ]; then
    rm -rf "$TARGET"
    mv "$BAK" "$TARGET" 2>/dev/null || true
  fi
}
trap cleanup_partial ERR EXIT

# 备份当前
if [ -e "$TARGET" ]; then
  rm -rf "$BAK"
  mv "$TARGET" "$BAK"
fi

# 解压或 mount dmg(MVP 阶段只支持 zip;dmg 留待后续接 hdiutil)
case "$DEST" in
  *.zip)
    UNZIP_DIR="$(mktemp -d)"
    unzip -q -o "$DEST" -d "$UNZIP_DIR"
    # 找 zip 里的 .app(假设只有一个)
    NEW_APP="$(find "$UNZIP_DIR" -maxdepth 4 -name '*.app' -type d | head -n 1)"
    if [ -z "$NEW_APP" ]; then
      echo "mac helper: no .app found in zip $DEST" >&2
      exit 70
    fi
    mv "$NEW_APP" "$TARGET"
    rm -rf "$UNZIP_DIR"
    ;;
  *.dmg)
    MOUNT_DIR="$(mktemp -d)"
    hdiutil attach -nobrowse -readonly -mountpoint "$MOUNT_DIR" "$DEST" >/dev/null
    NEW_APP="$(find "$MOUNT_DIR" -maxdepth 4 -name '*.app' -type d | head -n 1)"
    if [ -z "$NEW_APP" ]; then
      hdiutil detach "$(echo "$DEST" | sed 's|.*/||')" >/dev/null 2>&1 || true
      echo "mac helper: no .app in dmg" >&2
      exit 70
    fi
    cp -R "$NEW_APP" "$TARGET"
    hdiutil detach "$MOUNT_DIR" >/dev/null 2>&1 || true
    rm -rf "$MOUNT_DIR"
    ;;
  *)
    echo "mac helper: unsupported dest extension: $DEST" >&2
    exit 65
    ;;
esac

# 给主二进制可执行权限(mac 一般保留 quarantine)
chmod +x "$TARGET/Contents/MacOS/skill-box" 2>/dev/null || true

# 一切完成,清 .bak
if [ -e "$BAK" ]; then
  rm -rf "$BAK"
fi
trap - ERR EXIT

# 拉起新版本:用 env SKILLBOX_UPDATER_FROM 让 startupAsync 知道"我是被升级拉起的"
SKILLBOX_UPDATER_FROM="$OLD_VERSION" nohup open -a "$TARGET" >/dev/null 2>&1 &
exit 0
