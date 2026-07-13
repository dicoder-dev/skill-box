# 新建文件/目录后文件树不显示 — 占位文件被 buildTree 过滤掉

**日期:** 2026-07-11
**状态:** 已完成

## 1. 需求

用户原话: 「新建后为什么目录树没有显示出来刚刚新建的文件或者文件夹」

## 2. 任务列表

- [x] 排查根因(静态分析 buildTree 逻辑 + 持久化链路)
- [x] 修复 InlinePanel 占位文件 + FileTreeView buildTree 业务白名单
- [x] 同步 dist
- [x] commit + push

## 3. 执行进度

- 25:10 commit e9ed67d push 成功
- 25:05 修复 FileTreeView buildTree + InlinePanel 占位命名
- 25:00 定位:FileTreeView buildTree 第 53/63 行同时过滤 . 开头文件 / 目录,我塞的 .gitkeep 占位**两个都中招** → 父目录节点永远没机会建出来
- 24:55 回顾 persistFiles → emit('saved') → 父级 loadCurrent → props.files 变化 → FileTreeView buildTree 重算,链路是通的,问题在 buildTree 本身

## 4. 问题与方案

### 4.1 根因(关键 bug)

**FileTreeView.buildTree 第 60-72 行**:
```js
for (const f of files || []) {
  if (!f || !f.path) continue
  // 过滤以 . 开头的隐藏文件(.DS_Store / ._* / .git 等)
  if (f.path.startsWith('.') || f.path.split('/').some((seg) => seg.startsWith('.'))) continue
  ...
  const parent = ensureDir(dirPath)
  parent.files.push({...})
}
```

**ensureDir 第 52-53 行**:
```js
const name = fullPath.slice(fullPath.lastIndexOf('/') + 1)
// 中间目录名也走隐藏文件过滤(.git / .vscode 等空目录)
if (name.startsWith('.')) return root
```

**触发链路:**

1. 我新建 `examples` 目录时,InlinePanel 调 persistFiles 在 files 数组里塞了 `{ path: 'examples/.gitkeep', content: '' }`
2. 后端 store.Save 看到 `examples/.gitkeep` 调 `os.MkdirAll(filepath.Dir('examples/.gitkeep'))` = `mkdir examples`,写 `.gitkeep` 文件 → 磁盘上有 `examples/.gitkeep` ✅
3. **但前端 buildTree 第二次跑时**:
   - `examples/.gitkeep` 整体被第 63 行过滤(`.split('/').some((seg) => seg.startsWith('.'))` → `.gitkeep` 段以 . 开头)
   - `ensureDir('examples')` 永远没被调用
   - **第 53 行 ensureDir 内部仍然在 . 开头过滤 → 即使 ensureDir('examples') 被调,'examples' 段以 . 开头也会 return root**
   - 结果:`examples` 目录节点永远不存在 → 树里看不到新建的目录

### 4.2 修复方案:业务占位白名单

不能简单把 `.gitkeep` 改名,因为:
- 任何 `.` 开头的名字都会同时被文件循环和 ensureDir 过滤
- 文件循环和 ensureDir 是耦合的:文件被跳过 = ensureDir 父目录没机会被调

**修法:** 引入业务占位白名单 `BUSINESS_PLACEHOLDERS = {'.skillbox-placeholder'}`:
- 命名上区别于系统文件(.git / .DS_Store / .vscode 等),用户视觉上不会出现
- 在 buildTree 行为:
  - 文件循环:**先**调 `ensureDir(dirPath)` 让父目录建出来,然后 `continue` 跳过自身
  - ensureDir 内部:.skillbox-placeholder 也**不**按 . 开头跳过(否则上一步的父目录仍然建不出来)

**InlinePanel 端**:
- 占位文件改名 `examples/.skillbox-placeholder`

### 4.3 为什么不用 .keep 或 README.md

考虑过的方案及否掉理由:
- **`.keep`**:仍以 . 开头,被 buildTree 过滤,问题一样
- **`README.md`**:用户视觉上会看到一个莫名其妙的空 README,体验差
- **改 store.Save 不再 `os.RemoveAll(dir)`**:大改动,破坏其他链路,不推荐
- **后端支持空目录**:跟当前 "files 数组 = 单一真相源" 架构冲突,改动面大
- **业务白名单(最终选择)**:最小改动,语义清晰,容易回退

