#!/usr/bin/env bash
# scripts/release.sh - Skill Box 升级包发布脚本(双站:Gitee + GitHub)。
#
# 用法:
#   VERSION=1.2.3 CHANNEL=stable [DRY_RUN=1] [NO_PUSH=1] ./scripts/release.sh
#
# 更新源策略(与 api-server/pkg/updatersrc 一一对应):
#   - 中文用户(zh-CN)→ Gitee 优先,GitHub 兜底;
#   - 英文用户(en-US)→ GitHub 优先,Gitee 兜底。
#   两个站各托管一份 manifest 指针文件,内容除 assets[].urls 指向各自站点外完全一致:
#     GitHub: build/updater/manifest.github.json
#     Gitee : build/updater/manifest.gitee.json
#   客户端按语言选源,拿到哪份 manifest,安装包就从同一个站下载(不会出现半截状态)。
#
# 步骤(每个 fail-fast,失败 exit 非 0):
#   1) 校验 VERSION 严格 semver / git 工作树干净;
#   2) sed 替换版本号(app_svc.go / 3 个 plist / info.json / linux/desktop / ios);
#   3) 调用 task build 三平台(darwin universal + windows + linux);
#   4) 整理安装包:macOS/Windows 打 zip(Gitee 附件 100MB 上限,压缩后更稳);
#      产出 bin/release/assets.tsv 作为 manifest 的事实来源;
#   5) 计算 SHA256 + 生成两份 manifest(build/updater/manifest.{github,gitee}.json);
#   6) 上传:GitHub release(API)+ Gitee release(API,附件 ≤ GITEE_MAX_ATTACH_MB);
#   7) 提交并推送 manifest 指针文件到两个远端;
#   8) 校验:HEAD 请求各下载地址,404 直接失败。
#
# 可选 env:
#   DRY_RUN=1    只跑 1-5 步,不推送(便于本地演练)
#   NO_PUSH=1    跑完所有步骤但不调 API 上传(仍会生成本地产物)
#   SKIP_LINUX=1 单平台 demo 时跳过 Linux
#   SKIP_WIN=1   单平台 demo 时跳过 Windows
#   GITHUB_REPO  默认 dicoder-dev/skill-box
#   GITHUB_TOKEN GitHub 上传 token(需要 repo 写权限)
#   GITEE_REPO   默认 dicoder/skill-box
#   GITEE_USER   默认 dicoder
#   GITEE_TOKEN  Gitee 上传 token(需要仓库写权限)
#   GITEE_BRANCH 默认 main(必须是 Gitee 仓库的实际分支,否则 raw 404)
#   GITEE_MAX_ATTACH_MB 默认 100(Gitee 单附件上限)

set -euo pipefail

cd "$(cd "$(dirname "$0")/.." && pwd)"

APP_NAME="${APP_NAME:-Skill-Box}"   # 与 Taskfile.yml 的 APP_NAME 保持一致
BIN_DIR="${BIN_DIR:-bin}"            # 产物目录(测试时可指到别处)

log() {
  echo "[release $(date +%H:%M:%S)] $*" >&2
}

err() {
  echo "[release ERR] $*" >&2
  exit 1
}

VERSION="${VERSION:-}"
CHANNEL="${CHANNEL:-stable}"
DRY_RUN="${DRY_RUN:-0}"
NO_PUSH="${NO_PUSH:-0}"
SKIP_LINUX="${SKIP_LINUX:-0}"
SKIP_WIN="${SKIP_WIN:-0}"

GITHUB_REPO="${GITHUB_REPO:-dicoder-dev/skill-box}"
GITHUB_TOKEN="${GITHUB_TOKEN:-}"
GITEE_REPO="${GITEE_REPO:-dicoder/skill-box}"
GITEE_USER="${GITEE_USER:-dicoder}"
GITEE_TOKEN="${GITEE_TOKEN:-}"
GITEE_BRANCH="${GITEE_BRANCH:-main}"
GITEE_MAX_ATTACH_MB="${GITEE_MAX_ATTACH_MB:-100}"

RELEASE_TAG="v${VERSION}"
REL_DIR="${BIN_DIR}/release"

# ---------- 工具函数 ----------

sha256_of() {
  shasum -a 256 "$1" | awk '{print $1}'
}

file_size() {
  # stat 在 macOS / Linux 参数不同,统一用 wc -c
  wc -c < "$1" | tr -d ' '
}

