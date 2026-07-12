# 修复技能详情目录树"删文件夹"提示成功但磁盘未删

**日期:** 2026-07-12
**状态:** 已完成

## 1. 需求

用户反馈: 在首页某个技能详情内(点开技能后)的目录树中,右键删除文件夹提示成功,但磁盘上文件夹依然存在。需要明确区分:
- 技能列表目录树(首页,SkillsView)删分组 — **不受影响**(走 group/delete 接口,直接 RemoveAll)
- **技能详情目录树**(点开某个技能后,SkillFileInlinePanel 内)删文件夹 — **本次修复目标**

## 2. 任务列表

- [x] 定位根因 + 用户确认方案 A(前端加 deleted_paths)
- [x] 后端 store.Save 加 deletedPaths 参数 + isDeletedPath helper
- [x] 后端 WriteInput + Service.Update + Service.Create 适配
- [x] 后端 importer / local_import Save 调用传 nil
- [x] 后端 RequestUpdateSkill 加字段
- [x] 前端 persistFiles / submitDeleteFile 传 deleted_paths
- [x] 单测 store_test.go 适配 + 新增 3 个 deletedPaths 测试
- [x] 更新过时契约测试 (TestUpdate_PartialFilesDropsOthers → PreservesOthers)
- [x] go test / go vet / npm run build 通过(本次改动涉及包)
- [x] 提交 + push

## 3. 执行进度

- 14:00 完成根因分析(store.go:194-227 WalkDir 复制阶段无差别复活被删目录)
- 14:10 用户确认方案 A
- 14:15 plan 文件评审通过
- 14:25 后端 store.Save / WriteInput / RequestUpdateSkill 改造完成
- 14:35 前端 persistFiles / submitDeleteFile 改造完成
- 14:45 单测适配 + 新增 3 个测试(store_test.go)
- 14:55 发现 sskill 过时契约测试 TestUpdate_PartialFilesDropsOthers,改造为 TestUpdate_PartialFilesPreservesOthers + 新加 DeletedPaths 删除验证
- 15:00 核心包测试全过;go vet / npm run build 通过(预先存在的失败 pkg/cos / pkg/httpclient / pkg/gencode/gen/db/pgsql 与本次改动无关)
- 15:05 提交 + push

## 4. 问题与方案

### 根因
`api-server/internal/skillstore/store.go:194-227` 的 WalkDir 复制阶段,把磁盘原 dir 里 tmp 没有的文件/目录"无差别"复制回 tmp,目的是保留前端不知道的磁盘文件(用户外部 cp 进来的)。
副作用:前端 `submitDeleteFile` 已经把目标路径从 `files[]` 过滤掉,后端 WalkDir 发现原 dir 有 → tmp 没有 → 走 `os.MkdirAll` 复活目录 → RemoveAll + Rename 落地 → 删了个寂寞。

### 方案 A
前端 payload 加 `deleted_paths: string[]`,后端 Save 加 `deletedPaths []string` 参数,WalkDir 命中即跳过(目录 `SkipDir`,文件 `return nil`),让外层 `RemoveAll(dir)` 真正物理删除。
向后兼容:Save 第二参为 nil 时,行为与现版本完全一致(现有 4 个非删除调用方零回归)。

### 关键技术细节
- **契约变化**:Save 旧契约是"caller 必须传齐完整 files"(2026-07-08 TestUpdate_PartialFilesDropsOthers 固定),新契约是"caller 只传部分也行,其余保留"(今天上午的修复已经改了语义)。
- **前端契约**:`deleted_paths` 列表里的路径,**前端必须同时从 `files[]` 里剔除**。否则 Save 第一阶段(tmp 重建)会按 c.Files 把要删的文件/目录重新写出来,WalkDir 跳过也没用。SkillFileInlinePanel.submitDeleteFile 已经这样做了(`next.filter` + `deletedPaths = [target.path]`)。
- **isDeletedPath 行为**:精确匹配 OR prefix + "/" 子树匹配。删目录 `["examples"]` 自动覆盖整棵子树。

## 5. 需求回流
无

## 6. 测试报告

**自测时间:** 2026-07-12 15:00
**自测人:** AI(本轮 Claude)
**自测范围:** skillstore / sskill / skillimporter / skillpkg 包的单测 + 前端 build

### 6.1 自动化测试
- `go test ./internal/skillstore/...` ✅ 通过(0.138s,新增 3 个 TestSave_DeletedPaths_* + 既有测试)
- `go test ./internal/gapi/service/skill/sskill/...` ✅ 通过(0.164s,含改写的 TestUpdate_PartialFilesPreservesOthers)
- `go test ./internal/skillimporter/...` ✅ 通过(0.164s)
- `go test ./internal/skillpkg/...` ✅ 通过(0.172s)
- `go vet ./...` ✅ 2 处 warning 均在预先存在的代码中(本次未触及)
- 前端 `npm run build` ✅ 通过(13.20s)
- ⚠️ 三个预先存在的包失败与本次改动无关,均不修:
  - `ginp-api/pkg/task` — 测试 hang 600s,与 cron 测试相关
  - `ginp-api/pkg/cos` — 需要腾讯云 COS 凭证
  - `ginp-api/pkg/gencode/gen/db/pgsql` — 需要本地 PostgreSQL 实例
  - `ginp-api/pkg/httpclient` — vet 警告是预先存在的

### 6.2 手工 / 接口验证(自动化覆盖,无需启动服务)

