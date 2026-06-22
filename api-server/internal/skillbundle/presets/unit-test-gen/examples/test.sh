#!/bin/bash
# test.sh — 收集目标函数上下文,粘给 AI 生成单元测试。
# 用法:bash test.sh <FILE>:<FUNC>
# 例: bash test.sh api-server/internal/skillapp/applier.go:Apply
set -e
TARGET="${1:-}"
if [ -z "$TARGET" ]; then
  echo "usage: bash test.sh <FILE>:<FUNC>"
  exit 1
fi
FILE="${TARGET%:*}"
FUNC="${TARGET##*:}"
echo "=== 1. 函数源码 ==="
if [ -f "$FILE" ]; then
  # 简单取从 func 行开始到下一个顶层声明之前的范围
  awk -v fn="$FUNC" '
    $0 ~ "^func .*\\<"fn"\\(" {flag=1; brace=0}
    flag {print; for(i=1;i<=length($0);i++){c=substr($0,i,1); if(c=="{")brace++; if(c=="}"){brace--; if(brace==0){flag=0; print "--"; exit}}}}}' "$FILE"
fi
echo
echo "=== 2. 同文件 import(参考依赖) ==="
sed -n '/^import (/,/^)/p; /^import "/p' "$FILE" 2>/dev/null | head -30
echo
echo "=== 3. 调用方 / 反向引用 ==="
grep -rn "$FUNC(" --include="*.go" . 2>/dev/null | grep -v "_test.go" | head -10
echo
echo "=== 4. 测试生成 prompt ==="
cat <<EOF
请为上面的 \`$FUNC\` 生成 table-driven 单元测试:

- 文件: <按项目惯例>  (\`<filename>_test.go\` 与源码同包)
- 用例: 至少 3 条(happy / 边界 / 异常)
- 依赖: 真实依赖请用 fake / mock(参考调用方代码)
- 输出: 完整 \`_test.go\` 文件内容,不要省略
EOF