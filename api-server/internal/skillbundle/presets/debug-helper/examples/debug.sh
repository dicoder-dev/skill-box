#!/bin/bash
# debug.sh — 收集排查所需的信息,粘给 AI 做根因分析。
# 用法:bash debug.sh [COMMAND_TO_REPRODUCE]
set -e
REPRO="${1:-}"
echo "=== 1. 进程 / 版本 ==="
uname -a 2>/dev/null || true
go version 2>/dev/null || true
node --version 2>/dev/null || true
python3 --version 2>/dev/null || true
echo
echo "=== 2. 最近 git 状态 ==="
git log --oneline -5 2>/dev/null || echo "no git"
git status --short 2>/dev/null || true
echo
echo "=== 3. 复现命令 ==="
if [ -n "$REPRO" ]; then
  echo "running: $REPRO"
  bash -c "$REPRO" 2>&1 | head -50 || true
else
  echo "(未提供,粘 stack trace / 错误信息给 AI 即可)"
fi
echo
echo "=== 4. 调试 prompt ==="
cat <<'EOF'
请按以下结构帮我排查上面复现的 bug:
1. 最小复现(可执行命令)
2. 候选根因(按概率排序,每条给 1 行验证命令)
3. 锁定根因 + 证据
4. 最小修复 patch
5. 预防(单测 / 类型 / 文档)
EOF