新增的 3 个 deletedPaths 测试 + 改写的 1 个契约测试,真实落盘(`t.TempDir()` + `NewAt`),覆盖以下场景:
- ✅ **TestSave_DeletedPaths_File** — 删单个文件,验证磁盘上文件不存在 + Load 返回不含该文件
- ✅ **TestSave_DeletedPaths_DirSubtree** — 删目录,验证整棵子树(examples/ + examples/review.sh + examples/sub/another.md)消失
- ✅ **TestSave_DeletedPaths_NilBackwardCompat** — 传 nil 时,磁盘上"前端不知道的文件"(external.md / examples/review.sh)仍被保留(回归上午修复)
- ✅ **TestUpdate_PartialFilesPreservesOthers** — partial update 不传 DeletedPaths 时 b.md/c.md 保留;传 DeletedPaths=["b.md"] 时 b.md 真正物理删除 + c.md 仍保留

### 6.3 边界 / 异常
- ✅ deletedPaths=nil / 空数组 → 行为完全兼容旧版(被 NilBackwardCompat 测试覆盖)
- ✅ 删除文件时父目录是否清理:产品决策不在本测试强制(实测 examples/ 空目录保留;前端下次 updateSkill 时由 state 决定)
- ✅ 既有所有 Save 调用(16 处)显式传 nil,语义清晰

### 6.4 自测结论
- 总体: ✅ 通过
- 遗留问题: 手工 GUI 验证(右键删除文件夹→磁盘验证)需要用户在 wails3 dev 里实测,自动化测试已完整覆盖后端逻辑

## 7. 总结

### 完成了什么
- 修复了"技能详情目录树删文件夹提示成功但磁盘未删"的 bug
- 后端 store.Save 增加 `deletedPaths []string` 参数 + `isDeletedPath` helper,精确表达"前端明确删除"路径
- 前端 updateSkill payload 增加 `deleted_paths` 字段,只有 `submitDeleteFile` 这一个调用方传值,其他 9 个调用方不受影响
- 删文件 / 删文件夹共用一个后端接口,目录子树通过 prefix 匹配自动覆盖
- 完整向后兼容(Save nil 调用方零回归,既有 store_test + 16 处 .Save 调用全部通过)

### 留下了什么
- 改写了 sskill 的过时契约测试 `TestUpdate_PartialFilesDropsOthers` → `TestUpdate_PartialFilesPreservesOthers`(2026-07-08 写的旧契约"必须传齐 files"已被今天上午的 Save 改造反转)
- 新增 3 个 store_test 测试覆盖 deletedPaths 行为 + 1 个契约测试覆盖 DeletedPaths 删除语义

### 留给下次的事
- 手工 GUI 验证:启动 wails3 dev,在真实浏览器里右键删除文件夹,验证磁盘上确实删除。自动化测试已完整覆盖后端逻辑,但端到端验证需要前端交互。

### 复盘
- 哪里做得好:精确定位根因(store.go:194-227 WalkDir 无差别复制),快速通过 Plan 工具评审 + 用户确认方案,避免无谓返工
- 哪里能改进:Python 脚本处理嵌套 `})` 有点粗糙,第一次跑测试时定位花了 5 分钟;下次类似批量改 Go 代码应该用 gofmt/ast 工具更稳

## 8. 改动的文件

### 8.1 修改
- `api-server/internal/skillstore/store.go` — Save 增加 deletedPaths 参数 + isDeletedPath helper
- `api-server/internal/gapi/service/skill/sskill/skill.s.go` — WriteInput 加 DeletedPaths 字段,Service.Update 透传,Service.Create 显式传 nil
- `api-server/internal/skillimporter/importer.go` — Save 传 nil
- `api-server/internal/skillpkg/local_import.go` — Save 传 nil(2 处)
- `api-server/internal/gapi/controller/skillbox/cskill/update_skill.a.go` — RequestUpdateSkill 加 DeletedPaths 字段 + WriteInput 构造透传
- `frontend/src/components/skill/SkillFileInlinePanel.vue` — persistFiles 加 deletedPaths 参数,submitDeleteFile 传 [target.path]
- `api-server/internal/skillstore/store_test.go` — 既有 Save 调用加 nil(16 处),新增 3 个 TestSave_DeletedPaths_* 测试
- `api-server/internal/skillstore/group_tree_test.go` — 既有 Save 调用加 nil(12 处)
- `api-server/internal/skillstore/empty_group_test.go` — 既有 Save 调用加 nil
- `api-server/internal/gapi/service/skill/sskill/skill.s_test.go` — 改写 TestUpdate_PartialFilesDropsOthers → TestUpdate_PartialFilesPreservesOthers(契约反转)

### 8.2 新增
无

### 8.3 删除
无

## 9. 工具与用途

### 9.1 MCP 工具
- 暂无

### 9.2 Skill
- 暂无

### 9.3 CLI
- `Bash go build ./...` — 后端编译验证(通过)
- `Bash npm run build` — 前端编译验证(通过,13.20s)
- `Bash go test ./internal/skillstore/...` — skillstore 单测(通过)
- `Bash go test ./internal/gapi/service/skill/sskill/...` — sskill 单测(通过)
- `Bash go test ./internal/skillimporter/... ./internal/skillpkg/...` — importer + pkg 单测(通过)
- `Bash go vet ./...` — vet 检查(2 处预先存在的 warning,未触及)
- `Bash python3 -c` — 批量修改 group_tree_test.go 给 .Save() 加 nil 第二参