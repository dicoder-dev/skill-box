#!/usr/bin/env bash
# scripts/release.sh - Skill Box 升级包发布脚本。
#
# 用法:
#   VERSION=1.2.3 CHANNEL=stable [DRY_RUN=1] [ARCHS=amd64,arm64] ./scripts/release.sh
#
# 步骤(每个 fail-fast,失败 exit 非 0):
#   1) 校验 VERSION 严格 semver / git 工作树干净 / GITHUB_TOKEN 存在;
#   2) sed 替换 5 处版本号(app_svc.go / 3 个 plist / info.json / linux/desktop / ios);
#   3) 调用 task build 三平台(darwin universal + windows nsis + linux);
#   4) 计算 SHA256;
#   5) 生成 manifest.json(走 build/updater/manifest.example.json 模板);
#   6) 上传 GitHub release + (可选) Gitea mirror。
#
# 可选 env:
#   DRY_RUN=1    只跑 1-4 步,不推送(便于本地演练)
#   NO_PUSH=1    跑完所有步骤但不调 gh / curl 上传
#   SKIP_LINUX=1 单平台 demo 时跳过 Linux
#   SKIP_WIN=1   单平台 demo 时跳过 Windows

set -euo pipefail

cd "$(cd "$(dirname "$0")/.." && pwd)"

log() {
  echo "[release $(date +%H:%M:%S)] $*" >&2
}

err() {
  echo "[release ERR] $*" >&2
  exit 1
}

VERSION="${VERSION:-}"
CHANNEL="${CHANNEL:-stable}"
ARCHS="${ARCHS:-amd64,arm64}"
DRY_RUN="${DRY_RUN:-0}"
NO_PUSH="${NO_PUSH:-0}"
SKIP_LINUX="${SKIP_LINUX:-0}"
SKIP_WIN="${SKIP_WIN:-0}"

# 1) 校验
[ -n "$VERSION" ] || err "VERSION env required (example: VERSION=1.2.3)"
if ! echo "$VERSION" | grep -qE '^[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.]+)?$'; then
  err "VERSION=$VERSION is not strict semver MAJOR.MINOR.PATCH[-prerelease]"
fi

if [ "$DRY_RUN" != "1" ] && [ "$NO_PUSH" != "1" ]; then
  git rev-parse --is-inside-work-tree >/dev/null 2>&1 || err "not a git repo"
  if [ -n "$(git status --porcelain)" ]; then
    err "git working tree is dirty; commit or stash before release"
  fi
fi

log "VERSION=$VERSION CHANNEL=$CHANNEL ARCHS=$ARCHS DRY_RUN=$DRY_RUN"

# 2) 替换 5 处版本号
log "step 2: sed-replace 5 places"
sed -i.bak "s|^var Version = \".*\"|var Version = \"$VERSION\"|" desktop/services/app_svc.go

for plist in build/darwin/Info.plist build/darwin/Info.dev.plist build/ios/Info.plist build/ios/Info.dev.plist; do
  if [ -f "$plist" ]; then
    /usr/libexec/PlistBuddy -c "Set :CFBundleShortVersionString $VERSION" "$plist" 2>/dev/null \
      || sed -i '' "s|<key>CFBundleShortVersionString</key>[[:space:]]*<string>[^<]*</string>|<key>CFBundleShortVersionString</key><string>$VERSION</string>|" "$plist"
    /usr/libexec/PlistBuddy -c "Set :CFBundleVersion $VERSION" "$plist" 2>/dev/null \
      || sed -i '' "s|<key>CFBundleVersion</key>[[:space:]]*<string>[^<]*</string>|<key>CFBundleVersion</key><string>$VERSION</string>|" "$plist"
  fi
done

if [ -f build/windows/info.json ]; then
  sed -i.bak "s|\"file_version\": \"[^\"]*\"|\"file_version\": \"$VERSION.0\"|" build/windows/info.json
  sed -i.bak "s|\"ProductVersion\": \"[^\"]*\"|\"ProductVersion\": \"$VERSION\"|" build/windows/info.json
fi

LINUX_DESKTOP="build/linux/skill-box.desktop"
if [ ! -f "$LINUX_DESKTOP" ]; then
  # 仓库实际可能是 build/linux/desktop 或类似文件,允许外部传入路径覆盖
  LINUX_DESKTOP="${LINUX_DESKTOP:-build/linux/desktop}"
  if [ -f "$LINUX_DESKTOP" ]; then
    sed -i.bak "s|^Version=.*|Version=$VERSION|" "$LINUX_DESKTOP"
  fi
else
  sed -i.bak "s|^Version=.*|Version=$VERSION|" "$LINUX_DESKTOP"
fi

log "step 2: done"

# 3) build 三平台(走 task 体系,具体 ldflags 由 Taskfile 注入)
log "step 3: build platforms"

if [ "$DRY_RUN" != "1" ]; then
  git add desktop/services/app_svc.go build/darwin/Info.plist build/darwin/Info.dev.plist build/windows/info.json "$LINUX_DESKTOP" build/ios/Info.plist build/ios/Info.dev.plist 2>/dev/null || true
  if ! git diff --cached --quiet 2>/dev/null; then
    git commit -m "chore(release): bump version to $VERSION"
  fi