# add_asset 登记一个安装包到 assets.tsv。
# 参数:os arch kind 文件名 路径
add_asset() {
  printf '%s\t%s\t%s\t%s\t%s\n' "$1" "$2" "$3" "$4" "$5" >> "$REL_DIR/assets.tsv"
}

# json_get 从 stdin 的 JSON 里取顶层字段(避免依赖 jq)。
json_get() {
  python3 -c 'import json,sys
try:
    d = json.load(sys.stdin)
except Exception:
    sys.exit(1)
key = sys.argv[1]
cur = d
for seg in key.split("."):
    if isinstance(cur, list):
        cur = cur[int(seg)]
    elif isinstance(cur, dict):
        cur = cur.get(seg)
    else:
        cur = None
    if cur is None:
        break
print("" if cur is None else cur)' "$1"
}

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

log "VERSION=$VERSION CHANNEL=$CHANNEL DRY_RUN=$DRY_RUN NO_PUSH=$NO_PUSH"
log "GitHub: $GITHUB_REPO   Gitee: $GITEE_REPO (branch=$GITEE_BRANCH)"

# 2) 替换版本号
log "step 2: sed-replace version placeholders"
if [ "$DRY_RUN" = "1" ]; then
  # 演练模式不动仓库里的版本号(避免把工作树弄脏)
  log "step 2: skipped (DRY_RUN=1)"
else
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
  LINUX_DESKTOP="${LINUX_DESKTOP:-build/linux/desktop}"
  if [ -f "$LINUX_DESKTOP" ]; then
    sed -i.bak "s|^Version=.*|Version=$VERSION|" "$LINUX_DESKTOP"
  fi
else
  sed -i.bak "s|^Version=.*|Version=$VERSION|" "$LINUX_DESKTOP"
fi

log "step 2: done"
fi

# 3) build 三平台
log "step 3: build platforms"

export VERSION
export COMMIT="$(git rev-parse --short HEAD 2>/dev/null || echo unknown)"

if [ "$DRY_RUN" != "1" ]; then
  git add desktop/services/app_svc.go build/darwin/Info.plist build/darwin/Info.dev.plist build/windows/info.json "$LINUX_DESKTOP" build/ios/Info.plist build/ios/Info.dev.plist 2>/dev/null || true
  if ! git diff --cached --quiet 2>/dev/null; then
    git commit -m "chore(release): bump version to $VERSION"
  fi
fi

if [ "$DRY_RUN" != "1" ]; then
  wails3 task darwin:build:universal 2>&1 | tail -5
  if [ "$SKIP_WIN" != "1" ]; then
    wails3 task windows:package 2>&1 | tail -5 || log "warn: windows build skipped (cross-compile unavailable?)"
  fi
  if [ "$SKIP_LINUX" != "1" ]; then
    wails3 task linux:package 2>&1 | tail -5 || log "warn: linux build skipped"
  fi
else
  log "step 3: skipped (DRY_RUN=1,复用现有 bin/ 产物)"
fi

# 4) 整理安装包
log "step 4: collect artifacts"
mkdir -p "$REL_DIR"
: > "$REL_DIR/assets.tsv"

# --- macOS: .app 打成 zip ---
# 为什么打 zip:Gitee 附件单文件上限 100MB,universal 二进制压缩后更稳;
# 替身脚本 helper_darwin.sh 也只认 .zip / .dmg。
MAC_APP="${BIN_DIR}/${APP_NAME}.app"
if [ -d "$MAC_APP" ]; then
  MAC_ZIP_NAME="skill-box_${VERSION}_darwin_universal.zip"
  MAC_ZIP="$REL_DIR/$MAC_ZIP_NAME"
  rm -f "$MAC_ZIP"
  # macOS 的 zip 会带 __MACOSX 元数据目录,排除掉
  ( cd "$BIN_DIR" && zip -qyry "release/$MAC_ZIP_NAME" "${APP_NAME}.app" -x '*.DS_Store' -x '__MACOSX/*' )
  # universal 包同时登记 arm64 / amd64,PickAsset 才能按 GOARCH 命中
  add_asset darwin arm64 app "$MAC_ZIP_NAME" "$MAC_ZIP"
  add_asset darwin amd64 app "$MAC_ZIP_NAME" "$MAC_ZIP"
  log "  macOS: $MAC_ZIP_NAME ($(( $(file_size "$MAC_ZIP") / 1024 / 1024 )) MB)"
else
  log "  warn: $MAC_APP not found, macOS asset skipped"
fi

