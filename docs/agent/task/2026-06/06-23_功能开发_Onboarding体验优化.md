# Onboarding 体验优化 + bug 修复

**日期:** 2026-06-23
**状态:** 已完成

## 1. 需求

用户对首次配置 / Onboarding 界面提了 6 点优化:

1. **改左侧导航文案**:当前叫"首次配置",改成"导入技能"。
2. **去掉 3 步状态指示器**:点击 onboarding 导航后直接进入扫描结果列表(phase2),不要先走 phase1(状态)+ 3 个 step 状态。
3. **修复 tool tab 名称缺失 bug**:当 adapter 找不到工具目录时,scan report 中 `tool_name` 为空,导致前端 tool tab 名字为空,只剩图标和数量 0。
4. **扫描结果列表显示 skill 标题**:从 skill 的 md 文件中读取首行 `#` 作为标题展示。
5. **路径显示改为文件夹图标**:
   - 之前:显示完整路径
   - 现在:显示一个文件夹图标,鼠标 hover 显示完整路径(title 提示)
   - 点击图标 → 调用系统文件管理器打开该目录
6. **重复 skill 检测与互斥**:
   - 当前客户端 store 中已存在同名 skill → 置灰不可导入 + 标注"客户端已存在"
   - 跨工具同名 skill(比如 claude 和 codex 都装了同一个 skill)→ 选中一个后,另一个不可选

## 2. 任务列表

- [x] #5 优化 OnboardingView 导航,移除 3 步状态指示器
- [x] #4 修复 tool tab 名称缺失 bug(adapter 找不到工具目录)
- [x] #1 扫描结果列表显示 skill 标题(读 md)
- [x] #2 路径显示改为文件夹图标 + hover 路径 + 点击打开
- [x] #3 重复 skill 检测与互斥(客户端已存在/跨工具同名)
- [x] 任务总结 + memory 更新

## 3. 执行进度

- 14:00 阅读 OnboardingView.vue + onboarding 控制器 + skillstore + platform
- 14:05 制定任务列表 + 任务文件
- 14:30 #4 完成(后端 importer.go 过滤 Tools 字段 + FoundSkill.ToolName 兜底)
- 15:00 #5 完成(前端 phase 默认 'scan',3 步指示器删了,phase1 section 删了)
- 15:30 #2 #1 #3 完成(platform.fs 抽象 + cdesktop fs 端点 + pkg/fsutil)
- 16:00 OnboardingView 模板与样式完成(多行布局、tag、disabled、SkillTitle)
- 16:30 提交 4 个 commit,构建通过

## 4. 问题与方案

> 开发中遇到的非平凡问题(>5 分钟定位或设计取舍)。

### 问题 1:跨 module 共享代码 — pkg/fsutil 路径选择

桌面端 desktop 包和后端 cdesktop 都需要读文件 + reveal,一开始想把 fsutil 放在
`ginp-api/internal/fsutil` (跨包共享的 internal),结果 Go 规则不允许
`internal/` 跨 module 共享 — api-server (module ginp-api) 内的 internal 不能被
desktop (module skill-box) 引用。

最终方案:`pkg/fsutil` 放在 root module 下的 pkg/ 平面,两个 module 都能
import(`ginp-api/...` 通过 `skill-box/pkg/fsutil`,反之亦然)。参考根 module
go.mod 是 `skill-box`。

### 问题 2:Tool tab 名称缺失 bug 根因

用户描述:"找不到工具目录时,tab 中工具名称不显示,只显示图标和数量 0"。

根因:`foundByTool` computed 为 `scanReport.tools` 列表里的 toolID 都建空组
(name=''),如果某 tool 没有任何 found(目录没找到),组内 name 一直是空。
后端 `r.Tools` 总是 append 所有 adapter,所以 0 命中的 tool 也会出现在
envelope.tools。

修复:
- 后端:排序前过滤 `r.Tools`,只保留有 found 命中的 toolID
- 前端:toolTabs.name 兜底用 toolId(极端防御)

### 问题 3:跨工具同名互斥的判定粒度

需求:不同工具有同名 skill,选一个后另一个不可选。

判定粒度决策:用 (name + version) 作为互斥 key,不用 (name)。
理由:claude 有 foo@1.0,codex 有 foo@2.0,语义上是两个不同的 skill,应允许同时
勾选导入(导入后会按 version 落地到不同目录)。"已存在"判定用 name 不分 version
(因为 store 里 name 是 unique key)。

## 5. 需求回流

> 用户临时加塞 / 计划外需求。

无新需求。

## 6. 总结

### 完成了什么
- 后端 importer.go 过滤 0 命中 tool + FoundSkill.ToolName 兜底
- cdesktop 新增 /api/desktop/fs/{read-text,reveal} 端点
- 新建 pkg/fsutil(ReadText + Reveal 跨平台实现)
- desktop/wails_app.go 注入 FsReadText / FsReveal hooks
- 前端 OnboardingView 重写 phase2:三行布局(name/ver + title + path-btn)
- 新组件 SkillTitle.vue(按需拉,带 reqId 防过期)
- platform.fs 抽象(readText / reveal)
- 重复检测:客户端已存在标签 + 跨工具互斥
- i18n 加新 key,导航文案改"导入技能"

### 留下了什么
- 4 个 commit(后端工具链 → fsutil → 前端 OnboardingView → 任务文件)
- 新组件 `frontend/src/components/SkillTitle.vue`
- 新 package `pkg/fsutil`

### 留给下次的事
- 桌面端 reveal 在 Linux 上目前只打开父目录,后续可以加 dbus 接口做
  "reveal" 体验
- "已存在"判定目前只看 store 名字(不分 version),如果 store 改成
  (name, version) 作为 unique key,需要同步调整

### 复盘
- 好:bug 修复两边都兜底(后端过滤 + 前端兜底),不依赖单一防线
- 好:跨 module 代码用 pkg/ 共享,避开 internal/ 限制
- 待改进:第一次尝试把 fsutil 放在 `ginp-api/internal/fsutil` 浪费 2 分钟,
  应该一开始就确认模块关系

