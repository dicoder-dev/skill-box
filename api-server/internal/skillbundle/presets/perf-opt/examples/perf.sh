#!/bin/bash
# perf.sh — 收集性能分析所需的上下文,粘给 AI 定位热点。
# 用法:bash perf.sh [TARGET]
#   TARGET: cpu | mem | db | all(默认 all)
set -e
TARGET="${1:-all}"
echo "=== 1. 系统基线 ==="
uname -a
echo "--- CPU ---"
lscpu 2>/dev/null | grep -E "Model name|CPU\(s\)|MHz" | head -5 || sysctl -n machdep.cpu.brand_string
echo "--- 内存 ---"
free -h 2>/dev/null || vm_stat | head -5
echo
echo "=== 2. Go runtime(若适用) ==="
if command -v go >/dev/null 2>&1; then
  go version
  echo "--- pprof 入口检查 ---"
  grep -rE "net/http/pprof" --include="*.go" . 2>/dev/null | head -5
fi
echo
echo "=== 3. CPU profile(若已采集) ==="
[ -f cpu.prof ] && echo "found cpu.prof" && go tool pprof -top -nodecount=20 cpu.prof 2>/dev/null | head -30 || echo "(未提供 cpu.prof)"
echo
echo "=== 4. Heap profile(若已采集) ==="
[ -f mem.prof ] && echo "found mem.prof" && go tool pprof -top -nodecount=20 -alloc_space mem.prof 2>/dev/null | head -30 || echo "(未提供 mem.prof)"
echo
echo "=== 5. DB 慢查询 ==="
case "$TARGET" in
  db|all)
    if command -v psql >/dev/null 2>&1; then
      echo "(Postgres) 跑 \`SELECT * FROM pg_stat_statements ORDER BY total_time DESC LIMIT 10;\`"
    fi
    if [ -f data.db ]; then
      echo "(SQLite) 开启 log 并执行慢操作:"
      echo "  sqlite3 data.db \".timer on\""
      echo "  <运行目标 endpoint / 函数>"
    fi
    ;;
esac
echo
echo "=== 6. 性能分析 prompt ==="
cat <<'EOF'
请按以下顺序分析上面的 profile / 数据:

1. 列出 TOP 3 热点(按 self / cum 耗时)
2. 每个热点给"是不是根因"的一句话判断
3. 给 1-2 个最小改动(不动无关代码)
4. 每个改动给"如何复测"(具体命令 / 指标)
5. 复测后对比基线

注意:不许只说"应该会快",必须给量化对比。
EOF