# --- Windows: exe 打 zip(替身脚本只支持 zip) ---
WIN_EXE="${BIN_DIR}/${APP_NAME}.exe"
if [ -f "$WIN_EXE" ]; then
  WIN_ZIP_NAME="skill-box_${VERSION}_windows_amd64.zip"
  WIN_ZIP="$REL_DIR/$WIN_ZIP_NAME"
  rm -rf "$REL_DIR/win" && mkdir -p "$REL_DIR/win"
  # helper_windows.ps1 找的是 skill-box.exe,这里统一改名
  cp "$WIN_EXE" "$REL_DIR/win/skill-box.exe"
  cp "${BIN_DIR}/WebView2Loader.dll" "$REL_DIR/win/" 2>/dev/null || true
  rm -f "$WIN_ZIP"
  ( cd "$REL_DIR/win" && zip -qry "../$WIN_ZIP_NAME" . )
  add_asset windows amd64 app "$WIN_ZIP_NAME" "$WIN_ZIP"
  log "  windows: $WIN_ZIP_NAME ($(( $(file_size "$WIN_ZIP") / 1024 / 1024 )) MB)"
else
  log "  warn: $WIN_EXE not found, windows asset skipped"
fi

# --- Linux: AppImage 本身就是可执行包,直接用 ---
if [ "$SKIP_LINUX" != "1" ]; then
  for f in "${BIN_DIR}"/${APP_NAME}*.AppImage "${BIN_DIR}"/${APP_NAME}*.appimage; do
    [ -f "$f" ] || continue
    base="$(basename "$f")"
    case "$base" in
      *arm64*) larch=arm64 ;;
      *) larch=amd64 ;;
    esac
    add_asset linux "$larch" appimage "$base" "$f"
    log "  linux: $base ($(( $(file_size "$f") / 1024 / 1024 )) MB)"
  done
fi

# 手动下载用的附加产物(不进 manifest,只作为 release 附件)
EXTRA_FILES=""
for f in "${BIN_DIR}"/${APP_NAME}.dmg "${BIN_DIR}"/${APP_NAME}*-arm64.dmg "${BIN_DIR}"/${APP_NAME}*-amd64.dmg; do
  [ -f "$f" ] || continue
  EXTRA_FILES="$EXTRA_FILES $f"
done
[ -n "$EXTRA_FILES" ] && log "  extra (手动下载用): $EXTRA_FILES"

if [ ! -s "$REL_DIR/assets.tsv" ]; then
  err "没有可发布的安装包(assets.tsv 为空);先跑 build,或检查 bin/ 产物名"
fi

# 5) SHA256 + 生成两份 manifest
log "step 5: generate manifests"
GITHUB_BASE="https://github.com/${GITHUB_REPO}/releases/download/${RELEASE_TAG}"
GITEE_BASE="https://gitee.com/${GITEE_REPO}/releases/download/${RELEASE_TAG}"
export RELEASED_AT="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
export MIN_SUPPORTED="${MIN_SUPPORTED:-1.0.0}"

python3 - "$REL_DIR/assets.tsv" "$VERSION" "$CHANNEL" "$GITHUB_BASE" "$GITEE_BASE" "$GITEE_MAX_ATTACH_MB" "$REL_DIR" <<'PYEOF'
import hashlib, json, os, sys

tsv_path, version, channel, gh_base, gitee_base, gitee_max_mb, rel_dir = sys.argv[1:8]
gitee_max = int(gitee_max_mb) * 1024 * 1024

def sha256_of(path):
    h = hashlib.sha256()
    with open(path, "rb") as fp:
        for chunk in iter(lambda: fp.read(1 << 20), b""):
            h.update(chunk)
    return h.hexdigest()

rows = []
with open(tsv_path, encoding="utf-8") as fp:
    for line in fp:
        line = line.rstrip("\n")
        if not line.strip():
            continue
        os_name, arch, kind, filename, path = line.split("\t")
        rows.append((os_name, arch, kind, filename, path))

def build(base_url, skip_oversize=False):
    assets = []
    uploaded = []
    for os_name, arch, kind, filename, path in rows:
        size = os.path.getsize(path)
        if skip_oversize and size > gitee_max:
            print(f"  skip(> {gitee_max_mb}MB): {filename} ({size // 1024 // 1024} MB)", file=sys.stderr)
            continue
        assets.append({
            "os": os_name,
            "arch": arch,
            "kind": kind,
            "size": size,
            "sha256": sha256_of(path),
            "urls": [f"{base_url}/{filename}"],
        })
        uploaded.append(path)
    return assets, uploaded

