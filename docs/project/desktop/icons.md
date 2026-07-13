# 桌面端图标(Dock + 托盘)

> 桌面端有两套图标:**Dock / 应用图标**(彩色,出现在 Dock / Applications 文件夹 / AltTab)
> 和 **托盘菜单栏图标**(macOS 单色 + alpha,Windows/Linux 彩色)。
> 它们**共用同一张源图**(`build/appicon.png`),但走两条不同的编译路径,
> 必须**手动同步**到 `desktop/appicon.png`。

---

## 1. 资源文件清单

| 文件                                | 作用                                                              | 谁会读它                                          |
| ----------------------------------- | ----------------------------------------------------------------- | ------------------------------------------------- |
| `build/appicon.png`                 | **图标唯一源**,1024×1024 PNG                                      | `wails3 generate icons` 命令                      |
| `build/darwin/icons.icns`           | macOS 多尺寸图标(由 `appicon.png` 生成)                           | 打包 `.app` 时由 `build/darwin/Taskfile.yml` 拷贝 |
| `build/windows/icon.ico`            | Windows 多尺寸图标(由 `appicon.png` 生成)                        | 打包 `.exe` 时由 `build/windows/Taskfile.yml` 拷贝 |
| `desktop/appicon.png`               | **托盘图标嵌入源**,1024×1024 PNG,跟 `build/appicon.png` 保持一致   | `desktop/tray.go` 的 `//go:embed appicon.png`     |

> ⚠️ `desktop/appicon.png` **不是符号链接**,是真实文件 — 因为 Go 的
> `//go:embed` **不支持符号链接**(embed 阶段 `pattern cannot embed irregular file`)。
> 必须手动 cp 同步。注释里写的是符号链接是**过时的**,实际是独立文件。

---

## 2. 调用链路

### 2.1 Dock / 应用图标(系统级)

```
build/appicon.png (1024×1024 源)
       │
       │  wails3 task common:generate:icons
       ▼
build/darwin/icons.icns  ─┬─→  bin/<APP>.app/Contents/Resources/icons.icns  (macOS)
                          └─→  .dmg / .pkg 安装包
build/windows/icon.ico   ────→  bin/<APP>.exe 内嵌资源  (Windows)
```

macOS 系统的图标缓存 (`com.apple.iconservices.store`) 会缓存 .icns 的渲染结果,
替换图标后需要:

```bash
# 强制系统重新读取
touch bin/<APP>.app
rm -f ~/Library/Caches/com.apple.iconservices.store
killall Dock Finder
```

### 2.2 托盘菜单栏图标(应用内)

```
desktop/appicon.png (1024×1024)
       │
       │  //go:embed appicon.png (编译时嵌入)
       ▼
desktop/tray.go:var trayAppIconPNG []byte
       │
       │  generateTrayIcons()
       │
       ├───► tmpl  (64×64, RGB 全压黑 + 源图 alpha)
       │     └──► macOS: t.SetTemplateIcon(tmpl)
       │            ↑ macOS 系统根据菜单栏亮/暗自动反色成 黑/白
       │
       └───► color (32×32, 原图 RGBA 缩放)
             └──► Windows / Linux: t.SetIcon(color)
```

---

## 3. 两套图标的差异

| 维度         | Dock / 应用图标                          | 托盘图标(macOS 模板图)              |
| ------------ | ---------------------------------------- | ----------------------------------- |
| 颜色         | 彩色(原图)                              | **必须是单色**(RGB 全压黑)          |
| 背景         | 实色 / 透明 都可以                      | **必须透明 alpha**,不能有彩色背景  |
| 来源         | `build/appicon.png` → `.icns`            | `desktop/appicon.png` → 嵌入二进制  |
| 替换后生效   | `wails3 task common:generate:icons` + 重新 build + 刷缓存 | 重新 `wails3 dev` / `wails3 build` |

> **重要**:`tray.go:generateTrayIcons` 已经把彩色 RGB 压成黑色,
> 不需要源图本身是单色 — 任意彩色 PNG 都能用,**只要背景 alpha=0**。
> 系统会在亮色菜单栏画黑、暗色菜单栏画白,实现自动反色。

---

## 4. 替换图标的完整流程

### 4.1 准备源图

