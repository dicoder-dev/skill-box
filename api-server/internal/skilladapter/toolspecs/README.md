# toolspecs — AI 编程工具元数据配置(数据驱动)

> 给"接一个新 AI 编程工具"这件事准备的一份速查手册。
> **新加一个工具 = 在 `specs/` 加一个 yaml 文件,不需要改 Go 代码。**

---

## 目录

| 路径 | 作用 |
| --- | --- |
| `specs/*.yaml` | 工具清单(每工具一个文件) |
| `schema.go` | `ToolSpec` / `ToolPaths` / `CategoryPaths` 结构体定义 + 校验 |
| `loader.go` | `//go:embed` + `yaml.Unmarshal` 加载器 |
| `specadapter.go` | `NewSpecAdapter(spec)` 把 ToolSpec 转成 `skilladapter.BaseAdapter` |
| `registry.go` | `init()` 把全部 spec 注册到 `skilladapter.DefaultRegistry()` |

---

## ToolSpec schema 草案

```yaml
tool_id: <string, 必填,全局唯一>
display_name: <string, 必填,UI 展示名>
mdi_icon: <string, 必填,"mdi:xxx" 格式,前端 iconify 用的 mdi 图标>
maturity: <stable|experimental|deprecated, 选填,默认 stable>
note: <string, 选填,自由文本,前端不展示>

paths:
  global:                       # 用户级,挂在 $HOME 下
    user:                       # 用户自己装,可读可写
      - "~/.xxx/skills"
    system:                     # 工具自带 / vendor,只读
      - "~/.xxx/skills/.system"
  project:                      # 项目级,挂在 <project>/.xxx/skills
    user:
      - ".xxx/skills"
    system:                     # 可省略
      - ".xxx/.system"
```

### 字段约束

| 字段 | 必填 | 约束 |
| --- | --- | --- |
| `tool_id` | ✅ | 小写字母 + 数字 + `-` / `_`,全局唯一,变更会破坏 DB 关联 |
| `display_name` | ✅ | 非空字符串,前端 i18n 可覆盖 |
| `mdi_icon` | ✅ | 必须 `mdi:` 开头;查 https://pictogrammers.com/library/mdi/ 选 |
| `maturity` | ❌ | `stable` / `experimental` / `deprecated`,默认 `stable` |
| `paths.<scope>` | ✅ | 每个 scope(global / project)至少一个 user 或 system 路径非空 |
| `note` | ❌ | 自由文本,前端不展示,仅供阅读 / 日志 |

---

## 路径展开规则

YAML 里写 `~/xxx` 路径时,Go 启动期会把 `~/` 展开为 `$HOME` 绝对路径。
项目级路径(以 `.` 开头,例如 `.claude/skills`)不展开,原样透传。

> ⚠️ 不要写环境变量 `$HOME` / `%USERPROFILE%`,目前只支持 `~/` 缩写。
> 需要绝对路径直接写出来,例如 `/opt/opencode/skills`。

---

## 扩展一个新工具 — Step by Step

假设要加一个新工具 `mytool`,用户级目录 `~/.mytool/skills`,项目级 `.mytool/skills`。

### 1. 在 specs/ 加一个 yaml

```bash
# 在 api-server/internal/skilladapter/toolspecs/specs/ 下
cat > mytool.yaml <<'EOF'
tool_id: mytool
display_name: "My Tool"
mdi_icon: "mdi:tools"
maturity: stable

note: |
  简单说一下这个工具是什么、为什么这样配路径。

paths:
  global:
    user:
      - "~/.mytool/skills"
  project:
    user:
      - ".mytool/skills"
EOF
```

### 2. 验证(单元测试)

```bash
cd api-server
go test ./internal/skilladapter/...
```

`TestAllAdaptersRegistered` 会断言:
- adapter 列表 ≥ 9 个(5 老 + N 新)
- 每个 adapter `Icon()` 返回 `"mdi:xxx"`(非空、mdi: 开头)

如果 `maturity: experimental`,前端会标注该工具为"实验性"。

### 3. 验证(运行时)

启动后调用 `GET /api/skillbox/skills/scope-status?name=<any-skill>`,
响应 `tools[]` 数组里能看到新工具的 `{tool_id, display_name, icon}`。
`icon` 必须是 `mdi:xxx`,不是空串。

### 4. 完事

不需要改 Go 代码,不需要 build skillbox。`wails3 dev` 会重新加载
`internal/skilladapter/toolspecs/` 包,YAML 改动后下次 API 请求就生效。

---

## 一些坑

1. **`//go:embed` 不支持符号链接** — specs/ 目录里**别放 symlink**。本目录
   走 embed 加载,符号链接会触发 `pattern cannot embed irregular file` 错误。
   真实案例见 docs/agent/memory/项目里记录的踩坑。

2. **tool_id 重复会 panic** — loader 二次校验 tool_id 唯一性,如果两个 yaml
   写了同一个 tool_id,启动时直接 `log.Fatalf`,不静默放过。

3. **Maturity 拼写错会被拒绝** — 必须是 `stable` / `experimental` / `deprecated`
   之一(大小写敏感)。错拼写会 `LoadAll` 失败,服务起不来。

4. **system 路径不要写到 BaseAdapter.Tools** — system 走
   `BaseAdapter.SystemPaths`,这样 importer 才能区分 user / system 档位,
   前端 phase2 才把 system 列为"只读参考,不可勾选"。

5. **项目级路径** — 一刀切写 `<project>/.claude/skills/` 这种是错的,
   一定要看工具官方文档。Antigravity 是 `.gemini/antigravity/skills/`,
   Claude 是 `.claude/skills/`,OpenCode 是 `.opencode/skills/`。

6. **`mdi_icon` 找不到合适的图标** — 用 `mdi:puzzle-outline` 占位,
   后续在 pictogrammers 找到合适的再改。

---

## 已注册工具一览(2026-06-30)

| tool_id | display_name | mdi_icon | maturity | 个人级路径 |
| --- | --- | --- | --- | --- |
| `antigravity` | Antigravity | `mdi:rocket-launch-outline` | stable | `~/.gemini/antigravity/skills` |
| `claude` | Claude Code | `mdi:robot-outline` | stable | `~/.agents/skills` |
| `cline` | Cline | `mdi:file-document-outline` | stable | `~/.agents/skills` + `~/.cline/skills` |
| `codebuddy` | CodeBuddy | `mdi:buddy` | **experimental** | `~/.codebuddy/skills` |
| `codex` | Codex | `mdi:console` | stable | `~/.agents/skills` |
| `cursor` | Cursor | `mdi:cursor-default-click-outline` | stable | `~/.cursor/skills` |
| `jetbrains` | JetBrains AI | `mdi:language-java` | **experimental** | `~/.jetbrains/skills` |
| `opencode` | OpenCode | `mdi:code-tags` | stable | `~/.config/opencode/skills` |
| `trae` | Trae | `mdi:leaf` | stable | `~/.agents/skills` |

> **Why:** 5 个老工具是 2026-06 之前各占一个 Go 子包,2026-06-30 改造后
> 全部迁到本目录的 yaml;4 个新工具直接以 yaml 形式落地,不需要写
> 任何 Go 代码。
