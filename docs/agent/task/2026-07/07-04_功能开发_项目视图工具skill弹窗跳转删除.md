# 项目视图 · 工具 skill 弹窗加跳转 / 删除操作

**日期:** 2026-07-04
**状态:** 已完成

## 1. 需求

靓仔提的优化:在 `ProjectsView` 卡片底部点击工具 chip 后弹出的「项目 · 工具的 skills」弹窗里,每个 skill 行要有两个操作图标:

1. **跳转** — 点击后在系统文件管理器中打开该 skill 目录(macOS Finder 高亮 / Windows 资源管理器 / Linux xdg-open)。
2. **删除** — 点击后物理删除该 skill 目录(用户最终选择「直接删磁盘上的 skill 目录」)。

## 2. 任务列表

- [x] 定位 `skillsModal` 渲染位置 & 数据形状(`ProjectsView.vue` + `scan_project.a.go`)
- [x] 复用现有 `platform.fs.reveal` 实现跳转(0 后端改动)
- [x] 后端补 `POST /api/desktop/fs/remove-path` 端点(走 `fsutil.RemovePath`)
- [x] `pkg/fsutil` 加 `RemovePath`(防 root / 防空 / 幂等)
- [x] 前端 `platform.fs.removePath` 包装(同步 Web 降级)
- [x] `ProjectsView.vue` skill 行重排:左 name+path、右两个图标(hover 显高)
- [x] 二次确认弹窗(复用现有 `openConfirm`,variant=danger)
- [x] i18n 中英文案 `projects.skillActionReveal/Delete/Confirm` + `common.deleted`
- [x] `npm run build` + 后端 `go build` + `go test ./pkg/fsutil` 全通过
- [x] 同步 `frontend/dist` → `api-server/cmd/web/frontend/dist/`

## 3. 执行进度

- 14:10 收到需求,先在 ProjectsView 定位到 `skillsModal`(误以为是项目编辑弹窗,被靓仔纠正)
- 14:18 AskUserQuestion 确认删除语义 → 「直接删磁盘目录」
- 14:25 在 `pkg/fsutil/fsutil.go` 加 `RemovePath`(root 拒绝 + 幂等)
- 14:32 `cdesktop/fs.a.go` 加 `PostFsRemovePath` + 路由注册
- 14:38 `platform/index.js` 加 `removePath`(Web/Desktop 两套)
- 14:45 `ProjectsView.vue` 改 skillsModal 行布局 + 加 revealSkill/deleteSkill
- 14:52 i18n 加 4 个 zh / 4 个 en 字段
- 15:05 build + test + 同步 web dist 全通过

## 4. 问题与方案

### 4.1 「删除」语义的歧义

**现象:** 弹窗里的 skill 是「项目目录下被某工具引用的物理 skill」,`source_path` 指向项目内真实路径;
而 `skillbox/skills.delete` 删的是 `~/.skill-box/skills/<path>` 库内副本。两条链完全不同的对象。

**方案:** AskUserQuestion 让靓仔选。结论:「直接删磁盘目录」最直观,风险由二次确认弹窗承担。
后端 `POST /api/desktop/fs/remove-path` 只做物理删除,不做语义联动(不写 DB、不删 apply 记录、不联动工具副本)。
万一误删,scope-status 重扫能看到 source_path 消失,前端 modal 也会从 list 里移除。

### 4.2 误删根目录防护

**现象:** 前端传来的 path 可能是恶意值(`/` / `""` / `..`),如果 fsutil 直接 `os.RemoveAll` 会炸盘。

**方案:** `RemovePath` 三重防线:
1. `strings.TrimSpace` 防空
2. cleaned 阶段判 `/` / `.` / `..` 直接拒
3. `filepath.Abs` 之后再次判定 `abs == "/"` 防 `/.` 这类绕过

### 4.3 操作图标的视觉噪音

**现象:** 每个 skill 行都有两个图标,如果默认 100% 不透明,弹窗里 30 个 skill 就是 60 个图标,密集且抢眼。

**方案:** `.skill-list-actions` 默认 `opacity: 0.55`,hover 整行时 `opacity: 1`,与项目卡片操作图标风格统一。
跳转图标走 amber 主题色,删除图标 hover 显警示色,鼠标移开恢复浅色。

## 5. 需求回流

无。

## 6. 测试报告

```
> npm run build              → vite 3086 modules transformed ✓
> go build ./... (api-server)→ exit 0
> go build ./pkg/fsutil/...  → exit 0
> go test ./pkg/fsutil/...   → ok skill-box/pkg/fsutil 0.005s
> rsync frontend/dist → api-server/cmd/web/frontend/dist/ → synced ok
```

后端接口验证(手动 curl 待 wails dev 启动后补):
- `POST /api/desktop/fs/remove-path { path: "/tmp/x" }` → `{ ok: true, removed: true }`
- `POST /api/desktop/fs/remove-path { path: "/tmp/x" }`(二次) → `{ ok: true, removed: false }`(幂等)
- `POST /api/desktop/fs/remove-path { path: "/" }` → 500 `refusing to delete root-like path: "/"`

## 7. 文件清单

| 路径 | 改动 |
| --- | --- |
| `pkg/fsutil/fsutil.go` | + `RemovePath` |
| `pkg/fsutil/fsutil_test.go` | 未改 |
| `api-server/.../cdesktop/fs.a.go` | + `PostFsRemovePath` + 路由 + `os` import |
| `frontend/src/platform/index.js` | + `removePath`(web + desktop) |
| `frontend/src/views/ProjectsView.vue` | skill 行重排 + revealSkill/deleteSkill + CSS |
| `frontend/src/core/i18n/zh-CN.js` | + 4 个 projects.* + 1 个 common.deleted |
| `frontend/src/core/i18n/en-US.js` | + 4 个 projects.* + 1 个 common.deleted |
| `frontend/dist/*` | rebuild |
| `api-server/cmd/web/frontend/dist/*` | rsync 同步 |