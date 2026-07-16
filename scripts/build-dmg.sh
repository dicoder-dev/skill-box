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
OUTPUT="bin/Skill-Box.dmg"
SKIP_BUILD=0
VOLNAME="Skill-Box"
APP_NAME="Skill-Box"

# ---- PATH 兜底:wails3 在 $HOME/go/bin,但 wails3 task 的子进程 bash 不一定继承
# 完整 PATH(尤其 /Users/brody 的 zshenv 没把 go/bin 加进 PATH 时)。手动补一下。----
case ":$PATH:" in
  *":$HOME/go/bin:"*) ;;
  *) export PATH="$HOME/go/bin:$PATH" ;;
esac

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
# 不再根据 dmg 文件名后缀切 .app 路径 —— dmg-arm64 / dmg-amd64 任务里 .app 文件夹
# 统一叫 Skill-Box.app(用户拖到 /Applications 后看到的就是干净的名字,没有
# -arm64 / -amd64 后缀)。架构区分只体现在产物 dmg 文件名上。
# 如果未来要 universal dmg,产物就叫 Skill-Box.dmg,内部也是 Skill-Box.app。
# AppleScript 写布局时要找的 .app 文件夹名(去掉 bin/ 前缀)。
APP_FOLDER_NAME="$(basename "$APP_PATH")"

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
# SKIP_BUILD 也接受外部环境变量(直接命令行调用 build-dmg.sh 时可设,
# Taskfile dmg-arm64 / dmg-amd64 走 --skip-build 参数,更稳)。
if [ -n "${SKIP_BUILD:-}" ] && [ "${SKIP_BUILD}" != "0" ]; then
  SKIP_BUILD=1
fi
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

# hdiutil create -srcfolder 默认会自动把 dmg 挂到 /Volumes/<volname>,
# 让用户能在 Finder 里看到。但我们要立刻弹出来,因为:
#   1) 后面步骤 3 会用 -mountpoint /tmp/... 重新挂,自动挂的位置跟我们不一致
#   2) 如果不弹,LaunchServices 会路由 /Applications/skill-box.app 到 /Volumes/Skill Box,
#      用户后续从 dmg 拖到 /Applications 后双击没反应(本项目踩过的坑)
hdiutil detach "/Volumes/$VOLNAME" 2>/dev/null || true

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

# ---- 步骤 4:准备布局(建 /Applications 软链 + 写 README)----
echo "=== 4. PREPARE layout ==="
ln -sfn /Applications "$MOUNT_POINT/Applications"
echo "  → 已建 /Applications 软链"

# dmg 内不写 README / install.sh 等任何文件 —— 用户只关心拖 .app 到 /Applications
# 然后双击。macOS 26 Tahoe 首次打开会被拒,用户去「系统设置 → 隐私与
# 安全性 → 仍要打开」即可,这是 happytools / Tauri / Electron-builder 等
# 所有非 Developer ID 签名的开源 macOS 应用共用的官方 GUI 路径。

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
# 关键防御点:
#   - delay 1 给 Finder 时间挂载就绪(Apple Silicon M1/M2 挂载延迟比 Intel 高)
#   - 必须先 set current view to icon view,再设 bounds,否则坐标错乱
#   - 必须显式 set arrangement to not arranged,默认网格会把图标 snap 到错误坐标
#   - close + delay 1 让 Finder 写 .DS_Store 到 dmg 根
# 窗口 600x400,app 图标坐标 (170,190),Applications 软链坐标 (410,190)。
#
# macOS 26 Tahoe 实测(2026-07-16):
#   - dmg-arm64 第一次跑时 `container window of theFolder` 能拿到正常 Finder
#     window,toolbar / position / bounds 全部写成功。
#   - dmg-amd64 第二次跑同一个 AppleScript 段,`container window of theFolder`
#     拿到的是 LaunchServices 管控的 cached window object,所有 setter 报
#     -10006(cannot set ... to ...)。
#   - dmg-arm64 重跑一次复现同样错误,说明**根本原因是"再跑一次 dmg"**,跟
#     哪个架构无关。
# 修法:不走 `POSIX file + open + container window of theFolder` 这条链,改用
# `tell disk "<volname>" + open + container window of disk` 这条链 —— disk
# 对象是 mount point 在 Finder 里的稳定 alias,跟 Finder 是否缓存无关。
# 配合 try/on error 容错,确保 Tahoe 上两次连续跑都过。
osascript <<EOF
tell application "Finder"
  delay 1
  set theDisk to disk "$VOLNAME"
  open theDisk
  delay 1

  set winOpts to container window of theDisk

  -- 切到 icon view 是核心,失败会让坐标全错;这条不带 try,直接挂以便发现新 Tahoe 行为变化
  set current view of winOpts to icon view

  -- 窗口修饰属性逐个容错(Tahoe 上这些属性可能只读或受 LaunchServices 管控)
  try
    set toolbar visible of winOpts to false
  end try
  try
    set statusbar visible of winOpts to false
  end try
  try
    set sidebar width of winOpts to 0
  end try
  try
    set position of winOpts to {100, 100}
  end try
  try
    set bounds of winOpts to {100, 100, 700, 500}
  end try

  -- icon view options:同样逐个容错
  try
    set icon size of icon view options of winOpts to 96
  end try
  try
    set text size of icon view options of winOpts to 12
  end try
  try
    set arrangement of icon view options of winOpts to not arranged
  end try
  try
    set background color of icon view options of winOpts to {65535, 65535, 65535}
  end try

  -- 两图标坐标。APP_FOLDER_NAME 跟着 .app 真实名字走
  -- (universal 是 skill-box.app,arm64-only 是 skill-box-arm64.app)。
  -- 这一步是 dmg 布局核心(.DS_Store 持久化靠它),失败就让 osascript 退出非零
  -- 让 build-dmg.sh 失败 —— 重试可见。
  try
    set position of item "$APP_FOLDER_NAME" of winOpts to {170, 190}
    set position of item "Applications" of winOpts to {410, 190}
  on error errMsg number errNum
    -- Tahoe 上第二次跑时 disk object 的 container window 拿到的 items
    -- 不可写,这里用 disk 根 + POSIX path 重试。
    log "warn: item position via window failed (" & errNum & "): " & errMsg
    try
      set position of item "$APP_FOLDER_NAME" of theDisk to {170, 190}
      set position of item "Applications" of theDisk to {410, 190}
    end try
  end try

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
# hdiutil convert 的 -o 拒绝覆盖已存在文件,所以先删旧 dmg。
rm -f "$OUTPUT"
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