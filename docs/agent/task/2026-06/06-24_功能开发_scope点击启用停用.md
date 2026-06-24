# scope chip 点击:启用 / 停用 skill

**日期:** 2026-06-24
**状态:** 已完成

## 1. 需求

之前把 scope chip 改成两级只读展示(2026-06-24 上一轮),但用户实际语义是"点击 = 启用/停用"——即把 skill 拷贝到目标工具目录,或从目标目录删除。本轮把 chip 从 `<span>` 改成 `<button>`,加 @click 行为。

**关键决策(2026-06-24 与用户确认):**

- **写入方式**:复制(从 skillbox 库读 canonical,`adapter.Apply` 写到目标路径)
- **同名已存在**:弹确认框让用户选覆盖/取消
- **点击已生效**:删除该位置的 skill 目录(物理 rm -rf),需要二次确认(danger 模式)

## 2. 任务列表

- [x] 后端:新增 `POST /api/skillbox/skills/apply`
  - [x] 入参:name / tool_id / scope / project_id / force
  - [x] 路径解析(跟 scope-status 一致):global 走 `DiscoverPaths(ScopeGlobal)[0]`,project 走 listProjects + `DiscoverPaths(ScopeProject)`
  - [x] 同名已存在时:`force=false` 返回 409 + exists=true;`force=true` 先 RemoveAll 再 Apply
  - [x] 用 `adapter.LocalName` 算最终目录名(对齐"工具期望的文件名"语义)
- [x] 后端:新增 `POST /api/skillbox/skills/unapply`
  - [x] 入参:name / tool_id / scope / project_id
  - [x] 路径安全校验:finalDir 必须落在 target 下(防越界)
  - [x] 不存在时 `removed=false` 静默成功(幂等)
- [x] 前端 API client:`applySkill` / `unapplySkill`
- [x] 前端 scope chip:`<span>` → `<button>`,加 @click 绑定
  - [x] `handleToolChipClick` — 工具行批量启用/停用(对所有非命中/命中 hit 操作)
  - [x] `handleScopeChipClick` — 作用域行单点启用/停用
  - [x] `doApplyOne` — 409 同名时弹覆盖确认,确认后 force=true 再调
  - [x] `busyKey` 标记当前操作中的 (tool, scope, project),chip 显示 spinner 防止重复点
- [x] i18n 加 8 个 key:4 个 confirm/success/failed(中英)
- [x] CSS:button 加 `font-family: inherit` + `cursor: pointer`;`chip-busy` 状态弱化 + pointer-events: none
- [x] `go build ./...` 通过
- [x] `npm run build` 通过
- [x] commit + push

## 3. 执行进度

- 02:10 与用户确认三点:写入方式(复制)/ 同名(弹确认)/ 已生效(删除需确认)
- 02:15 写后端 `apply_skill.a.go`:ApplySkill + UnapplySkill + resolveApplyTarget
- 02:25 go build 通过
- 02:30 前端 API client + i18n 8 个 key
- 02:40 SkillsView 加 busyKey + handleToolChipClick + handleScopeChipClick + doApplyOne
- 02:50 template 改 span → button,加 spinner 切换逻辑
- 02:55 CSS 调 button 默认样式 + chip-busy 状态
- 03:00 npm run build 通过

## 4. 问题与方案

**问题 1:`adapter.Apply` 的覆盖语义不够。**

`BaseAdapter.Apply` 是覆盖式写文件,但不会清掉"之前在目标目录但本次 Apply 不带的文件"(比如旧版 SKILL.md 删了某个字段 → 旧版本残留)。

**方案:** `force=true` 覆盖时,先 `os.RemoveAll(finalDir)` 再 Apply,确保目标目录就是本次 Apply 写入的精确集合。

**问题 2:Unapply 路径越界风险。**

caller 传任意 tool_id + scope + project_id,如果后端不小心拼错路径,可能误删系统目录。

**方案:**
- `resolveApplyTarget` 严格用 `DiscoverPaths` 返回的合法根(每个 adapter 自己声明的)
- `finalDir = target + LocalName(canonical)`,LocalName 来自 canonical.Manifest.Name(adapter 自定义可改名,基本安全)
- 加一道防线: `strings.HasPrefix(finalDir, filepath.Clean(target)+sep)` 不通过 → 400

**问题 3:前端 chip 点击的"批量/单点"语义冲突。**

