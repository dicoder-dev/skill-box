# toolspecs — AI 编程工具元数据(DB 驱动版)

> 给"接一个新 AI 编程工具"这件事准备的一份速查手册。
> **2026-06-30 二改**:工具元数据从 `specs/*.yaml` embed 改成 **e_tool + e_tool_path 数据库表**,
> 新加工具 = 在前端 UI 加一行(或改 paths),不需要改 Go 代码、不需要 build。

---

## 目录

| 路径 | 作用 |
| --- | --- |
| `schema.go` | `ToolSpec` / `ToolPaths` / `CategoryPaths` 结构体定义 + 校验(转换器,运行时用) |
| `dbload.go` | `LoadAllFromDB(db)` — 查 e_tool + e_tool_path 拼成 `[]*ToolSpec` |
| `dbload_registry.go` | `ReloadAllFromDB(db)` — 把全部 spec 转成 BaseAdapter,刷到 `DefaultRegistry` |
| `specadapter.go` | `NewSpecAdapter(spec)` — ToolSpec → `BaseAdapter` 工厂 |
| `loader_test.go` | `TestSpecAdapter_*` + `TestToolSpec_Validate` 纯逻辑测试 |

> 历史文件:`loader.go` / `registry.go` / `specs/*.yaml` 已删除(2026-06-30 二改)。

---

## 数据流

```
启动期:
  cmd/bootstrap/start_db.go
    → AutoMigrate(e_tool, e_tool_path)        // 创建/更新表
    → toolseed.EnsureSeeded(db)                // 全新 DB 写 9 个默认工具
    → toolspecs.ReloadAllFromDB(db)            // 拉一次注册到 DefaultRegistry

运行时(用户在 UI 改了工具):
  frontend POST /api/skillbox/tools/update
    → ctool.UpdateTool
      → stool.Service.Update
        → mtool.Update(...) + replacePaths(...)
        → ctool 返回成功
  frontend POST /api/skillbox/tools/reload
    → stool.Service.Reload
      → toolspecs.ReloadAllFromDB(db)
        → DefaultRegistry().Reload(adapters)  // 整体替换,旧 tool_id 真的没了
```

---

## e_tool + e_tool_path schema

### e_tool(主表)

| 字段 | 类型 | 约束 |
| --- | --- | --- |
| `id` | uint | PK |
| `tool_id` | varchar(32) | **uniqueIndex** — 全局唯一,业务上不可改 |
| `display_name` | varchar(64) | UI 展示名 |
| `mdi_icon` | varchar(64) | 前端 mdi 图标(mdi:xxx) |
| `maturity` | varchar(16) | stable / experimental / deprecated |
| `note` | text | 自由文本 |
| `is_system` | bool | seed 出的系统工具(不可删 / 不可改 tool_id) |
| `enabled` | bool | 全局开关;false 时 Reload 跳过 |
| `sort_order` | int | 列表展示顺序 |
| `created_at` / `updated_at` | timestamp | GORM 自动 |

### e_tool_path(子表,一对多)

| 字段 | 类型 | 约束 |
| --- | --- | --- |
| `id` | uint | PK |
| `tool_id` | uint | 逻辑外键 → e_tool.id,删 tool 级联 |
| `scope` | varchar(16) | global / project |
| `category` | varchar(16) | user / system |
| `path` | varchar(512) | 绝对或相对路径(含 ~/),运行时展开 |
| `path_order` | int | 同一 (scope, category) 内的顺序 |

**唯一索引:** `(tool_id, scope, category, path)` — 防重复。

---

## 业务约束(由 stool 服务层保证)

| 场景 | 行为 |
| --- | --- |
| 用户新建 tool | `is_system` 强制 false |
| 改系统工具的 `tool_id` | 后端忽略(本字段不是 UpdateInput 的一部分) |
| 删系统工具 (`is_system=true`) | **拒绝**,返回 400 `ErrSystemToolFrozen` |
| 删用户工具 | 事务里级联删 e_tool_path,不留悬空 |
| 改 path | `UpdateInput.Paths` 非 nil = 覆盖式替换;nil = 不动 |
| 加单条 path | `POST /api/skillbox/tools/paths/add`,追加不覆盖 |
| `tool_id` 重复 | 拒绝,返回 409 `ErrToolIDConflict` |
| `mdi_icon` 不是 `mdi:` 开头 | 拒绝,返回 400 `ErrEmptyMdi` |
| `maturity` 不是 stable/experimental/deprecated | 拒绝,返回 400 `ErrBadMaturity` |
| 路径 scope 不是 global/project | 拒绝,返回 400 `ErrBadScope` |
| 路径 category 不是 user/system | 拒绝,返回 400 `ErrBadCategory` |