- 尺寸:**1024×1024** PNG
- 背景:透明 (alpha=0)
- 主体:彩色,居中,留 10%~15% padding

### 4.2 复制到两个位置

```bash
SRC=/path/to/new-icon.png   # 1024x1024 透明背景 PNG

cp "$SRC" build/appicon.png
cp "$SRC" desktop/appicon.png
```

> 两个文件必须保持一致,**任一不一致都会导致图标分裂**(Dock 跟托盘不是同一个)。

### 4.3 重新生成各平台图标

```bash
wails3 task common:generate:icons
```

这一步会从 `build/appicon.png` 重新生成 `build/darwin/icons.icns` 和
`build/windows/icon.ico`,覆盖之前的版本。

### 4.4 强制刷新 macOS Dock 图标缓存

如果 `.app` 之前已经 build 出来,需要把新的 `.icns` 同步到 app bundle 并刷缓存:

```bash
# 1. 同步 .icns 到 .app
cp build/darwin/icons.icns bin/<APP>.app/Contents/Resources/icons.icns

# 2. touch 触发 iconservices 重新读取
touch bin/<APP>.app

# 3. 清缓存 + 重启 Dock/Finder
rm -f ~/Library/Caches/com.apple.iconservices.store
killall Dock Finder
```

> 也可以直接 `wails3 build` 重编一遍,新 `.app` 会自动用最新的 `.icns`。

### 4.5 让托盘图标生效

托盘图标嵌入在 Go 二进制里,必须**重新编译桌面端**才能生效:

```bash
wails3 dev   # 开发期,热重启后生效
# 或
wails3 build # 生产构建
```

---

## 5. 常见坑

### 5.1 Dock 换了但托盘没换

**原因**:`build/appicon.png` 改了,但 `desktop/appicon.png` 没同步。
**修复**:`cp build/appicon.png desktop/appicon.png`,重跑 `wails3 dev`。

### 5.2 托盘图标一直是 wails 默认图标

**原因**:`desktop/tray.go:86` 用了 `SetTemplateIcon(trayOfficialTemplatePNG)`(官方默认),
或 `generateTrayIcons` 报错但被降级到 `SetLabel("Skill Box")`(见 `NewTrayManager`
的 iconErr 分支)。
**修复**:确认 `SetTemplateIcon(tmpl)` 走的是我们自己生成的模板图;
启动日志 `tray: 图标生成失败` 出现时说明 `generateTrayIcons` 报错,
通常是因为 `//go:embed appicon.png` 找不到文件(检查 `desktop/appicon.png` 存在)。

### 5.3 托盘图标是黑色色块,看不出形状

**原因**:源图是**不透明背景**(没有 alpha=0)。`generateTrayIcons` 把所有像素的
RGB 都压成黑色,结果背景区域也变成黑色,系统反色后整个菜单栏图标区域被涂满。
**修复**:用图像工具把背景的 alpha 设为 0(透明),只保留主体的不透明像素。

### 5.4 macOS Dock 一直显示旧图标

**原因**:`com.apple.iconservices.store` 缓存没刷新。
**修复**:见 §4.4。
**最坏情况**:`sudo rm /Library/Caches/com.apple.iconservices.store`(需要密码)。

---

## 6. 相关文件

| 文件                                    | 说明                                                  |
| --------------------------------------- | ----------------------------------------------------- |
| `desktop/tray.go`                       | 托盘图标核心逻辑:`//go:embed` + `generateTrayIcons`   |
| `desktop/wails_app.go`                  | Wails App 构造,会调 `NewTrayManager`                 |
| `desktop/assets/official_template.png`  | wails 官方 `DefaultMacTemplateIcon`(调试用,不用于发布)|
| `build/Taskfile.yml` (`generate:icons`) | `wails3 generate icons` 任务定义                      |
| `build/darwin/Taskfile.yml`             | macOS build 时把 `icons.icns` 拷进 `.app/Contents/Resources` |
| `build/windows/Taskfile.yml`            | Windows build 时嵌入 `icon.ico` 到 `.exe`             |

---

## 7. 版本历史

| 日期       | 改动                                                                              |
| ---------- | --------------------------------------------------------------------------------- |
| 2026-07-13 | 新建本目录,记录图标资源、调用链路、替换流程、常见坑(由 ChatGPT 工具箱图标准备替换触发的文档化) |