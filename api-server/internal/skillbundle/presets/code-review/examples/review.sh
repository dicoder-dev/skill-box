#!/bin/bash
# review.sh — 把当前 git diff 喂给 AI 做 code review 的 helper。
# 用法:bash review.sh [BASE_REF]    # 默认 HEAD~1
set -e
BASE="${1:-HEAD~1}"
DIFF=$(git diff "$BASE" -- ':!*.lock' ':!package-lock.json')
if [ -z "$DIFF" ]; then
  echo "no diff vs $BASE"
  exit 0
fi
echo "=== diff vs $BASE ==="
echo "$DIFF"
echo
echo "=== review prompt ==="
cat <<EOF
请按以下 4 个维度审查上面的 diff:
1. 可读性(命名 / 长度 / 嵌套 / 注释)
2. 正确性(边界 / 错误处理 / 并发)
3. 一致性(与项目风格)
4. 可维护性(耦合 / 测试 / 文档)

每个问题给:文件:行号 + 维度 + 严重程度 + 可粘贴的修复代码。
末尾给整体评分和 1-2 句总结。
EOF
