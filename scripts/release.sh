#!/usr/bin/env bash
# scripts/release.sh - Skill Box 升级包发布脚本(双站:Gitee + GitHub)。
#
# 用法:
#   VERSION=1.2.3 CHANNEL=stable ./scripts/release.sh              # MODE=build 本地构建后发双站
#   VERSION=1.2.3 MODE=mirror ./scripts/release.sh                 # MODE=mirror 拉 GitHub 产物同步到 Gitee
#
# 更新源策略(与 api-server/pkg/updatersrc 一一对应):
#   - 中文用户(zh-CN)→ Gitee 优先,GitHub 兜底;
#   - 英文用户(en-US)→ GitHub 优先,Gitee 兜底。
#   两个站各托管一份 manifest 指针文件,内容除 assets[].urls 指向各自站点外完全一致:
#     GitHub: build/updater/manifest.github.json
#     Gitee : build/updater/manifest.gitee.json
#   客户端按语言选源,拿到哪份 manifest,安装包就从同一个站下载(不会出现半截状态)。
#
# MODE=build 步骤:
#   1) 校验 VERSION 严格 semver / git 工作树干净;
#   2) bump_version.sh 写入版本号;
#   3) build 三平台 → collect_artifacts.sh 整理成发布命名约定的产物;
#   4) gen_manifest.py 生成两份 manifest;
#   5) 上传 GitHub release + Gitee release;
#   6) 提交并推送 manifest 指针文件到两个远端;
#   7) HEAD 校验各下载地址,404 直接失败。
#
# MODE=mirror 步骤(本地 mac 编不出 win/linux 时走这条):
#   1) 从 GitHub release 下载产物(产物是 CI 打的);
#   2) 上传到 Gitee release;
#   3) gen_manifest.py 生成两份 manifest;
#   4) 提交并推送 manifest 指针文件到两个远端;
#   5) HEAD 校验各下载地址,404 直接失败。
#
# 可选 env:
#   MODE=build|mirror  默认 build
#   DRY_RUN=1          只跑到"生成 manifest",不推不传(便于本地演练)
#   NO_PUSH=1          跑完所有步骤但不调 API 上传
#   GITHUB_REPO        默认 dicoder-dev/skill-box
#   GITHUB_TOKEN       GitHub 上传 token(需要 repo 写权限)
#   GITEE_REPO         默认 dicoder/skill-box
#   GITEE_USER         默认 dicoder
#   GITEE_TOKEN        Gitee 上传 token(需要仓库写权限)
#   GITEE_BRANCH       默认 main(必须是 Gitee 仓库的实际分支,否则 raw 404)
#   GITEE_MAX_ATTACH_MB 默认 100(Gitee 单附件上限)

set -euo pipefail

cd "$(cd "$(dirname "$0")/.." && pwd)"

APP_NAME="${APP_NAME:-Skill-Box}"   # 与 Taskfile.yml 的 APP_NAME 保持一致
BIN_DIR="${BIN_DIR:-bin}"            # 产物目录(测试时可指到别处)

log() { echo "[release $(date +%H:%M:%S)] $*" >&2; }
err() { echo "[release ERR] $*" >&2; exit 1; }

VERSION="${VERSION:-}"
CHANNEL="${CHANNEL:-stable}"
MODE="${MODE:-build}"
DRY_RUN="${DRY_RUN:-0}"
NO_PUSH="${NO_PUSH:-0}"

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

sha256_of() { shasum -a 256 "$1" | awk '{print $1}'; }
file_size() { wc -c < "$1" | tr -d ' '; }

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

# load_token <变量名> <说明>
# 取值顺序:环境变量 → ~/.skill-box/release.env → 交互输入(不回显,并询问是否记住)。
load_token() {
  local var="$1" label="$2"
  local current="${!var:-}"
  if [ -n "$current" ]; then
    return 0
  fi
  local env_file="${SKILLBOX_HOME:-$HOME/.skill-box}/release.env"
  if [ -f "$env_file" ]; then
    # shellcheck disable=SC1090
    . "$env_file"
    current="${!var:-}"
    if [ -n "$current" ]; then
      return 0
    fi
  fi
  if [ ! -t 0 ]; then
    err "$label 未设置(当前不是交互式终端,无法输入)"
  fi
  local input=""
  read -r -s -p "请输入 $label(输入不回显): " input
  echo >&2
  [ -n "$input" ] || err "$label 不能为空"
  export "$var=$input"
  if [ -f "$env_file" ] && grep -q "^${var}=" "$env_file"; then
    :
  else
    mkdir -p "$(dirname "$env_file")"
    printf '%s=%s\n' "$var" "$input" >> "$env_file"
    chmod 600 "$env_file"
    log "已把 $label 记到 $env_file(下次自动读取)"
  fi
}

