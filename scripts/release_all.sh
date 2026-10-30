#!/usr/bin/env bash
# scripts/release_all.sh - 一键发版(GitHub Actions 打包三平台 → 同步到 Gitee)。
#
# 用法:
#   bash scripts/release_all.sh            # 交互式输入版本号
#   VERSION=1.2.3 bash scripts/release_all.sh
#
# 流程:
#   1) 校验 semver + git 工作树干净;
#   2) bump_version.sh 写版本号 → 提交 → push 到 origin;
#   3) 打 tag v<version> 并 push(触发 .github/workflows/release.yml);
#   4) 轮询 GitHub API 等 CI 产出 release(默认最多等 60 分钟);
#   5) 调 scripts/release.sh MODE=mirror:下载产物 → 传 Gitee → 生成 manifest → 推指针文件;
#   6) 校验两个站的下载地址。
#
# 为什么不在本地编 win/linux:本地是 macOS,交叉编译链(CGO / NSIS / AppImage)不可用。
# 为什么不在 CI 里直传 Gitee:Gitee 凭据不放进 CI,且产物要先落到本地做一次统一整理。
#
# 可选 env:
#   VERSION=1.2.3     指定版本号(不给就交互输入,默认取当前代码里的版本)
#   WAIT_TIMEOUT=3600 等待 CI 的秒数(默认 3600)
#   GITHUB_REPO       默认 dicoder-dev/skill-box
#   GITHUB_TOKEN      GitHub token(读不到会提示输入,存到 ~/.skill-box/release.env)
#   GITEE_REPO/GITEE_USER/GITEE_TOKEN/GITEE_BRANCH 同 release.sh

set -euo pipefail

cd "$(cd "$(dirname "$0")/.." && pwd)"

log() { echo "[release-all $(date +%H:%M:%S)] $*" >&2; }
err() { echo "[release-all ERR] $*" >&2; exit 1; }

GITHUB_REPO="${GITHUB_REPO:-dicoder-dev/skill-box}"
GITHUB_TOKEN="${GITHUB_TOKEN:-}"
GITEE_REPO="${GITEE_REPO:-dicoder/skill-box}"
GITEE_USER="${GITEE_USER:-dicoder}"
GITEE_TOKEN="${GITEE_TOKEN:-}"
GITEE_BRANCH="${GITEE_BRANCH:-main}"
WAIT_TIMEOUT="${WAIT_TIMEOUT:-3600}"
TOKEN_FILE="${SKILLBOX_HOME:-$HOME/.skill-box}/release.env"

# ---------- token 读取(与 release.sh 同一套规则) ----------
load_token() {
  local var="$1" label="$2"
  local current="${!var:-}"
  [ -n "$current" ] && return 0
  if [ -f "$TOKEN_FILE" ]; then
    # shellcheck disable=SC1090
    . "$TOKEN_FILE"
    current="${!var:-}"
    [ -n "$current" ] && return 0
  fi
  if [ ! -t 0 ]; then
    err "$label 未设置(非交互式终端无法输入)"
  fi
  local input=""
  read -r -s -p "请输入 $label(输入不回显): " input
  echo >&2
  [ -n "$input" ] || err "$label 不能为空"
  export "$var=$input"
  mkdir -p "$(dirname "$TOKEN_FILE")"
  if ! grep -q "^${var}=" "$TOKEN_FILE" 2>/dev/null; then
    printf '%s=%s\n' "$var" "$input" >> "$TOKEN_FILE"
    chmod 600 "$TOKEN_FILE"
    log "已把 $label 记到 $TOKEN_FILE"
  fi
}

# ---------- 1) 版本号 ----------

CURRENT_VERSION=$(sed -n 's|^var Version = "\(.*\)"|\1|p' desktop/services/app_svc.go | head -n 1)
VERSION="${VERSION:-}"
if [ -z "$VERSION" ] && [ -t 0 ]; then
  echo "当前代码版本号: ${CURRENT_VERSION:-未知}"
  read -r -p "请输入要发布的版本号(留空则用当前版本): " VERSION