def dump_manifest(name, assets):
    manifest = {
        "channel": channel,
        "version": version,
        "released_at": os.environ.get("RELEASED_AT", ""),
        "min_supported": os.environ.get("MIN_SUPPORTED", "1.0.0"),
        "notes": {
            "zh-CN": f"Release {version}:自动生成。",
            "en-US": f"Release {version}: auto-generated.",
        },
        "assets": assets,
    }
    with open(name, "w", encoding="utf-8") as fp:
        json.dump(manifest, fp, ensure_ascii=False, indent=2)
        fp.write("\n")
    print(f"  wrote {name} ({len(assets)} assets)")

os.makedirs("build/updater", exist_ok=True)

gh_assets, gh_files = build(gh_base, skip_oversize=False)
dump_manifest("build/updater/manifest.github.json", gh_assets)

gitee_assets, gitee_files = build(gitee_base, skip_oversize=True)
dump_manifest("build/updater/manifest.gitee.json", gitee_assets)

# 上传清单(按站拆开,Gitee 超限的包只进 GitHub)
with open(os.path.join(rel_dir, "upload.github.list"), "w", encoding="utf-8") as fp:
    for p in gh_files:
        fp.write(p + "\n")
with open(os.path.join(rel_dir, "upload.gitee.list"), "w", encoding="utf-8") as fp:
    for p in gitee_files:
        fp.write(p + "\n")
print(f"  upload lists: github={len(gh_files)} gitee={len(gitee_files)}")
PYEOF

# 6) 上传
if [ "$NO_PUSH" = "1" ] || [ "$DRY_RUN" = "1" ]; then
  log "step 6: skipped (NO_PUSH=$NO_PUSH DRY_RUN=$DRY_RUN)"
