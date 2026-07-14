#!/bin/bash
# build-dmg.sh — Skill Box macOS 可拖拽 DMG 一键打包
# 零第三方依赖,只用 macOS 系统自带 hdiutil + osascript。
# 用法:
#   ./scripts/build-dmg.sh                         # 默认 universal,产物 bin/skill-box.dmg
#   ./scripts/build-dmg.sh --arch arm64            # 只打 arm64
#   ./scripts/build-dmg.sh --skip-build            # 假设 .app 已生成,只重打 dmg(快速迭代布局)
#   ./scripts/build-dmg.sh --output dist/sb.dmg    # 自定义输出路径
#   ./scripts/build-dmg.sh --volname "Skill Box"   # 自定义卷标
#   ./scripts/build-dmg.sh --app-name skill-box    # 自定义 app 名(默认 skill-box)
#
# 前置:
#   - macOS(本脚本只支持 Darwin)
#   - 已安装 wails3 CLI(用于 --arch 时调用 darwin:package[:universal])
#   - bin/<APP_NAME>.app 必须存在(可由本脚本自动触发 build,也可 --skip-build 跳过)
set -euo pipefail

# ---- 默认值 ----
ARCH="universal"
OUTPUT="bin/skill-box.dmg"
SKIP_BUILD=0
VOLNAME="Skill Box"
APP_NAME="skill-box"

# 临时状态(trap 清理用)
DMG_STAGING=""
MOUNT_POINT=""

# ---- 用法 ----
usage() {
  sed -n '2,12p' "$0"
  exit 1
}

# ---- 参数解析 ----
while [ $# -gt 0 ]; do
  case "$1" in
    --arch)        ARCH="$2"; shift 2 ;;
    --output)      OUTPUT="$2"; shift 2 ;;
    --skip-build)  SKIP_BUILD=1; shift ;;
    --volname)     VOLNAME="$2"; shift 2 ;;
    --app-name)    APP_NAME="$2"; shift 2 ;;
    -h|--help)     usage ;;
    *) echo "❌ 未知参数: $1"; usage ;;
  esac
done

APP_PATH="bin/${APP_NAME}.app"

# ---- 平台守卫 ----
if [ "$(uname -s)" != "Darwin" ]; then
  echo "❌ DMG 打包仅支持 macOS,当前平台: $(uname -s)"
  exit 1
fi

# ---- 工具自检 ----
for tool in hdiutil osascript; do
  command -v "$tool" >/dev/null 2>&1 || { echo "❌ 缺少系统工具: $tool"; exit 1; }
done

# ---- 失败清理 trap ----
# trap 在任何退出路径(成功 / 失败 / Ctrl-C)都会被调用,通过把 DMG_STAGING / MOUNT_POINT
# 置空来精确控制"已移交"的资源不被误删。
cleanup() {
  local rc=$?
  # 1. 如果挂载点还在,先 detach
  if [ -n "$MOUNT_POINT" ] && [ -d "$MOUNT_POINT" ]; then
    echo "🧹 卸载挂载点: $MOUNT_POINT"
    hdiutil detach "$MOUNT_POINT" 2>/dev/null || \
      hdiutil detach -force "$MOUNT_POINT" 2>/dev/null || true
    # 删除临时挂载点目录
    rmdir "$MOUNT_POINT" 2>/dev/null || true
  fi
  # 2. 删 staging dmg(只有在第 7 步 convert 成功后才置空,避免误删最终产物)
  if [ -n "$DMG_STAGING" ] && [ -f "$DMG_STAGING" ]; then
    echo "🧹 清理 staging: $DMG_STAGING"
    rm -f "$DMG_STAGING"
  fi
  exit $rc
}
trap cleanup EXIT INT TERM

# ---- 步骤 1:可选触发 build ----
echo "=== 1. BUILD .app (arch=$ARCH) ==="
if [ "$SKIP_BUILD" -eq 0 ]; then
  case "$ARCH" in
    arm64|amd64) wails3 task darwin:package ;;
    universal)   wails3 task darwin:package:universal ;;
    *) echo "❌ --arch 只支持 arm64/amd64/universal,收到: $ARCH"; exit 1 ;;
  esac
else
  echo "⏭️  --skip-build,假设 ${APP_PATH} 已存在"
fi

[ -d "$APP_PATH" ] || { echo "❌ 找不到 ${APP_PATH},请先 build 或检查 --app-name"; exit 1; }
echo "  → ${APP_PATH} ($(du -sh "$APP_PATH" | cut -f1))"