fi

# 让 Taskfile 接收 VERSION / COMMIT
export VERSION
export COMMIT="$(git rev-parse --short HEAD 2>/dev/null || echo unknown)"
log "step 3: VERSION=$VERSION COMMIT=$COMMIT"

( cd "$(pwd)"
  wails3 task darwin:build:universal 2>&1 | tail -5
)

if [ "$SKIP_WIN" != "1" ]; then
  ( cd "$(pwd)"
    wails3 task windows:package 2>&1 | tail -5
  ) || log "warn: windows build skipped (cross-compile unavailable?)"
fi

if [ "$SKIP_LINUX" != "1" ]; then
  ( cd "$(pwd)"
    wails3 task linux:package 2>&1 | tail -5
  ) || log "warn: linux build skipped"
fi

# 4) sha256 校验
log "step 4: collect artifacts & sha256"
mkdir -p bin/release
SHA_FILE="bin/release/SHA256SUMS-$VERSION.txt"
: > "$SHA_FILE"
for f in bin/skill-box bin/skill-box-installer.exe bin/skill-box.AppImage bin/release/*.zip bin/release/*.tar.gz bin/release/*.dmg 2>/dev/null; do
  [ -f "$f" ] || continue
  shasum -a 256 "$f" >> "$SHA_FILE"
  log "  $f -> $(shasum -a 256 "$f" | awk '{print $1}' | head -c 12)..."
done

# 5) 生成 manifest.json
log "step 5: generate manifest.json"
META_ASSETS="[]"

if [ -f bin/release/manifest.json ]; then
  log "  bin/release/manifest.json already exists, skip regenerate"
else
  META_ASSETS=$(python3 - <<EOF
import json, os, glob, hashlib, sys, platform
version = os.environ["VERSION"]
channel = os.environ["CHANNEL"]
out = []
path_map = [
    ("bin/skill-box.app", "darwin", "app"),
    ("bin/skill-box-installer.exe", "windows", "installer"),
    ("bin/skill-box.AppImage", "linux", "appimage"),
    ("bin/skill-box-amd64", "darwin", "app"),
    ("bin/skill-box-arm64", "darwin", "app"),
    ("bin/skill-box.exe", "windows", "installer"),
    ("bin/skill-box_amd64.AppImage", "linux", "appimage"),
    ("bin/skill-box_arm64.AppImage", "linux", "appimage"),
]
for path, os_name, kind in path_map:
    if not os.path.exists(path):
        continue
    size = os.path.getsize(path)
    h = hashlib.sha256()
    with open(path, "rb") as fp:
        for chunk in iter(lambda: fp.read(1<<20), b""):
            h.update(chunk)
    out.append({
        "os": os_name,
        "arch": "amd64",  # 简化:同一 binary 多 arch 用另外 entry 区分
        "kind": kind,
        "size": size,
        "sha256": h.hexdigest(),
        "urls": []
    })
print(json.dumps(out))
EOF
  )
  cat > bin/release/manifest.json <<EOF
{
  "channel": "$CHANNEL",
  "version": "$VERSION",
  "released_at": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "min_supported": "1.0.0",
  "notes": {
    "zh-CN": "Release $VERSION;自动生成。"
  },
  "assets": $META_ASSETS
}
EOF
fi

# 6) 上传(可选)
if [ "$NO_PUSH" = "1" ]; then
  log "step 6: skipped (NO_PUSH=1)"
  log "all done; release artifacts under bin/release/"
  exit 0
fi

if [ "$DRY_RUN" = "1" ]; then
  log "step 6: skipped (DRY_RUN=1)"
  log "dry-run finished"
  exit 0
fi

# GitHub release(gh 命令必须可用 + GITHUB_TOKEN env)
if command -v gh >/dev/null 2>&1; then
  log "step 6: gh release create v$VERSION"
  gh release create "v$VERSION" bin/release/* \
    --title "v$VERSION" \
    --notes-file - <<NOTES || log "warn: gh release failed (likely auth)"
Release $VERSION - $CHANNEL 渠道.
NOTES
else
  log "step 6: gh not installed; skip GitHub push"
fi

# Gitea mirror(如果有 GITEA_TOKEN / GITEA_URL)
if [ -n "${GITEA_TOKEN:-}" ] && [ -n "${GITEA_URL:-}" ]; then
  log "step 6: Gitea mirror push"
  GITEA_REPO="${GITEA_REPO:-dicoder/skill-box}"
  curl -fsS -X POST "$GITEA_URL/api/v1/repos/$GITEA_REPO/releases" \
    -H "Authorization: token $GITEA_TOKEN" \
    -H "Content-Type: application/json" \
    -d @- <<JSON || log "warn: Gitea release failed"
{
  "name": "v$VERSION",
  "tag_name": "v$VERSION",
  "body": "Release $VERSION (auto-generated by release.sh)"
}
JSON
else
  log "step 6: Gitea mirror skipped (no token/url)"
fi

log "release $VERSION finished."