else
  log "step 6: upload to GitHub + Gitee"

  # ---- GitHub ----
  if [ -z "$GITHUB_TOKEN" ]; then
    err "GITHUB_TOKEN 未设置,无法上传 GitHub release"
  fi
  GH_RELEASE_JSON=$(curl -fsS -X POST "https://api.github.com/repos/${GITHUB_REPO}/releases" \
    -H "Authorization: Bearer ${GITHUB_TOKEN}" \
    -H "Accept: application/vnd.github+json" \
    -H "Content-Type: application/json" \
    -d "{\"tag_name\":\"${RELEASE_TAG}\",\"name\":\"${RELEASE_TAG}\",\"body\":\"Release ${VERSION} (${CHANNEL})\",\"draft\":false,\"prerelease\":false}" || true)
  GH_RELEASE_ID=$(printf '%s' "$GH_RELEASE_JSON" | json_get id || true)
  if [ -z "$GH_RELEASE_ID" ]; then
    # release 已存在(重发同一版本)→ 复用既有 release id
    GH_RELEASE_ID=$(curl -fsS "https://api.github.com/repos/${GITHUB_REPO}/releases/tags/${RELEASE_TAG}" \
      -H "Authorization: Bearer ${GITHUB_TOKEN}" \
      -H "Accept: application/vnd.github+json" | json_get id || true)
    [ -n "$GH_RELEASE_ID" ] || err "GitHub release 创建失败: $GH_RELEASE_JSON"
    log "  github release 已存在,复用 id=$GH_RELEASE_ID"
  else
    log "  github release created id=$GH_RELEASE_ID"
  fi

  upload_github_asset() {
    local path="$1"
    local name
    name="$(basename "$path")"
    curl -fsS -X POST \
      "https://uploads.github.com/repos/${GITHUB_REPO}/releases/${GH_RELEASE_ID}/assets?name=${name}" \
      -H "Authorization: Bearer ${GITHUB_TOKEN}" \
      -H "Content-Type: application/octet-stream" \
      --data-binary "@${path}" >/dev/null || err "GitHub 上传失败: $name"
    log "  github uploaded: $name"
  }

  # ---- Gitee ----
  if [ -z "$GITEE_TOKEN" ]; then
    err "GITEE_TOKEN 未设置,无法上传 Gitee release"
  fi
  GITEE_RELEASE_JSON=$(curl -fsS -X POST "https://gitee.com/api/v5/repos/${GITEE_REPO}/releases" \
    -H "Content-Type: application/json" \
    --data-urlencode "access_token=${GITEE_TOKEN}" \
    --data-urlencode "tag_name=${RELEASE_TAG}" \
    --data-urlencode "name=${RELEASE_TAG}" \
    --data-urlencode "body=Release ${VERSION} (${CHANNEL})" \
    --data-urlencode "target_commitish=${GITEE_BRANCH}" \
    --data-urlencode "prerelease=false" || true)
  GITEE_RELEASE_ID=$(printf '%s' "$GITEE_RELEASE_JSON" | json_get id || true)
  if [ -z "$GITEE_RELEASE_ID" ]; then
    GITEE_RELEASE_ID=$(curl -fsS "https://gitee.com/api/v5/repos/${GITEE_REPO}/releases/tags/${RELEASE_TAG}" \
      --data-urlencode "access_token=${GITEE_TOKEN}" | json_get id || true)
    [ -n "$GITEE_RELEASE_ID" ] || err "Gitee release 创建失败: $GITEE_RELEASE_JSON"
    log "  gitee release 已存在,复用 id=$GITEE_RELEASE_ID"
  else
    log "  gitee release created id=$GITEE_RELEASE_ID"
  fi

  upload_gitee_asset() {
    local path="$1"
    local name size
    name="$(basename "$path")"
    size="$(file_size "$path")"
    if [ "$size" -gt $(( ${GITEE_MAX_ATTACH_MB} * 1024 * 1024 )) ]; then
      err "Gitee 附件超过 ${GITEE_MAX_ATTACH_MB}MB 上限: $name ($(( size / 1024 / 1024 )) MB)"
    fi
    curl -fsS -X POST \
      "https://gitee.com/api/v5/repos/${GITEE_REPO}/releases/${GITEE_RELEASE_ID}/attach_files" \
      --data-urlencode "access_token=${GITEE_TOKEN}" \
      -F "file=@${path}" >/dev/null || err "Gitee 上传失败: $name"
    log "  gitee uploaded: $name"
  }

  # 逐个上传(按站拆开:Gitee 超限的包只进 GitHub)
  # 注意:用文件重定向而不是管道,否则 while 循环在子 shell 里跑,err 的 exit 不会终止主脚本。
  while read -r f; do
    [ -n "$f" ] && [ -f "$f" ] || continue
    upload_github_asset "$f"
  done < "$REL_DIR/upload.github.list"

  while read -r f; do
    [ -n "$f" ] && [ -f "$f" ] || continue
    upload_gitee_asset "$f"
  done < "$REL_DIR/upload.gitee.list"

  # dmg 等手动下载用的附加产物只挂 GitHub
  for f in $EXTRA_FILES; do
    [ -f "$f" ] || continue
    upload_github_asset "$f"
  done

  # 8) 校验下载地址(上传后立即 HEAD,404 直接失败)
  log "step 8: verify download urls"
  : > "$REL_DIR/verify.urls"
  while read -r f; do
    [ -n "$f" ] && [ -f "$f" ] || continue
    name="$(basename "$f")"
    echo "${GITHUB_BASE}/${name}" >> "$REL_DIR/verify.urls"
    echo "${GITEE_BASE}/${name}" >> "$REL_DIR/verify.urls"
  done < "$REL_DIR/upload.gitee.list"
  while read -r url; do
    [ -n "$url" ] || continue
    code=$(curl -s -o /dev/null -w '%{http_code}' -L -I "$url" || echo 000)
    [ "$code" = "200" ] || err "下载地址不可用(code=$code): $url"
    log "  verified: $url"
  done < "$REL_DIR/verify.urls"
fi

# 7) 提交并推送 manifest 指针文件(客户端靠这两个固定 URL 找"最新版本")
log "step 7: commit & push manifest pointers"
if [ "$DRY_RUN" != "1" ] && [ "$NO_PUSH" != "1" ]; then
  git add build/updater/manifest.github.json build/updater/manifest.gitee.json
  if ! git diff --cached --quiet; then
    git commit -m "chore(release): 更新 v${VERSION} 的 manifest 指针(GitHub + Gitee)"
  fi
  git push origin HEAD:main
  # Gitee 远端用一次性 URL 推,避免把 token 写进 .git/config
  git push "https://${GITEE_USER}:${GITEE_TOKEN}@gitee.com/${GITEE_REPO}.git" "HEAD:${GITEE_BRANCH}"
  log "  pushed manifest pointers to origin + gitee"
else
  log "step 7: skipped (NO_PUSH=$NO_PUSH DRY_RUN=$DRY_RUN)"
fi

log "release $VERSION finished."
log "manifest pointers:"
log "  GitHub: https://raw.githubusercontent.com/${GITHUB_REPO}/main/build/updater/manifest.github.json"
log "  Gitee : https://gitee.com/${GITEE_REPO}/raw/${GITEE_BRANCH}/build/updater/manifest.gitee.json"