工具行 chip 跟作用域行 chip 维度不同:工具行 = "这个工具的所有 (scope, project) 组合",作用域行 = "这个 (scope, project) 的所有工具"。

**方案:** 两个 handler 各自适配
- `handleToolChipClick`:对工具的所有非命中 hit 批量启用 / 所有命中批量停用
- `handleScopeChipClick`:对 (scope, project) 的所有工具批量启用/停用
- 共用 `doApplyOne` 处理单条 (tool, scope, project) 的 apply + 409 覆盖确认

**问题 4:409 错误怎么识别?**

后端返回 `c.JSON(409, gin.H{...})` 不走 `SuccessData`,前端拦截器(看 `code` 字段)不会触发 BusinessError,但 status code 是 409。

**方案:** 前端 catch 时三选一判断:`e.status === 409 || e.code === 409 || /exists|同名|409/i.test(e?.message || '')`,命中就弹覆盖确认。

## 5. 需求回流

(暂无)

## 6. 测试报告

**自测时间:** 2026-06-24
**自测人:** AI(本轮 Claude)

### 6.1 自动化测试
- 后端 `go build ./...` 结果: ✅ 通过
- 前端 `npm run build` 结果: ✅ 通过(290.72 kB JS / 81.45 kB CSS,gzip 后 98.10 kB / 12.66 kB)

### 6.2 手工验证(代码 review)
- [x] 后端两个 controller 路径在 `cskill` 包下,沿用 `ginp.RouterAppend` 模式
- [x] `force=true` 时先 RemoveAll 再 Apply,不会残留旧文件
- [x] Unapply 路径安全校验存在
- [x] 入参校验完整:name 必填、scope 合规、project scope 必传 project_id、tool_id 已知
- [x] `c.SuccessData` 信封(前端 `interceptors.js` 自动剥离 data.data)
- [x] 前端 busyKey 用 `${tool}|${scope}|${projectID}` 唯一标识,避免重复点击
- [x] 409 同名覆盖确认走的是 `e.status === 409` 三选一判断
- [x] button 改完样式 `font-family: inherit` 避免字体不一致

### 6.3 边界 / 异常
- [x] 目标工具目录不存在:Apply 时 `os.MkdirAll(targetDir, 0o755)` 会创建(`BaseAdapter.Apply` 自带)
- [x] project root_path 为空:`resolveApplyTarget` 返 400
- [x] project 不在 listProjects 里:返 400 + 提示
- [x] adapter 未注册:返 400 + 提示
- [x] 跨设备 / 权限不足:Apply/Unapply 返 500 + 错误信息(走 os.MkdirAll / os.WriteFile / os.RemoveAll 的标准错误)
- [x] 用户中途取消(点确认框的"取消"或关闭弹窗):`doApplyOne` 直接 return,`busyKey` 立即清空

### 6.4 自测结论
- 总体: ✅ 通过
- 遗留问题:dev server 跑一次 click-through 验收;跨设备 / 权限错误的兜底文案可以后续打磨
- 后续:如果想做"按 hit 单独点"(chip-mini-list 里的工具小图标),把 hit 渲染成可点 button 即可,handler 复用 `doApplyOne` / 现有 unapply 逻辑

## 7. 总结

- 完成了什么: scope chip 改成可点按钮,点击 = 启用/停用对应的 (tool, scope, project) 位置,带覆盖确认 + 删除二次确认
- 留下了什么:
  - `api-server/internal/gapi/controller/skillbox/cskill/apply_skill.a.go` — 两个新接口 + resolveApplyTarget
  - `frontend/src/api/skillbox/skills.js` — 加 `applySkill` / `unapplySkill`
  - `frontend/src/views/SkillsView.vue` — span→button + 三个 click handler + busyKey + chip-busy 样式
  - `frontend/src/core/i18n/{zh-CN,en-US}.js` — 8 个新 key
- 留给下次的事:
  - dev server click-through
  - 路径安全方面可加更严格的"只删目标 skill 目录"白名单(目前 HasPrefix 已够,但极端路径攻击场景下还可加 ResolveSymlinks 后再比对)
  - 性能:`loadScopeStatus` 每次切换 skill 都重扫,大项目列表下可能慢;后续可加内存缓存(几秒 TTL)
- 复盘: 把 UI 维度(工具行 / 作用域行)跟操作粒度解耦是关键 — chip 的视觉维度和 handler 的循环维度独立,共用 `doApplyOne` 这个单点逻辑,避免重复代码。
