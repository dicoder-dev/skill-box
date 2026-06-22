#!/bin/bash
# doc.sh — 收集待写文档的模块信息,粘给 AI 生成 README / API 参考 / CHANGELOG。
# 用法:bash doc.sh [MODULE_PATH] [TARGET]
#   TARGET: readme | api | changelog(默认 readme)
set -e
TARGET="${2:-readme}"
MODULE_PATH="${1:-.}"
echo "=== 1. 模块路径 ==="
echo "module: $MODULE_PATH"
echo
echo "=== 2. 关键文件清单(按扩展名) ==="
if [ -d "$MODULE_PATH" ]; then
  find "$MODULE_PATH" -maxdepth 3 -type f \
    \( -name "*.go" -o -name "*.ts" -o -name "*.vue" -o -name "*.py" \
       -o -name "*.rs" -o -name "*.js" -o -name "*.md" \) \
    | grep -vE '/(node_modules|vendor|dist|build)/' \
    | sort | head -40
fi
echo
echo "=== 3. 包 / 模块入口 ==="
case "$TARGET" in
  readme)
    head -1 "$MODULE_PATH"/README.md 2>/dev/null || echo "(无 README)"
    ls "$MODULE_PATH" | head -20
    ;;
  api)
    echo "(API: 粘 endpoint 定义 / handler 函数签名即可)"
    grep -rE "func .* (Handle|List|Get|Create|Update|Delete)" "$MODULE_PATH" 2>/dev/null \
      --include="*.go" | head -20 || true
    ;;
  changelog)
    # 取最近 20 条 commit
    git log --oneline -20 2>/dev/null || echo "(无 git 历史)"
    ;;
esac
echo
echo "=== 4. 文档生成 prompt ==="
cat <<EOF
请为以上模块生成 **$TARGET**。

要求:
- 不修改源代码,只产出新文件
- 中英文版都生成(\`README.md\` + \`README.en.md\`)
- 引用具体函数名 / 类型 / endpoint,不放空话
EOF
