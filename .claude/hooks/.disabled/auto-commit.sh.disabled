#!/bin/bash
# Auto-commit hook for Skill Box
# - 读取 stdin 的 PostToolUse JSON
# - 把对应仓库内文件 add + commit(没有变化则跳过)
INPUT=$(cat)
FILE=$(echo "$INPUT" | jq -r '.tool_response.filePath // .tool_input.file_path // empty')
[ -z "$FILE" ] && exit 0
ROOT="/Volumes/MyDrive/Home/dicoder/projects/skill-box"
cd "$ROOT" || exit 0
# 计算仓库内相对路径(兼容 macOS:用 python3)
REL=$(python3 -c "import os,sys; p=os.path.abspath('$FILE'); r=os.path.abspath('$ROOT'); print(os.path.relpath(p,r) if p.startswith(r) else '')" 2>/dev/null)
[ -z "$REL" ] && exit 0
# 排除垃圾与自身
case "$REL" in
  ../*) exit 0 ;;
  data.db|configs.yaml|*.db|*.lock|package-lock.json|node_modules/*|frontend/node_modules/*|frontend/dist/*|bin/*) exit 0 ;;
esac
[ "$REL" = ".claude/settings.local.json" ] && exit 0
# 跳过没有变化的(untracked 文件 status 输出为 "??")
STAT=$(git status --porcelain -- "$REL")
[ -z "$STAT" ] && exit 0
git add -- "$REL"
git diff --cached --quiet && exit 0
NAME=$(basename "$REL")
SCOPE=$(echo "$REL" | awk -F/ 'NF==1{print "root"; exit} {print $1; exit}')
TS=$(date +%H:%M:%S)
git commit -m "auto($SCOPE): $NAME @ $TS" >/dev/null 2>&1 || true
