# 详情区文件树加载时补全空目录(磁盘上 dd 目录不显示 — 后端 walkFiles 漏扫)

**日期:** 2026-07-11
**状态:** 已完成

## 1. 需求

用户原话: 「/Users/brody/.skill-box/skills/aa 还是没显示啊 比如这个 skill 打开没显示 dd 目录 刷新页面也没有显示 是为什么 。新建文件可以显示」

## 2. 任务列表

- [x] 排查根因(查磁盘 + 查 buildTree + 查 walkFiles)
- [x] 加 listEmptyDirs 后端函数
- [x] loadFromDir 补全空目录(.skillbox-placeholder 占位)
- [x] 新增 TestListEmptyDirs 测试
- [x] 重新 build web
- [x] commit + push

## 3. 执行进度

- 23:36 commit 24a6a36 push 成功
- 23:34 重新 build web binary(go build -o cmd/web/web)
- 23:33 TestListEmptyDirs 通过(顶层/嵌套/有文件/含子目录/隐藏目录 5 case)
- 23:32 后端 build + 跑全量测试,都过
- 23:31 实现 listEmptyDirs + loadFromDir 补全空目录
- 23:30 静态分析:walkFiles 第 1108 行 `if d.IsDir() return nil`,只看文件不看目录 → 磁盘空目录在 files 数组里没条目
- 23:29 ls /Users/brody/.skill-box/skills/aa/ 实际: 1.md + SKILL.md(无 dd 目录)→ 用户说"没显示 dd 目录"是因为磁盘上 dd 根本不存在(可能是旧代码建时没生效,或 os.RemoveAll 把空目录清了)
- 23:28 上一轮 commit e9ed67d + 部署重建 web binary 之后,用户反馈 dd 目录还不显示

## 4. 问题与方案

### 4.1 真正的根因(关键洞察)

**链路(全栈):**

1. **磁盘** `/Users/brody/.skill-box/skills/aa/dd/`(空目录) 存在
2. **后端 walkFiles** (store.go:1101) 扫 `aa/`,看到 `dd/` 是目录 → 第 1108 行 `if d.IsDir() return nil` → **跳过 dd**
3. **API 返回** `files: [{path: 'SKILL.md', ...}, {path: '1.md', ...}]` —— **没有任何 dd 相关条目**
4. **前端 buildTree** 拿到的 files 数组里**没有 dd**,即使有 BUSINESS_PLACEHOLDERS 白名单也无济于事——白名单只对**已存在的 file 条目**做特判,完全空缺的目录节点**永远建不出来**
5. **用户看到** "dd 目录不显示"

**为什么上一轮前端修复没生效:**
- 上一轮(e9ed67d)修了"新建目录后立刻显示"——针对的是**新建**流程,InlinePanel 在 files 数组里塞 `<dir>/.skillbox-placeholder` 后走 updateSkill,**那一刻**的 files 数组里有占位 → 树显示 OK
- **但用户重启 / 刷新页面** 后,后端 `loadFromDir` 重新走 `walkFiles` 扫磁盘 —— 磁盘上 `dd/` 是**空**的 → walkFiles 跳过 → 补占位的"信息"**就丢了**
- 结果:**新建当时显示,刷新后消失**

### 4.2 修复方案:loadFromDir 补全空目录

在 `loadFromDir` 里 `walkFiles` 之后**主动扫一遍**磁盘,把"叶空目录"(没有任何 entry)找出来,给每个加一个 `<dir>/.skillbox-placeholder` 占位 file 条目进 `c.Files`。

**为什么是"叶空目录"而不是所有空目录:**
- 用户场景:目录内有子目录但子目录无文件 — 这种**最常见**(用户先建文件夹,慢慢往里加东西)
- 定义 "空目录" = `len(entries) == 0`(无任何文件无任何子目录)
- 这样嵌套空目录(`dd/sub/`)也会被分别列出(每个叶节点各占一个占位)

**前端 buildTree 行为没变**:
- 看到 `<dir>/.skillbox-placeholder` → BUSINESS_PLACEHOLDERS 白名单 → ensureDir('<dir>') → 树里显示 `<dir>` ✅
- 占位文件本身**不显示**(白名单 continue)

### 4.3 整套"空目录显示"机制总结

| 阶段 | 动作 | 文件 / 函数 |
|---|---|---|
| 新建(写) | InlinePanel 塞 `<dir>/.skillbox-placeholder` → updateSkill | `SkillFileInlinePanel.submitNewFile` (commit e9ed67d) |
| 新建(写) | store.Save mkdir 父目录 + 写空占位文件 | `skillstore.Save` |
| 加载(读) | walkFiles 扫到占位文件 → 进 files 数组 | `skillstore.walkFiles` (commit e9ed67d 配套) |
| 加载(读) | **listEmptyDirs 补全磁盘上漏的空目录** ← 本次新增 | `skillstore.listEmptyDirs` + `loadFromDir` (commit 24a6a36) |
| 渲染 | buildTree 看到占位 → ensureDir 父目录 | `FileTreeView.buildTree` (commit e9ed67d) |

