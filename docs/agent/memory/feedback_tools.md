---
name: 工具使用偏好
description: 图片理解 / 联网搜索统一使用 MiniMax MCP,无需确认
type: feedback
---

## 规则

- **图片理解 / 分析** → 用 `MiniMax - understand_image` MCP
- **联网搜索** → 用 `MiniMax - web_search` MCP
- 上述 MCP 调用 **无需跟用户确认**

**Why:** 用户在全局 CLAUDE.md 明确指定,且无需确认是为了减少交互摩擦。

**How to apply:** Claude 默认识别图片 / 联网需求时,主动选 MiniMax MCP,
不要 fallback 到内置 `WebFetch` / `WebSearch`(除非 MCP 不可用)。