---

## 启动期 seed

`internal/toolseed/builtin.go` 内置 9 个默认工具的 Go 常量(原 yaml 内容)。
启动期(全新 DB,e_tool.Count == 0)自动写入,事务内保证一致性。

| 工具 | tool_id | maturity | sort_order |
| --- | --- | --- | --- |
| Claude Code | `claude` | stable | 10 |
| Codex | `codex` | stable | 20 |
| Cursor | `cursor` | stable | 30 |
| OpenCode | `opencode` | stable | 40 |
| Trae | `trae` | stable | 50 |
| Antigravity | `antigravity` | stable | 60 |
| Cline | `cline` | stable | 70 |
| CodeBuddy | `codebuddy` | **experimental** | 80 |
| JetBrains AI | `jetbrains` | **experimental** | 90 |

**判定语义:** `e_tool.Count() == 0` → seed;`> 0` → 跳过(已初始化过)。
不区分"系统" / "用户"行 — 全新 DB seed 后含 9 个;用户加的工具
意味着 DB 早就被 seed 过了。

**幂等性:** Count==0 时重跑 seed 是幂等的(无主键冲突)。
**无历史遗留:** 项目未发布,seed 不考虑"老数据迁移"。

---

## 改一个新工具 — Step by Step

### 方式 1(用户友好):前端 UI

1. 进 Settings → 工具管理
2. 点"新建工具",填:
   - `tool_id`: 全小写,字母数字 + `-` / `_`
   - `display_name`: 任意(前端 i18n 可覆盖)
   - `mdi_icon`: 查 https://pictogrammers.com/library/mdi/
   - `maturity`: stable / experimental / deprecated
   - `paths`: 至少 (1 个 global + 1 个 project) 各一条 user path
3. 保存
4. 调 `POST /api/skillbox/tools/reload` → 立刻生效

### 方式 2(改 seed 默认):修改内置 9 个工具

1. 改 `internal/toolseed/builtin.go`,增删条目
2. 重新 `go build`(只改常量,无新文件)
3. 全新 DB 用户:启动自动 seed 新内容
4. 已初始化用户:**不会自动更新** — 需要清空 e_tool + e_tool_path 表,或用前端 UI 改

---

## 一些坑

1. **改完业务数据没调 reload** — 前端必须 reload;后端 reload 是同步阻塞,
   无性能问题(约 10ms)。

2. **`~/` 展开时机** — 路径在 DB 里以 `~/xxx` 形式存,运行时由
   `BaseAdapter` 在生成 adapter 时按当前用户 home 展开。
   不同用户(系统)可能共享同一个 DB 快照,但 home 不同 — 这种设计兼容。

3. **GORM 软删除 vs 硬删除** — 当前 `Delete` 走 `SoftDelete: false`,
   真删。新建 + 删同 tool_id 不会冲突(没残留行)。

4. **并发安全** — `Registry` 用 `sync.RWMutex` 保护;`Reload()` 整体替换
   时用写锁,保证"读"侧(skillimporter.Scan / scope-status)不会看到中间状态。

5. **用户工具的 tool_id 重命名** — 本表设计不支持(tool_id 不可改);
   如果想重命名,先删后建。

6. **seed 阶段失败 → 服务起不来** — AutoMigrate 后立即 seed,失败 panic;
   启动期硬约束,符合"DB 不一致就别起来"原则。

---

## 调试技巧

```bash
# 1. 看当前 DB 里的工具
sqlite3 ~/.skill-box/data.db "SELECT tool_id, display_name, is_system, enabled FROM tools;"

# 2. 强制重新 seed(全新状态)
sqlite3 ~/.skill-box/data.db "DELETE FROM tool_paths; DELETE FROM tools;"
# 重启服务,EnsureSeeded 会重新写 9 条

# 3. 看某个工具的 path
sqlite3 ~/.skill-box/data.db "SELECT t.tool_id, p.scope, p.category, p.path FROM tools t JOIN tool_paths p ON p.tool_id = t.id WHERE t.tool_id = 'codex';"
```
