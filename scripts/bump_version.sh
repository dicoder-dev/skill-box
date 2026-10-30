#!/usr/bin/env bash
# scripts/bump_version.sh - 把版本号写进各平台构建产物的元数据。
#
# 用法: bash scripts/bump_version.sh <VERSION>
#
# 改动位置(与 wails3 打包链路一一对应):
#   desktop/services/app_svc.go   → Go 二进制里的 Version(ldflags 也注入,这里保持一致)
#   build/darwin/Info.plist       → macOS .app 的 CFBundleShortVersionString / CFBundleVersion
#   build/ios/Info.plist          → iOS
#   build/windows/info.json       → Windows exe 的 ProductVersion / file_version
#   build/linux/skill-box.desktop → Linux .desktop 的 Version=
#
# 这个脚本被三处复用:
#   - scripts/release.sh        (本地构建发版)
#   - scripts/release_all.sh    (一键发版:本地 bump 后打 tag)
#   - .github/workflows/release.yml (CI 里保证产物版本号与 tag 一致)
#
# 幂等:同一个版本号重复执行结果一致。

set -euo pipefail

cd "$(cd "$(dirname "$0")/.." && pwd)"

VERSION="${1:-}"
if [ -z "$VERSION" ]; then
  echo "usage: bash scripts/bump_version.sh <VERSION>" >&2
  exit 1
fi
if ! echo "$VERSION" | grep -qE '^[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.]+)?$'; then
  echo "bump_version: VERSION=$VERSION 不是严格 semver" >&2
  exit 1
fi

# Go 二进制里的版本号
sed -i.bak "s|^var Version = \".*\"|var Version = \"$VERSION\"|" desktop/services/app_svc.go

# macOS / iOS 的 Info.plist
for plist in build/darwin/Info.plist build/darwin/Info.dev.plist build/ios/Info.plist build/ios/Info.dev.plist; do
  [ -f "$plist" ] || continue
  if ! /usr/libexec/PlistBuddy -c "Set :CFBundleShortVersionString $VERSION" "$plist" 2>/dev/null; then
    # 没有 PlistBuddy(Linux CI)时退回 sed
    sed -i 's|<key>CFBundleShortVersionString</key>[[:space:]]*<string>[^<]*</string>|<key>CFBundleShortVersionString</key><string>'"$VERSION"'</string>|' "$plist"
    sed -i 's|<key>CFBundleVersion</key>[[:space:]]*<string>[^<]*</string>|<key>CFBundleVersion</key><string>'"$VERSION"'</string>|' "$plist"
  else
    /usr/libexec/PlistBuddy -c "Set :CFBundleVersion $VERSION" "$plist" 2>/dev/null || true
  fi
done

# Windows exe 元数据
if [ -f build/windows/info.json ]; then
  sed -i.bak "s|\"file_version\": \"[^\"]*\"|\"file_version\": \"$VERSION.0\"|" build/windows/info.json
  sed -i.bak "s|\"ProductVersion\": \"[^\"]*\"|\"ProductVersion\": \"$VERSION\"|" build/windows/info.json
fi

# Linux .desktop
for desktop_file in build/linux/skill-box.desktop build/linux/desktop build/linux/Skill-Box.desktop; do
  [ -f "$desktop_file" ] || continue
  sed -i.bak "s|^Version=.*|Version=$VERSION|" "$desktop_file"
done

echo "[bump_version] 版本号已写入: $VERSION"