fi
VERSION="${VERSION:-$CURRENT_VERSION}"
[ -n "$VERSION" ] || err "拿不到版本号"
if ! echo "$VERSION" | grep -qE '^[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.]+)?$'; then
  err "VERSION=$VERSION 不是严格 semver"
fi

log "准备发布 v${VERSION}"

# ---------- 2) 前置检查 ----------

git rev-parse --is-inside-work-tree >/dev/null 2>&1 || err "not a git repo"
if [ -n "$(git status --porcelain)" ]; then
  err "git 工作树不干净,先提交或 stash 再发版"
fi
if git rev-parse "v${VERSION}" >/dev/null 2>&1; then
  err "tag v${VERSION} 已存在,换个版本号"
fi

# ---------- 3) bump 版本并提交 ----------

log "step 1: bump version + commit"
bash scripts/bump_version.sh "$VERSION"
git add desktop/services/app_svc.go build/darwin/Info.plist build/darwin/Info.dev.plist \
        build/ios/Info.plist build/ios/Info.dev.plist build/windows/info.json \
        build/linux/skill-box.desktop 2>/dev/null || true
git add -u
if ! git diff --cached --quiet; then
  git commit -m "chore(release): bump version to $VERSION"
fi
git push origin HEAD:main
log "  已提交并推送版本号"

# ---------- 4) 打 tag 触发 CI ----------

log "step 2: tag + push (触发 GitHub Actions)"
git tag -a "v${VERSION}" -m "Release ${VERSION}"
git push origin "v${VERSION}"
log "  tag v${VERSION} 已推送,CI 开始构建三平台"
echo "  Actions 页面: https://github.com/${GITHUB_REPO}/actions"

# ---------- 5) 等 CI 出 release ----------

log "step 3: 等待 GitHub Actions 产出 release(最多 ${WAIT_TIMEOUT}s)"
[ -n "$GITHUB_TOKEN" ] || load_token GITHUB_TOKEN "GITHUB_TOKEN"

WAITED=0
RELEASE_READY=0
while [ "$WAITED" -lt "$WAIT_TIMEOUT" ]; do
  code=$(curl -s -o /tmp/.rel_check.json -w '%{http_code}' \
    -H "Authorization: Bearer ${GITHUB_TOKEN}" \
    -H "Accept: application/vnd.github+json" \
    "https://api.github.com/repos/${GITHUB_REPO}/releases/tags/v${VERSION}" || echo 000)
  if [ "$code" = "200" ]; then
    RELEASE_READY=1
    break
  fi
  if [ $(( WAITED % 60 )) -eq 0 ]; then
    log "  ...已等待 ${WAITED}s,CI 还在跑"
  fi
  sleep 15
  WAITED=$(( WAITED + 15 ))
done

if [ "$RELEASE_READY" != "1" ]; then
  err "等待 ${WAIT_TIMEOUT}s 仍未拿到 release v${VERSION};请到 Actions 页面看失败原因"
fi
log "  release 已就绪"

# ---------- 6) 同步到 Gitee ----------

log "step 4: 同步到 Gitee(下载产物 → 上传 → 生成 manifest → 推指针)"
[ -n "$GITEE_TOKEN" ] || load_token GITEE_TOKEN "GITEE_TOKEN"

VERSION="$VERSION" CHANNEL="${CHANNEL:-stable}" MODE=mirror \
  GITHUB_REPO="$GITHUB_REPO" GITHUB_TOKEN="$GITHUB_TOKEN" \
  GITEE_REPO="$GITEE_REPO" GITEE_USER="$GITEE_USER" GITEE_TOKEN="$GITEE_TOKEN" \
  GITEE_BRANCH="$GITEE_BRANCH" \
  bash scripts/release.sh

log "发布完成: v${VERSION}"
echo "  GitHub: https://github.com/${GITHUB_REPO}/releases/tag/v${VERSION}"
echo "  Gitee : https://gitee.com/${GITEE_REPO}/releases/tag/v${VERSION}"
