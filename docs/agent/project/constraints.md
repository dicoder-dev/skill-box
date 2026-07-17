# 项目级强制约束(MCP / 图标库 / 测试)

> 这里的每一条都是 **硬约束**:Claude 改了就是违规,用户会立即指出。
> 与 `conventions.md`(命名/风格)并列,但 **优先级更高** —
> 遇到冲突时以本文件为准,例如「不准用 emoji」比「保持代码注释简洁」优先级高。
>
> 维护:每条规则标明首次建立日期和最近修订日期,方便回溯。

---

## 1. MCP 工具使用规范

> 优先级: **模型自带能力 → MCP 兜底**。规则不冲突时优先用模型原生,模型不具备的能力再降级到 MCP。

### 1.1 图片理解 / 分析
- **优先使用模型自带的图片识别能力**(原生多模态),不调 MCP。
- **模型不支持多模态时**,才降级到 MiniMax `understand_image` MCP,无需跟用户确认。
- 业务侧(读图、改 UI 调样式、看错误截图)优先靠模型视觉能力,模型搞不定再走 MCP。

**反例(禁止)**:
- 模型没有视觉能力时,跳过 MCP 直接根据文本描述"猜"问题在哪。

### 1.2 联网搜索
- **优先使用模型自带的联网搜索能力**(如 WebSearch),不调 MCP。
- **模型没有联网搜索能力时**,才降级到 MiniMax `web_search` MCP,无需跟用户确认。
- WebFetch 仅用于:已知具体 URL 需要看完整页面内容时(MCP 搜索只返摘要)。

**反例(禁止)**:
- 看到不熟的 API / 错误码,不去搜就开始"按印象写代码"。
- 模型自带搜索就够用,却硬塞 MCP 走兜底路径,徒增步骤。

---

## 2. 前端图标库统一:iconpark

> 源约定:仓库根 `CLAUDE.md`「禁止使用 emoji 作为项目图标」。
> 历史包袱:早期混用过 `@iconify/vue`、naive-ui 图标、emoji,后统一收口到 iconpark。

### 2.1 必须遵守
- 前端图标 **100% 走 `@icon-park/vue-next`**,通过 `<IconPark />` 封装组件使用。
- 业务侧禁止直接 `import { Xxx } from '@icon-park/vue-next'`,统一走封装组件,
  便于统一走 `mdi:xxx → iconpark 组件名` 的映射兜底。
- mdi 风格字符串(`mdi:xxx`)是历史包袱,允许继续使用,但内部必须能在
  `frontend/src/core/icons/iconparkMap.js` 里查到对应 iconpark 组件名。
- iconpark 没有完全对应的图标时,选语义最贴近的,**留 `(alias)` 注释**方便回查,
  **不允许 fallback 到 emoji / inline svg / iconify 远端 API**。
- 新增 mdi 用法前必须先检查映射表;缺失就补一行,**不允许让未映射的 key 上线**。

### 2.2 严禁
- ❌ 任何 emoji 作为业务图标(🚁/📁/💡 等)。CLAUDE.md 硬规则。
- ❌ `@iconify/vue` 的 `<Icon icon="mdi:xxx" />` 写法,业务侧已全部下掉。
- ❌ naive-ui 的 `<n-icon>` 组件。
- ❌ inline `<svg>` 手画图标(仅 logo 等品牌资产例外,放 `frontend/public/`)。
- ❌ wails3 webview 加载 iconify 在线 API(offline-only,见 `docs/agent/memory/wails3-webview-iconify-network.md`)。

### 2.3 维护
- 映射表: `frontend/src/core/icons/iconparkMap.js`,NotFound 走 `Help`(问号)兜底。
- 补映射时同步更新本文件 + 在 commit message 写明新增了哪些 key。

---

## 3. 测试策略

> 源约定:用户 2026-07-18 在多次会话中明确强调。

### 3.1 默认:Chrome MCP 自测,不启动桌面端
- 所有 web 端 / 跨端功能的验证,**统一用 chrome-devtools MCP** 在浏览器里跑。
- 流程:启动 `npm run dev` → chrome MCP `navigate_page` → `take_snapshot` + `list_console_messages`
  → 必要时截图存到 `docs/agent/tmp-screens/`。
- 完成后才反馈"通过",**不能让用户去测试**(用户原话:"要自测再反馈通过,不要让我去测试")。

### 3.2 桌面端特定场景:AI 不测试
- **桌面端专属功能**(macOS 红绿灯让位、LaunchServices、`launchctl asuser` 等)
  Claude 模型无法真正驱动测试,硬测只能 mock / 跑模拟,无意义。
- 这类场景:
  - 不主动尝试跑 wails3 build / 启动桌面 binary。
  - 改动后 **写明"此改动仅桌面端生效,需用户在桌面端验证"**,不假装"已自测通过"。
  - 不在 commit message / PR 描述里写 "tested on macOS desktop"。

### 3.3 何时该停下来问用户
- 桌面端 binary 启动失败、amfi / Gatekeeper 报错、LS DB stale —— 这类根因在
  macOS 26 Tahoe 上 AI 没办法复现 + 排查,直接建议用户在「隐私与安全 → 仍要打开」
  或跑 `scripts/dev/start-skillbox--dev.sh` 验证,不要自己反复重试。
- 见 `docs/agent/memory/macos-tahoe-amfi-423-gui-still-open.md`、
  `docs/agent/memory/macos-tahoe-amfi-短路所有启动路径.md`。

### 3.4 反例(禁止)
- ❌ 改完代码直接说"应该没问题,用户可以试试" — 用户原话禁止。
- ❌ 在没有 chrome MCP / 截图证据时说"已自测通过"。
- ❌ 改桌面端专属功能,然后强行启动 wails3 binary 测 — 浪费时间且不可靠。

---

## 4. 维护与索引

- 本文件首次建立: 2026-07-18。
- 修订记录: 见本文每节标题下的「源约定」行,以及 git blame。
- 与 `docs/agent/memory/` 的关系: 长期踩坑教训放 memory(具体案例),
  本文件只放通用规则(适用所有项目)。改规则前先在 memory 沉淀案例,
  再把通用部分抽到本文件。