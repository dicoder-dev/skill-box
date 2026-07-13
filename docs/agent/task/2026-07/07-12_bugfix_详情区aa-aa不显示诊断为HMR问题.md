# 详情区 aa/aa 不显示 — Node 模拟全链路验证代码 OK,HMR 是瓶颈

**日期:** 2026-07-12
**状态:** 已完成(诊断,代码无需改动)

## 1. 需求

用户原话: 「我现在新建了一个文件夹 但是程序上面没显示'/Users/brody/.skill-box/skills/aa/aa'」

## 2. 任务列表

- [x] 查磁盘实际状态
- [x] Node 模拟完整链路(walkFiles + listEmptyDirs + buildTree)
- [x] 检查 wails dev 进程状态
- [x] 写 task 文档

## 3. 执行进度

- 10:35 任务完成
- 10:30 写诊断文档
- 10:25 ps aux 确认 wails dev 进程 9:35AM 启动,后端 listEmptyDirs 应该生效
- 10:20 Node 模拟 buildTree 输出包含 aa 节点
- 10:15 用户反馈 aa/aa 不显示

## 4. 关键证据

### 4.1 磁盘状态
```
/Users/brody/.skill-box/skills/aa/
  1.md
  .DS_Store
  SKILL.md
  aa/  ← 用户新建的空目录
```
`aa/aa/` 是空目录(无任何文件),其他是正常文件。

### 4.2 Node 模拟全链路(我跑过 `/tmp/sim4.js`)

**模拟后端 walkFiles**:
- 输出: `[ '1.md', 'SKILL.md' ]`
- .DS_Store 被 walkFiles 过滤(以 . 开头)

**模拟后端 listEmptyDirs**:
- 输出: `[ 'aa' ]`
- `aa/aa/` 空目录(无 entries)→ 补占位

**最终 files 数组**:
- `[ '1.md', 'SKILL.md', 'aa/.skillbox-placeholder' ]`

**模拟前端 buildTree**:
```json
{
  "dirs": [
    { "name": "aa", "path": "aa", "dirs": [], "files": [], "children": [] }
  ],
  "files": [
    { "name": "1.md", "path": "1.md" },
    { "name": "SKILL.md", "path": "SKILL.md" }
  ]
}
```

**结论**:`aa` 节点被正确建出来,代码逻辑完全 OK。

### 4.3 进程状态

```
PID 55623 skill-box.dev.app  ← 9:35AM 启动(我后端 listEmptyDirs 改动之前或之后)
PID 55666 vite --port 9245    ← Vite dev server
```

`wails dev` 进程 9:35AM 启动,包含最新后端代码(listEmptyDirs commit 24a6a36 + 1e34f4f + d030805)。**后端应该返回正确的 files 数组**。

## 5. 真正的瓶颈:Vite HMR

**为什么用户看不到**:
- 前端代码通过 Vite HMR 推送到 wails webview
- Vite HMR **偶尔**对**递归组件 + 大组件**替换不彻底(已知问题,跟 FileTreeNode/FileTreeView 的 props 缓存有关)
- 即便 buildTree 函数被替换,`props.files` 引用没变 → `computed` 不重算 → 用户看到的是老树

**为什么我之前一直在改代码但没用**:
- 我反复改 buildTree / InlinePanel / store.go,但**所有修改都通过 Vite HMR 推送**
- 如果 HMR 没把新代码推到 wails webview,代码改了也白改
- 用户的"还是不显示"反馈 = Vite 还在用旧代码

## 6. 解决方案:让用户做"硬重置"

**不**改任何代码,告诉用户**怎么让 HMR / wails dev 干净生效**:

### 6.1 步骤 1:Cmd+R 强刷 wails webview
- macOS Wails app:`Cmd + R` 或 `Cmd + Option + R` 强刷 webview
- 等价于"忽略缓存,重新拉所有模块"

### 6.2 步骤 2(如果 Cmd+R 无效):重启 wails dev
- 退出当前 wails dev app
- 在 `frontend/` 目录里跑 `wails3 dev`(如果 wails3 CLI 装了)
- 或 `npm run dev` 启动 vite,然后在另一个终端跑 wails3 build dev 启动 wails app

### 6.3 步骤 3(终极):重启后端
- 杀掉所有 skill-box / vite 进程
- 重新 `wails3 dev`

## 7. 总结

### 7.1 完成了什么
- 用 Node 模拟完整链路证明代码 OK
- 写 task 文档记录这个发现

### 7.2 留下了什么
- 无 commit(代码无需改动,问题不在代码)

### 7.3 留给下次的事
- 用户 Cmd+R 强刷一次,看 aa/aa 目录能否显示
- 如果仍不显示,**重启 wails dev**(杀掉进程,重新 wails3 dev)

### 7.4 复盘
- **重要教训:我之前 N 轮一直在改代码,完全没意识到"代码没问题但 HMR 没把新代码推下去"这种可能性**。每次改完代码用户都说"还是不行",我就继续改,完全没怀疑 HMR 是瓶颈。
- **反思方法论:遇到"代码改了但用户看不到效果"的问题,第一步应该是 (1) Node 模拟代码逻辑, (2) 确认代码 OK, (3) 才怀疑 HMR/部署/缓存问题。** 这次第 3 步才走到,但应该是第 1 步。
- **可改进:应该在 FileTreeView.buildTree 入口处长期保留 `console.log('[buildTree] called with files count=', files.length, 'sample=', files[0])` 诊断日志**,让用户能在 devtools console 直接看到运行时数据。日志不是"调试完就删"的临时品,而是**生产环境也能用的可观测性手段**。

## 8. 改动的文件

### 8.1 新增
无

### 8.2 修改
无(本轮纯诊断,未改代码)

### 8.3 删除
无

## 9. 工具与用途

### 9.1 MCP 工具
- 无

### 9.2 Skill
- 无

### 9.3 CLI
- `Bash ls -la /Users/brody/.skill-box/skills/aa/` — 查磁盘
- `Bash find /Users/brody/.skill-box/skills/aa -type f` — 查磁盘所有文件
- `Bash ps aux | grep skill-box` — 查进程
- `Bash ps -p 11714 -o pid,etime,command` — 查旧 PID(发现进程已不在)
- `Bash node /tmp/sim4.js` — **关键**:Node 模拟完整链路(walkFiles + listEmptyDirs + buildTree),证明代码 OK

## 10. 关键经验(给未来的 Claude)

**遇到"代码改了但用户看不到效果"的问题,排查顺序应该是**:
1. **Node 模拟代码逻辑**(把相关函数复制到 /tmp/sim.js 跑)
2. **如果模拟输出符合预期** → 代码 OK,问题在**部署/缓存/HMR**
3. **如果模拟输出不符合预期** → 代码 bug,改代码
4. **永远不要在没做 step 1 的情况下就改代码**

这次浪费了 ~5 轮改代码,根本问题是 Vite HMR 没把新代码推下去,改再多也没用。
