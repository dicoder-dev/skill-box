# mcp 启动问题排查与修复

**问题现象：** `/mcp` 命令持续报 `Failed to reconnect to chrome-devtools: -32000`。

## 根本原因

`PATH` 中 DevEco-Studio 自带的 Node 优先级高于 Homebrew Node：

- `chrome-devtools-mcp@1.4.0` 要求 `Node ^20.19.0 || ^22.12.0 || >=23`
- `/Volumes/MyDrive/Applications/DevEco-Studio.app/Contents/tools/node/bin/node` 版本是 **v18.20.1**
- `/opt/homebrew/bin/node` 版本是 **v23.11.0**(满足要求)
- `npx` 内部通过 `#!/usr/bin/env node` 找 PATH 中**第一个** node,命中了 DevEco 的 v18,直接启动失败

## 修复方案

1. 写包装脚本 `~/local/bin/chrome-devtools-mcp.sh`(已创建并 chmod +x)
   - 强制使用绝对路径 `/opt/homebrew/bin/node`
   - 优先复用 `~/.npm/_npx/*/node_modules/chrome-devtools-mcp/build/src/bin/chrome-devtools-mcp.js` 缓存
   - 缓存被清理时,自动 `npm exec --package=chrome-devtools-mcp` 重新拉取并 `require.resolve` 定位入口
2. `.mcp.json` 的 `command` 字段指向该脚本,不再用 npx

## 验证方式

```bash
/Users/brody/.local/bin/chrome-devtools-mcp.sh --help
# 应输出完整的 chrome-devtools-mcp 选项列表
```

## 适用范围

本机 PATH 中混入了多套 Node 时,任何 `npx -y <pkg>` 形式的 MCP 都可能遇到同样问题。
修复后只对 chrome-devtools MCP 生效;若新增其他需要新 Node 的 MCP,建议复用同一脚本模式。