# ---- 步骤 2:创建可写 staging DMG ----
echo "=== 2. CREATE staging DMG ==="
mkdir -p "$(dirname "$OUTPUT")"
DMG_STAGING="${OUTPUT}.staging.dmg"
# 用 UDRW(可读写)创建 staging,因为我们要:
#   1) 挂载后写 .DS_Store(布局)
#   2) ln -s /Applications 建软链
#   3) 最终 convert 成 UDRO staging(布局固化)+ UDZO 压缩(分发给用户)
# UDRO 不能 -readwrite 挂载(macOS 报"操作不被允许"),所以中间必须经过 UDRW。
hdiutil create \
  -srcfolder "$APP_PATH" \
  -format UDRW \
  -volname "$VOLNAME" \
  -fs HFS+ \
  -size 200m \
  -ov \
  "$DMG_STAGING"
echo "  → staging: $DMG_STAGING ($(du -h "$DMG_STAGING" | cut -f1))"

# ---- 步骤 3:挂载 staging ----
echo "=== 3. MOUNT staging ==="
# macOS 不允许非交互进程把可写 dmg 挂到 /Volumes/<volname> 路径下(会报
# "操作不被允许"),所以我们显式指定一个临时挂载点,在 /tmp 下,不影响 /Volumes。
MOUNT_BASE="/tmp/dmg-mount.$$"
mkdir -p "$MOUNT_BASE"
ATTACH_OUTPUT=$(hdiutil attach -readwrite -noverify -nobrowse -mountpoint "$MOUNT_BASE" "$DMG_STAGING" 2>&1)
MOUNT_POINT="$MOUNT_BASE"
[ -d "$MOUNT_POINT" ] || { echo "❌ 挂载失败: $ATTACH_OUTPUT"; exit 1; }
echo "  → 挂载点: $MOUNT_POINT"

# ---- 步骤 4:准备布局(建 /Applications 软链)----
echo "=== 4. PREPARE layout ==="
ln -sfn /Applications "$MOUNT_POINT/Applications"
echo "  → 已建 /Applications 软链"

# ---- 步骤 5:AppleScript 写 Finder 布局 ----
echo "=== 5. WRITE Finder layout (osascript) ==="
# 关键防御点:
#   - delay 1 给 Finder 时间挂载就绪(Apple Silicon M1/M2 挂载延迟比 Intel 高)
#   - 必须先 set current view to icon view,再设 bounds,否则坐标错乱
#   - 必须显式 set arrangement to not arranged,默认网格会把图标 snap 到错误坐标
#   - close + delay 1 让 Finder 写 .DS_Store 到 dmg 根
# 窗口 600x400,app 图标坐标 (170,190),Applications 软链坐标 (410,190)。
#
# 重要:不走 `tell disk "Skill Box"`(会报 -1728),改用 `POSIX file` 直接定位挂载点
# 卷标 + Finder disk 对象在 -mountpoint /tmp/... 时未必注册,POSIX path 最稳。
MOUNT_OSA=$(echo "$MOUNT_POINT" | sed 's/ /\\ /g')
osascript <<EOF
tell application "Finder"
  delay 1
  set theFolder to POSIX file "$MOUNT_OSA" as alias
  open theFolder

  set winOpts to container window of theFolder

  -- 切到 icon view + 关掉 toolbar/sidebar/statusbar
  set current view of winOpts to icon view
  set toolbar visible of winOpts to false
  set statusbar visible of winOpts to false
  set sidebar width of winOpts to 0

  -- 窗口位置 + 大小
  set position of winOpts to {100, 100}
  set bounds of winOpts to {100, 100, 700, 500}

  -- icon view options:必须显式设,否则用默认网格坐标全错
  set icon size of icon view options of winOpts to 96
  set text size of icon view options of winOpts to 12
  set arrangement of icon view options of winOpts to not arranged
  set background color of icon view options of winOpts to {65535, 65535, 65535}

  -- 两图标坐标
  set position of item "${APP_NAME}.app" of winOpts to {170, 190}
  set position of item "Applications" of winOpts to {410, 190}

  -- 关键:close 触发 Finder flush .DS_Store 到 dmg 根
  close winOpts
  delay 1
end tell
EOF
echo "  → 布局已写入"

# ---- 步骤 6:卸载 ----
echo "=== 6. DETACH ==="
hdiutil detach "$MOUNT_POINT"
MOUNT_POINT=""   # 标记已移交,trap 不再处理

# ---- 步骤 7:UDRO → UDZO 压缩 ----
echo "=== 7. CONVERT → final UDZO ==="
# UDZO = compressed read-only,体积比 UDRO 小 30%~50%。
# 转完后 staging 使命完成,trap 会删它(此时 DMG_STAGING 还没置空)。
hdiutil convert "$DMG_STAGING" -format UDZO -o "$OUTPUT"
DMG_STAGING=""   # 标记已移交,trap 不再删
echo "  → 最终: $OUTPUT"

# ---- 步骤 8:校验 ----
echo "=== 8. VERIFY ==="
hdiutil verify "$OUTPUT" || { echo "❌ dmg 校验失败"; exit 1; }
ls -lh "$OUTPUT"

echo ""
echo "✅ DMG 已生成: $OUTPUT"
echo "   双击后可拖拽 skill-box.app 到 /Applications 安装"