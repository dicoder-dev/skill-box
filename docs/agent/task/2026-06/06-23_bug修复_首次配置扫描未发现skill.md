# Bug 修复:首次配置扫描未发现本机已装的 skill

**日期:** 2026-06-23
**状态:** 已完成

## 1. 需求
首次配置界面"开始扫描"按钮触发后,扫描报告 `found` 数量为 0,无法发现本机已经安装的 skill。需要让扫描能正确发现本机所有 skill。

## 2. 任务列表
- [x] 定位扫描主入口:PostOnboardingScan → skillimporter.Scan → BaseAdapter.Scan
- [x] 复现并列出根因(隐藏目录 / 嵌套 / symlink)
- [x] 修复 BaseAdapter.Scan:递归扫描 + 不跳过隐藏目录 + 跟随 symlink
- [x] 验证本地扫描结果能匹配实际 skill 数量(0 → 74)
- [x] 提交 git(commit: c2c7512)

## 3. 执行进度
- HH:MM 复现:扫描后 `report.FoundSkills` 长度=0
- HH:MM 定位:`api-server/internal/skilladapter/base.go:50-82` BaseAdapter.Scan 实现缺陷
- HH:MM 三个具体问题:
  1. **跳过隐藏目录** (line 65-67):`strings.HasPrefix(e.Name(), ".")` 会跳过 `.system`(5 个 skill)和 `.curated`(20+ skill)
  2. **只扫描一层** (line 50-82 的 `os.ReadDir`):Claude marketplaces 路径 `~/.claude/plugins/marketplaces/<m>/plugins/<p>/skills/<n>/SKILL.md` 是 4 层嵌套,扫不到
  3. **不跟随 symlink** (line 61 `!e.IsDir()`):Trae 的 skill 全部是 symlink → `../../.agents/skills/xxx`,`e.IsDir()` 对指向目录的 symlink 返回 false,会被跳过
- HH:MM 改写:递归 walkSkills + 跟随 symlink + EvalSymlinks 去重 + maxScanDepth=8
- HH:MM 新增 5 个单元测试(隐藏目录/嵌套/symlink/最大深度/元数据文件跳过)
- HH:MM 真实目录端到端验证:trae 5 + codex-.system 5 + codex-.curated 39 + claude-marketplaces 25 = 74 个

## 4. 问题与方案
> 现象 → 定位 → 方案 → 教训

**现象**:用户本机装了 ~40 个 skill(codex 25+, trae 5, claude 10+),扫描全为 0。

**定位**:BaseAdapter.Scan 的实现做了三个"看似安全"的判断:
```go
if !e.IsDir() { continue }                        // 漏 symlink
if strings.HasPrefix(e.Name(), ".") { continue }  // 漏 .system/.curated
// os.ReadDir 只读一层                                       // 漏嵌套
```

**方案**:把 BaseAdapter.Scan 改为基于"找含 SKILL.md 的目录"的递归扫描:
- 入口是 DiscoverPaths 给的根目录(顶层)
- 递归向下找所有子目录里**直接含 SKILL.md 文件**的目录
- symlink 解析为目录后继续递归
- 跳过 `.DS_Store` / `.marker` 这样的非目录、文件
- 仍跳过嵌套过深(>8 层)防止死循环

**教训**:写通用扫描时,"跳过隐藏目录"是个常见反 pattern;正确的处理是"找目标文件(SKILL.md)在哪"。

## 5. 需求回流
无。

## 6. 总结(任务结束时填)
- **完成了什么**:BaseAdapter.Scan 从 1 层非递归改为递归扫描 + 跟随 symlink + 允许隐藏目录,5 个新测试覆盖。
- **留下了什么**:commit c2c7512;base.go 重写 + base_test.go 新建;真实目录验证脚本已删除(临时)
- **留给下次的事**:
  - Claude marketplaces 下同名 skill 出现 3 次(access/configure),因为不同 plugin 下名字相同。是否要按 plugin 来源去重?目前按 (tool_id, name) 去重,import 阶段会冲掉,需要决策。
  - `importResult.source_path` 当前显示的是 `<filepath>/<name>`,嵌套时是冗长全路径,UI 截断即可。
- **复盘**:
  - 做得好的:从用户报修直接看到 3 个"看似安全"的过滤都出问题,根因定位非常快。
  - 能改进的:Claude marketplaces 的 4 层嵌套其实是"在 adapter 层拆 3 个 DiscoverPaths"也能搞定,但递归更通用,新工具接入时不用再改 BaseAdapter。

