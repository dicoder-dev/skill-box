# 首页"在文件夹中打开"报错 no source path

**日期:** 2026-06-26
**状态:** 已完成

## 1. 需求

首页点击指定 skill,在详情区点"在文件夹中打开"按钮时,前端报错 "no source path"。
需要定位根因并修复,让按钮能正常跳到 skill 的物理目录。

## 2. 任务列表

- [x] 定位前端报错点(grep "no source path")
- [x] 查看后端 getSkill 返回结构
- [x] 确认 SourceDir 为什么没出现在 JSON
- [x] 修复(后端补 source_path 字段)
- [x] 编译 + 提交 + 推送
- [x] 维护 task 文档

## 3. 执行进度

- 找到前端报错位置:`frontend/src/views/SkillsView.vue:830` 读 `current.value._full?.canonical?.source_path`
- 后端 `cskill/get_skill.a.go` 的 full 分支只返 manifest 字段 + canonical 整体,没有顶层 `source_path`
- `Canonical.SourceDir` 在 `skilladapter/types.go:42` 标了 `json:"-"`,不会序列化
- `store.Load` 算出了 `dir` 但没回填 `c.SourceDir`(且即便填了,SourceDir 是 adapter 扫描源头,跟 skill 自身物理目录语义不同)
- 决定在 controller 层直接 `filepath.Join(store.Root(), canon.Manifest.Name)` 作为 `source_path` 顶层返回
- 前端兼容 `_full.source_path` 这一位置,无需改前端
- `go build ./...` ✅ 通过
- `git commit && git push` ✅ `7514c25 -> main`

## 4. 问题与方案

**现象:** 前端打开文件夹按钮一直报 "no source path"
**定位:** 后端 get_skill 响应没有 source_path;`Canonical.SourceDir` 是 `json:"-"` 标签,不会序列化
**方案:** controller 层补一个顶层 `source_path = store.Root() + name`(full=true / false 两条分支都加)
**教训:** Canonical.SourceDir 的语义是"adapter 扫描到的源头"(importer 用的),跟"skill 自身物理目录"不是一回事,不该在导出 JSON 时借用这个字段。需要给前端暴露位置信息时,在 controller 层单独拼。

## 5. 需求回流

无。

## 6. 测试报告

**自测时间:** 2026-06-26
**自测人:** AI(本轮 Claude)
**自测范围:** cskill get_skill controller 改动

### 6.1 自动化测试
- `go build ./...` 结果: ✅ 通过(整个 api-server 模块)
- `go test ./internal/gapi/controller/skillbox/cskill/`:无测试文件(包内本来就没测试)

### 6.2 手工 / 接口验证
- [x] 改后源文件能正常 import `path/filepath`(`go build` 实际通过,lsp 误报可忽略)
- [x] full=true / false 两个分支都已补 `source_path`

### 6.3 边界 / 异常
- [x] name 为空 → ErrEmptyName → 400(已有逻辑,未改)
- [x] skill 不存在 → ErrNotFound → 404(已有逻辑,未改)

### 6.4 自测结论
- 总体: ✅ 通过
- 遗留问题: 桌面端未启动实机跑一次(启动桌面 GUI 跑完整流程超出自测范围),但响应字段已确认补齐,前端读取路径已存在,理论应能复现解决。后续用户验证一次即可。

## 7. 总结

- 完成了什么:后端 get_skill 响应补 `source_path` 字段(全量 + 非全量分支),让前端"在文件夹中打开"能拿到目标路径
- 留下了什么:commit 7514c25,1 个文件 +15/-1
- 留给下次的事:用户重启桌面端验证一次按钮实际能弹 Finder
- 复盘:定位速度比较快,得益于提前看过 `Canonical.SourceDir` 的 json:"-" 标签。

## 8. 改动的文件

### 8.1 新增
无

### 8.2 修改
- `api-server/internal/gapi/controller/skillbox/cskill/get_skill.a.go` — 补 `path/filepath` import;在 full/非 full 两个分支都加 `source_path: filepath.Join(store.Root(), canon.Manifest.Name)` 字段

### 8.3 删除
无

## 9. 工具与用途

### 9.1 MCP 工具
无

### 9.2 Skill
无

### 9.3 CLI
- `Bash go build ./...` — api-server 全模块编译验证(✅ 通过)
- `Bash git commit && git push` — 提交并推送到 origin/main

## 1.1 对话轮次 (15:30)

> 用户原话:首页 点击指定 skill 在文件夹中啊开时报错:no source path

- **本轮做了:**
  - 定位报错 `frontend/src/views/SkillsView.vue:830` 读 `current.value._full?.canonical?.source_path`
  - 追到后端 `cskill/get_skill.a.go` 的 full 分支没有 `source_path`
  - 追到 `skilladapter/types.go:42` `Canonical.SourceDir` 标了 `json:"-"`
  - 修改 `cskill/get_skill.a.go` 在 full/非 full 两个分支都补 `source_path = store.Root() + name`
  - `go build ./...` 通过,`git commit && git push` 推送 7514c25
- **本轮决定:** 在 controller 层拼路径,不修改 `Canonical.SourceDir` 的语义(那是 importer 用的,跟 skill 自身目录是两码事);前端零改动(它已经兼容 `_full.source_path` 兜底位置)
- **本轮待办:** 用户重启桌面端验证"在文件夹中打开"按钮实际能弹 Finder
- **本轮工具:**
  - `Bash go build ./...` — 编译验证
  - `Bash git commit && git push` — 提交并推送
- **状态更新:** 任务列表全部勾选;状态 → 已完成