# ---------- 1) 校验 ----------

[ -n "$VERSION" ] || err "VERSION env required (example: VERSION=1.2.3)"
if ! echo "$VERSION" | grep -qE '^[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.]+)?$'; then
  err "VERSION=$VERSION is not strict semver MAJOR.MINOR.PATCH[-prerelease]"
fi
case "$MODE" in
  build | mirror) ;;
  *) err "MODE=$MODE 无效(支持 build / mirror)" ;;
esac

if [ "$DRY_RUN" != "1" ] && [ "$NO_PUSH" != "1" ]; then
  git rev-parse --is-inside-work-tree >/dev/null 2>&1 || err "not a git repo"
  if [ -n "$(git status --porcelain)" ]; then
    err "git working tree is dirty; commit or stash before release"
  fi
fi

GITHUB_BASE="https://github.com/${GITHUB_REPO}/releases/download/${RELEASE_TAG}"
GITEE_BASE="https://gitee.com/${GITEE_REPO}/releases/download/${RELEASE_TAG}"

log "VERSION=$VERSION CHANNEL=$CHANNEL MODE=$MODE DRY_RUN=$DRY_RUN NO_PUSH=$NO_PUSH"
log "GitHub: $GITHUB_REPO   Gitee: $GITEE_REPO (branch=$GITEE_BRANCH)"

# ---------- 2) 版本号 ----------

if [ "$MODE" = "build" ]; then
  log "step 2: bump version"
  if [ "$DRY_RUN" = "1" ]; then
    log "step 2: skipped (DRY_RUN=1)"
  else
    bash scripts/bump_version.sh "$VERSION"
  fi

  # ---------- 3) build 三平台 ----------
  log "step 3: build platforms"
  export VERSION
  export COMMIT="$(git rev-parse --short HEAD 2>/dev/null || echo unknown)"

  if [ "$DRY_RUN" = "1" ]; then
    log "step 3: skipped (DRY_RUN=1,复用现有 $BIN_DIR/ 产物)"
  else
    wails3 task darwin:package:universal 2>&1 | tail -5
    wails3 task windows:package 2>&1 | tail -5 || log "warn: windows build skipped (cross-compile unavailable?)"
    wails3 task linux:create:appimage 2>&1 | tail -5 || log "warn: linux build skipped"
  fi

  # ---------- 4) 整理产物 ----------
  log "step 4: collect artifacts"
  mkdir -p "$REL_DIR"
  for target in darwin windows linux; do
    bash scripts/collect_artifacts.sh "$target" "$VERSION" || log "warn: $target 产物缺失,跳过"
  done
  ARTIFACT_FILES=$(ls -1 "$REL_DIR" 2>/dev/null | grep -v '^\.' || true)
  if [ -z "$ARTIFACT_FILES" ]; then
    err "没有可发布的安装包($REL_DIR 为空);先跑 build,或检查 bin/ 产物名"
  fi
  log "产物清单: $(echo "$ARTIFACT_FILES" | tr '\n' ' ')"
fi

# ---------- mirror:从 GitHub release 下载产物 ----------

if [ "$MODE" = "mirror" ]; then
  log "step 2: download GitHub release assets"
  [ -n "$GITHUB_TOKEN" ] || load_token GITHUB_TOKEN "GITHUB_TOKEN"
  mkdir -p "$REL_DIR"

  RELEASE_JSON=$(curl -fsS "https://api.github.com/repos/${GITHUB_REPO}/releases/tags/${RELEASE_TAG}" \
    -H "Authorization: Bearer ${GITHUB_TOKEN}" \
    -H "Accept: application/vnd.github+json" || true)
  if [ -z "$RELEASE_JSON" ]; then
    err "GitHub 上找不到 release ${RELEASE_TAG}(先跑 CI 或用 MODE=build 本地发)"
  fi
  # 输出 "name<TAB>download_url" 行
  ASSET_LIST=$(printf '%s' "$RELEASE_JSON" | python3 -c 'import json,sys
d = json.load(sys.stdin)
for a in d.get("assets", []):
    print(f"{a[\"name\"]}\t{a[\"browser_download_url\"]}")')
  if [ -z "$ASSET_LIST" ]; then
    err "GitHub release ${RELEASE_TAG} 里没有资产"
  fi
  while IFS=$'\t' read -r name url; do
    [ -n "$name" ] || continue
    case "$name" in
      manifest.*) continue ;;   # manifest 由本地生成,不用下载
    esac
    log "  下载 $name"
    curl -fsSL -H "Authorization: Bearer ${GITHUB_TOKEN}" -o "$REL_DIR/$name" "$url" \
      || err "下载失败: $name"
  done <<< "$ASSET_LIST"
  ARTIFACT_FILES=$(ls -1 "$REL_DIR" 2>/dev/null | grep -v '^\.' || true)
  if [ -z "$ARTIFACT_FILES" ]; then
    err "下载后 $REL_DIR 仍为空"
  fi
  log "产物清单: $(echo "$ARTIFACT_FILES" | tr '\n' ' ')"