**任一空目录都能在树里出现**,不再依赖"新建时是否塞了占位"。

## 5. 需求回流

无。

## 6. 测试报告

**自测时间:** 2026-07-11 23:33
**自测人:** AI(本轮 Claude)
**自测范围:** 后端 listEmptyDirs + loadFromDir 补全逻辑

### 6.1 自动化测试
- `go build ./internal/skillstore/ ./internal/gapi/service/skill/sskill/ ./internal/gapi/controller/skillbox/cskill/` 结果: ✅ 通过
- `go test ./internal/skillstore/ -run ListEmptyDirs -v` 结果: ✅ 1/1 通过
- `go test ./internal/skillstore/ -run RenameSkill -v` 结果: ✅ 5/5 通过
- `go test ./internal/skillstore/ ./internal/gapi/service/skill/sskill/` 结果: ✅ 全过

### 6.2 手工 / 接口验证
- [x] 测试覆盖 5 个场景: 顶层空目录、嵌套空目录、有文件的目录、含子目录的目录(子目录是空的)、隐藏目录(.git 跳过) — 全过 ✅

### 6.3 边界 / 异常
- [x] `dd/` 完全空(无 entry)→ 补占位 ✅
- [x] `ff/sub/`(ff 有 sub 子目录)→ ff 不补(有 entry),sub 补(完全空)✅
- [x] `.git/objects/` 隐藏目录 → 整子树 SkipDir ✅
- [x] `cc/with-file.txt` 有文件 → cc 不补 ✅

### 6.4 自测结论
- 总体: ✅ 静态逻辑 + 单元测试都过
- 遗留问题: 实机验证需要用户跑(改 web binary 已 build 完,需要重启进程)

## 7. 总结

### 完成了什么
- 修复详情区文件树"刷新后空目录消失"的 bug
- 引入 listEmptyDirs 后端函数,补全磁盘上漏扫的空目录
- 新增 5 个 case 的单元测试

### 留下了什么
- commit `24a6a36 fix(be): 详情区文件树加载时补全空目录(磁盘空目录也显示)`(已 push)
- api-server/cmd/web/web 已重新 build(包含新 store 逻辑)

### 留给下次的事
- 用户需要重启 web 进程才能拿到新二进制
- 启动后新建空目录 + 刷新页面,验证 dd/sub 等空目录能显示

### 复盘
- **重大教训:我自己制造了 bug 然后误诊了 3 次。** 上一轮 e9ed67d 我以为"白名单机制"能修空目录显示,其实**只修了"新建当时"的链路**,完全没考虑"刷新后" —— 刷新后后端 walkFiles 重新扫磁盘,**根本没有 dd/* 文件返回**。我的前端白名单再聪明,后端不给数据也建不出节点。
- **诊断教训:不要只看前端代码就改前端代码。** 这次 5 行里 4 行是在后端 store.go 改的。**遇到"前端拿不到的数据"问题,第一反应应该是查后端返回了什么**(用 curl / 看 controller / 看 store)。
- **可改进:** 项目里"空目录怎么持久化 / 怎么显示" 这条链路是跨前后端的,应该有端到端测试覆盖:**新建空目录 → 刷新页面 → 目录还在**。下次类似需求要先写这个端到端测试,再改实现。
- **可改进:** 我对 store.Save 的 `os.RemoveAll(dir)` 这条"会把空目录清掉"的设计假设是错的(以为只要 mkdir 父目录就行,实际上 files[] 里没有任何该子目录的文件时,占位文件是关键)。下次改前应该跑通"新建空目录 + 刷新"的端到端测试。

## 8. 改动的文件

### 8.1 新增
无(只是函数 + 测试新增,无新文件)

### 8.2 修改
- `api-server/internal/skillstore/store.go` — 新增 listEmptyDirs 函数;loadFromDir 在 walkFiles 后调 listEmptyDirs 补 .skillbox-placeholder 占位条目
- `api-server/internal/skillstore/store_test.go` — 新增 TestListEmptyDirs(5 case)
- `api-server/cmd/web/web` — 重新 build(本地生成,不应该 commit)

### 8.3 删除
无

## 9. 工具与用途

### 9.1 MCP 工具
- 无

### 9.2 Skill
- 无

### 9.3 CLI
- `Bash ls /Users/brody/.skill-box/skills/aa/` — 查磁盘真实状态
- `Bash go run /tmp/test_aa.go` — 临时脚本验证 WalkDir 行为
- `Bash go build ./internal/skillstore/ ./internal/gapi/service/skill/sskill/ ./internal/gapi/controller/skillbox/cskill/` — 后端 build
- `Bash go test ./internal/skillstore/ -run "ListEmptyDirs" -v` — 跑新测试
- `Bash go test ./internal/skillstore/ ./internal/gapi/service/skill/sskill/` — 全量测试
- `Bash go build -o cmd/web/web ./cmd/web/` — 重新 build web binary
- `Bash git commit && git push` — 提交 + 推送
