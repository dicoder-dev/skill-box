#!/bin/bash
# commit.sh — 收集 staged diff 并生成 commit message 提示词。
# 用法:bash commit.sh
set -e
DIFF=$(git diff --staged -- ':!*.lock' ':!package-lock.json')
if [ -z "$DIFF" ]; then
  echo "no staged changes"
  exit 0
fi
echo "=== staged diff ==="
echo "$DIFF"
echo
echo "=== commit message prompt ==="
cat <<'EOF'
请按 Conventional Commits 规范为上面的 diff 生成 commit message:

格式:
<type>(<scope>): <subject>

<body>

<footer>

要求:
- subject ≤ 72 字符,祈使语气
- body 解释"为什么"而不是"是什么"
- type 取值: feat / fix / refactor / docs / test / chore / perf / style
- scope 从改动路径推断
- 有 breaking change 必须在 footer 写 BREAKING CHANGE:
EOF