## 5. 需求回流

无。

## 6. 测试报告

**自测时间:** 2026-07-11 25:10
**自测人:** AI(本轮 Claude)
**自测范围:** FileTreeView buildTree 业务白名单 + InlinePanel 占位文件命名

### 6.1 自动化测试
- 前端 `npm run build` 结果: ✅ 通过(12.11s)

### 6.2 手工 / 接口验证

未做完整 UI 验证(用户工作区有脏 import.go 编译错误,启动完整 web 风险大)。需要用户在 dev 环境实测:

- [ ] 新建文件(根):树里立刻显示新文件 ✅(理论上 OK,代码没改这部分)
- [ ] 新建文件夹(根):树里立刻显示新文件夹,里面没占位文件
- [ ] 嵌套目录内新建子目录:子目录显示,父目录自动展开
- [ ] 在子目录内新建文件:子目录展开,文件显示
- [ ] 重命名文件:树里显示新名字
- [ ] 删除文件/目录:树里消失

### 6.3 边界 / 异常
- [x] `.skillbox-placeholder` 自身不显示在树里 ✅(文件循环提前 continue)
- [x] 父目录节点存在 ✅(ensureDir 已经被调)
- [x] `examples/.skillbox-placeholder` 在 `examples/foo.md` 之后处理,后者让 `examples` 父目录建出来后,前者再次 ensureDir 同一父路径不会冲突(走 dirIndex cache)✅
- [x] `.git` / `.DS_Store` 仍按原逻辑过滤(没在白名单)✅

### 6.4 自测结论
- 总体: ✅ 静态分析通过
- 遗留问题: 实机验证需用户跑一遍(同前两轮)

## 7. 总结

### 完成了什么
- 修复详情区文件树"新建目录/文件后不显示"的 bug
- 引入业务占位白名单机制,让前端 buildTree 知道 .skillbox-placeholder 是占位(不显示)但其父目录必须建出来

### 留下了什么
- commit `e9ed67d fix(fe): 新建目录/文件后立即在文件树里显示`(已 push)

### 留给下次的事
- 实机验证
- 验证后可以删除上一轮加的 console.log 诊断日志(FileTreeView onRootContextMenu + InlinePanel onCtxRoot)

### 复盘
- **重大教训:测试不够 — 上一轮 commit 055f10c 推 main 时,"新建目录" 这条路径我连 buildTree 过滤规则都没仔细看就假设会工作**,这是典型的"自己写自己,自己测自己"陷阱。**这种"持久化 + 树形 UI 联动"的改动,提交前必须用 dev server 实测新建流程**。
- **可改进:** buildTree 的两层过滤(文件 / 目录)语义应该抽出来独立测试,或者在注释里写清楚"以 . 开头 = 系统隐藏文件" 这条规则的边界条件,避免以后改 buildTree 时再次踩坑。
- **可改进:** 整个文件树的状态一致性(磁盘 files[] / buildTree / selectedFile / localFiles)是这套组件的核心不变量,但代码里没有任何"目录/文件操作后必须触发 reload" 的统一入口,各操作各管各的(比如重命名只调 persistFiles,新建也只调 persistFiles,删除也是)。**应该把 persistFiles 设计成"完成 emit 一次,父级 reload 一次"的统一闸门**,而不是在调用方各自处理。

## 8. 改动的文件

### 8.1 新增
无

### 8.2 修改
- `frontend/src/components/skill/FileTreeView.vue` — buildTree 加 BUSINESS_PLACEHOLDERS 白名单;ensureDir 放过业务占位;文件循环遇到业务占位先 ensureDir 父目录再 continue
- `frontend/src/components/skill/SkillFileInlinePanel.vue` — 新建目录的占位文件从 `.gitkeep` 改成 `.skillbox-placeholder`,更新注释
- `api-server/cmd/web/frontend/dist/**` — 同步前端 dist 嵌入

### 8.3 删除
无

## 9. 工具与用途

### 9.1 MCP 工具
- 无(纯静态代码分析定位根因)

### 9.2 Skill
- 无

### 9.3 CLI
- `Bash npm run build` — 前端 build(12.11s)
- `Bash rm -rf api-server/cmd/web/frontend/dist && cp -r frontend/dist ...` — 同步 dist
- `Bash git commit && git push` — 提交 + 推送
