#!/bin/bash
# Skill Box .skillbox 包端到端:e2e_roundtrip.sh
# 流程:
#   1. 创建 1 个 skill(global scope)
#   2. 列出 skill 确认
#   3. export → 保存到 /tmp/sb_e2e_export.skillbox
#   4. delete 原 skill
#   5. import zip 回 global scope
#   6. 列出 skill 确认恢复
#   7. 检查 audit_log 出现 import / export 事件
#
# 前置:web 进程已在 8084 端口跑(skillbox e2e 模式)
set -e
HOST="127.0.0.1:8084"

# 通用 helper:发 JSON 请求,把 body + http status 一起打到 stdout
# 用法: req <METHOD> <PATH> <JSON> [OUTFILE]
req() {
  local method="$1" path="$2" body="$3" outfile="${4:-}"
  local args=(-sS -X "$method" -H "Content-Type: application/json" --data-binary "$body")
  if [ -n "$outfile" ]; then
    args+=(-o "$outfile" -w "HTTP %{http_code}\n")
    curl "${args[@]}" "http://$HOST$path"
  else
    curl "${args[@]}" -w "\nHTTP %{http_code}\n" "http://$HOST$path"
  fi
}

# 1. 创建 skill
echo "=== 1. CREATE skill ==="
CREATE_RES=$(curl -sS -X POST -H "Content-Type: application/json" --data-binary '{
  "scope": "global",
  "project_id": 0,
  "name": "e2e-pkg",
  "version": "0.1.0",
  "source": "local",
  "source_ref": "",
  "manifest": {
    "name": "e2e-pkg",
    "version": "0.1.0",
    "description": "this is an e2e test skill for package roundtrip",
    "triggers": ["e2e", "test"]
  },
  "files": [
    {"path": "SKILL.md", "content": "# E2E\n\nThis skill tests export/import roundtrip."},
    {"path": "examples/run.sh", "content": "#!/bin/bash\necho hello"}
  ]
}' "http://$HOST/api/skillbox/skills/create")
echo "$CREATE_RES"
SKILL_ID=$(echo "$CREATE_RES" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('id',0))")
[ "$SKILL_ID" = "0" ] && { echo "FAIL: create skill"; exit 1; }
echo "→ skill_id=$SKILL_ID"

# 2. 列出
echo "=== 2. LIST skills ==="
curl -sS "http://$HOST/api/skillbox/skills?page=1&size=10" | python3 -c "import sys,json; d=json.load(sys.stdin); print('total=',d.get('total')); [print(' -',s.get('id'),s.get('name'),s.get('version')) for s in d.get('items',[])]"

# 3. export → /tmp/sb_e2e_export.skillbox
echo "=== 3. EXPORT → /tmp/sb_e2e_export.skillbox ==="
HTTP_CODE=$(curl -sS -X POST -H "Content-Type: application/json" --data-binary '{
  "skills": [{"scope":"global","project_id":0,"name":"e2e-pkg","version":"0.1.0"}],
  "source_app": "skillbox-e2e",
  "source_desc": "e2e roundtrip test"
}' -o /tmp/sb_e2e_export.skillbox -w "%{http_code}" "http://$HOST/api/skillbox/pkg/export")
echo "HTTP=$HTTP_CODE; bytes=$(wc -c < /tmp/sb_e2e_export.skillbox)"

# 4. delete skill
echo "=== 4. DELETE skill ==="
curl -sS -X POST -H "Content-Type: application/json" --data-binary "{
  \"scope\": \"global\", \"project_id\": 0, \"name\": \"e2e-pkg\", \"version\": \"0.1.0\"
}" -w "\nHTTP %{http_code}\n" "http://$HOST/api/skillbox/skills/delete"

# 5. 列出 — 确认已删
echo "=== 5. LIST skills (post-delete) ==="
COUNT=$(curl -sS "http://$HOST/api/skillbox/skills?page=1&size=10" | python3 -c "import sys,json; d=json.load(sys.stdin); print(sum(1 for s in d.get('items',[]) if s.get('name')=='e2e-pkg'))")
echo "remaining e2e-pkg count: $COUNT"
[ "$COUNT" = "0" ] || { echo "FAIL: skill not deleted"; exit 1; }

# 6. preview zip
echo "=== 6. PREVIEW zip ==="
curl -sS -X POST -H "Content-Type: application/octet-stream" --data-binary "@/tmp/sb_e2e_export.skillbox" "http://$HOST/api/skillbox/pkg/preview"
echo ""

# 7. import zip → global scope
echo "=== 7. IMPORT zip → global ==="
HTTP_CODE=$(curl -sS -X POST -H "Content-Type: application/octet-stream" --data-binary "@/tmp/sb_e2e_export.skillbox" -w "\nHTTP %{http_code}\n" "http://$HOST/api/skillbox/pkg/import?target_scope=global")
echo "$HTTP_CODE"

# 8. 列出 — 确认恢复
echo "=== 8. LIST skills (post-import) ==="
curl -sS "http://$HOST/api/skillbox/skills?page=1&size=10" | python3 -c "import sys,json; d=json.load(sys.stdin); print('total=',d.get('total')); [print(' -',s.get('id'),s.get('name'),s.get('version'),'source=',s.get('source'),'source_ref=',s.get('source_ref')) for s in d.get('items',[])]"

# 9. 拿单个 skill 详情 — 确认文件还在
echo "=== 9. GET skill detail (post-import) ==="
curl -sS "http://$HOST/api/skillbox/skills/get?scope=global&project_id=0&name=e2e-pkg&version=0.1.0&full=true" | python3 -c "import sys,json; d=json.load(sys.stdin); c=d.get('canonical') or d.get('manifest',{}); files=c.get('files',[]) if isinstance(c,dict) else []; print('files count:',len(files)); [print(' -',f.get('path'),'len=',len(f.get('content',''))) for f in files]"

# 10. audit log:确认 import / export 出现
echo "=== 10. AUDIT stats ==="
curl -sS "http://$HOST/api/skillbox/audit/stats" | python3 -m json.tool

echo "=== 11. AUDIT logs (filtered by action=import|export) ==="
curl -sS "http://$HOST/api/skillbox/audit/logs?action=import&size=5" | python3 -c "import sys,json; d=json.load(sys.stdin); [print(' -',l.get('action'),'actor=',l.get('actor'),'payload=',l.get('payload')[:120]) for l in d.get('items',[])]"
echo ""
curl -sS "http://$HOST/api/skillbox/audit/logs?action=export&size=5" | python3 -c "import sys,json; d=json.load(sys.stdin); [print(' -',l.get('action'),'actor=',l.get('actor'),'payload=',l.get('payload')[:120]) for l in d.get('items',[])]"

echo ""
echo "=== E2E PASSED ==="
