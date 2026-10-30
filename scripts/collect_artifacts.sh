#!/usr/bin/env bash
# scripts/collect_artifacts.sh - 把各平台构建产物整理成发布命名约定的文件。
#
# 用法: bash scripts/collect_artifacts.sh <target> <version>
#   target: darwin | windows | linux
#
# 命名约定(manifest 与 release 资产都按这个来):
#   skill-box_<version>_<os>_<arch>.<ext>
#     darwin  → skill-box_<version>_darwin_universal.zip   (内含 Skill-Box.app,universal 二进制)
#     windows → skill-box_<version>_windows_amd64.zip      (内含 skill-box.exe,替身脚本只认 zip)
#     linux   → skill-box_<version>_linux_amd64.AppImage   (AppImage 本身可执行)
#   仅供手动下载(不进 manifest 的 assets[]):
#     skill-box_<version>_windows_amd64-installer.exe      (nsis 安装包,替身脚本不支持 exe)
#
# 产物统一落到 bin/release/(被 .gitignore 覆盖,不入库)。
#
# 这个脚本被两处复用:
#   - .github/workflows/release.yml (CI 里三平台各跑一次)
#   - scripts/release.sh MODE=build (本地构建发版)

set -euo pipefail

cd "$(cd "$(dirname "$0")/.." && pwd)"

TARGET="${1:-}"
VERSION="${2:-}"
APP_NAME="${APP_NAME:-Skill-Box}"
BIN_DIR="${BIN_DIR:-bin}"
REL_DIR="$BIN_DIR/release"

log() { echo "[collect $TARGET] $*" >&2; }
err() { echo "[collect ERR] $*" >&2; exit 1; }

[ -n "$TARGET" ] || err "用法: bash scripts/collect_artifacts.sh <darwin|windows|linux> <version>"
[ -n "$VERSION" ] || err "用法: bash scripts/collect_artifacts.sh <darwin|windows|linux> <version>"

mkdir -p "$REL_DIR"

# make_zip <输出文件> <基准目录> [要打包的条目...]
# 优先用 zip;Windows runner 上没有 zip 时退回 7z。
make_zip() {
  local out="$1" base_dir="$2"
  shift 2
  rm -f "$out"
  if command -v zip >/dev/null 2>&1; then
    ( cd "$base_dir" && zip -qyry "$out" "$@" -x '*.DS_Store' -x '__MACOSX/*' )
  elif command -v 7z >/dev/null 2>&1; then
    ( cd "$base_dir" && 7z a -tzip "$out" "$@" >/dev/null )
  else
    err "既没有 zip 也没有 7z,无法打包 $out"
  fi
}

case "$TARGET" in
  darwin)
    SRC="$BIN_DIR/${APP_NAME}.app"
    [ -d "$SRC" ] || err "找不到 $SRC(先跑 wails3 task darwin:package:universal)"
    OUT="$REL_DIR/skill-box_${VERSION}_darwin_universal.zip"
    # 只打包 .app bundle,不要把 bin/ 下别的东西(比如另一个 arch 的产物)带进去
    make_zip "$OUT" "$BIN_DIR" "${APP_NAME}.app"
    log "产出 $(basename "$OUT")"
    ;;
  windows)
    SRC="$BIN_DIR/${APP_NAME}.exe"
    [ -f "$SRC" ] || err "找不到 $SRC(先跑 wails3 task windows:package)"
    STAGE="$REL_DIR/.win-stage"
    rm -rf "$STAGE" && mkdir -p "$STAGE"
    # helper_windows.ps1 找的是 skill-box.exe,这里统一改名(Windows 文件名不区分大小写)
    cp "$SRC" "$STAGE/skill-box.exe"
    cp "$BIN_DIR/WebView2Loader.dll" "$STAGE/" 2>/dev/null || true
    OUT="$REL_DIR/skill-box_${VERSION}_windows_amd64.zip"
    make_zip "$OUT" "$STAGE" .
    rm -rf "$STAGE"
    log "产出 $(basename "$OUT")"
    # nsis 安装包:仅供手动下载(替身脚本不支持 exe)
    for inst in "$BIN_DIR"/${APP_NAME}-*-installer.exe "$BIN_DIR"/${APP_NAME}-installer.exe; do
      [ -f "$inst" ] || continue
      cp "$inst" "$REL_DIR/skill-box_${VERSION}_windows_amd64-installer.exe"
      log "产出 $(basename "$REL_DIR/skill-box_${VERSION}_windows_amd64-installer.exe")(仅手动下载)"
    done
    ;;
  linux)
    FOUND=0
    for f in "$BIN_DIR"/${APP_NAME}*.AppImage "$BIN_DIR"/${APP_NAME}*.appimage; do
      [ -f "$f" ] || continue
      base="$(basename "$f")"
      case "$base" in
        *arm64*) arch=arm64 ;;
        *) arch=amd64 ;;
      esac
      out="$REL_DIR/skill-box_${VERSION}_linux_${arch}.AppImage"
      cp "$f" "$out"
      FOUND=1
      log "产出 $(basename "$out")"
    done
    [ "$FOUND" = "1" ] || err "找不到 AppImage(先跑 wails3 task linux:package)"
    ;;
  *)
    err "未知 target: $TARGET(支持 darwin / windows / linux)"
    ;;
esac

log "完成,产物在 $REL_DIR"