fi

# ---------- 5) 生成两份 manifest ----------

log "step 5: generate manifests"
mkdir -p build/updater
RELEASED_AT="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
FILE_ARGS=()
for f in $ARTIFACT_FILES; do
  FILE_ARGS+=("$REL_DIR/$f")
done

python3 scripts/gen_manifest.py \
  --version "$VERSION" \
  --channel "$CHANNEL" \
  --github-base "$GITHUB_BASE" \
  --gitee-base "$GITEE_BASE" \
  --out-dir build/updater \
  --released-at "$RELEASED_AT" \
  --gitee-max-mb "$GITEE_MAX_ATTACH_MB" \
  --files "${FILE_ARGS[@]}"

# ---------- 6) 上传 ----------

if [ "$NO_PUSH" = "1" ] || [ "$DRY_RUN" = "1" ]; then
  log "step 6: skipped (NO_PUSH=$NO_PUSH DRY_RUN=$DRY_RUN)"
else
  log "step 6: upload"

  # ---- GitHub ----
  if [ "$MODE" = "build" ]; then
    [ -n "$GITHUB_TOKEN" ] || load_token GITHUB_TOKEN "GITHUB_TOKEN"
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
      local path="$1" name
      name="$(basename "$path")"
      curl -fsS -X POST \
        "https://uploads.github.com/repos/${GITHUB_REPO}/releases/${GH_RELEASE_ID}/assets?name=${name}" \
        -H "Authorization: Bearer ${GITHUB_TOKEN}" \
        -H "Content-Type: application/octet-stream" \
        --data-binary "@${path}" >/dev/null || err "GitHub 上传失败: $name"
      log "  github uploaded: $name"
    }

    while read -r f; do
      [ -n "$f" ] && [ -f "$f" ] || continue
      upload_github_asset "$f"
    done < <(printf '%s\n' "${FILE_ARGS[@]}")
  fi

  # ---- Gitee ----
  [ -n "$GITEE_TOKEN" ] || load_token GITEE_TOKEN "GITEE_TOKEN"
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
    local path="$1" name size
    name="$(basename "$path")"
    size="$(file_size "$path")"
    if [ "$size" -gt $(( GITEE_MAX_ATTACH_MB * 1024 * 1024 )) ]; then
      err "Gitee 附件超过 ${GITEE_MAX_ATTACH_MB}MB 上限: $name ($(( size / 1024 / 1024 )) MB)"
    fi
    curl -fsS -X POST \
      "https://gitee.com/api/v5/repos/${GITEE_REPO}/releases/${GITEE_RELEASE_ID}/attach_files" \
      --data-urlencode "access_token=${GITEE_TOKEN}" \
      -F "file=@${path}" >/dev/null || err "Gitee 上传失败: $name"
    log "  gitee uploaded: $name"
  }

  while read -r f; do
    [ -n "$f" ] && [ -f "$f" ] || continue
    upload_gitee_asset "$f"
  done < <(printf '%s\n' "${FILE_ARGS[@]}")
fi

# ---------- 7) 推送 manifest 指针文件 ----------

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

# ---------- 8) 校验下载地址 ----------

if [ "$NO_PUSH" != "1" ] && [ "$DRY_RUN" != "1" ]; then
  log "step 8: verify download urls"
  : > "$REL_DIR/verify.urls"
  for f in "${FILE_ARGS[@]}"; do
    [ -f "$f" ] || continue
    name="$(basename "$f")"
    echo "${GITHUB_BASE}/${name}" >> "$REL_DIR/verify.urls"
    echo "${GITEE_BASE}/${name}" >> "$REL_DIR/verify.urls"
  done
  while read -r url; do
    [ -n "$url" ] || continue
    code=$(curl -s -o /dev/null -w '%{http_code}' -L -I "$url" || echo 000)
    [ "$code" = "200" ] || err "下载地址不可用(code=$code): $url"
    log "  verified: $url"
  done < "$REL_DIR/verify.urls"
fi

log "release $VERSION finished."
log "manifest pointers:"
log "  GitHub: https://raw.githubusercontent.com/${GITHUB_REPO}/main/build/updater/manifest.github.json"
log "  Gitee : https://gitee.com/${GITEE_REPO}/raw/${GITEE_BRANCH}/build/updater/manifest.gitee.json"
