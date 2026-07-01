* 编写 md 文档时，用户更喜欢简洁明了，最好一句话能突出主题的 -20260623更新
* 期望 task 文件每轮对话都要维护(不依赖 git commit):每轮对话结束在对应 task 文件追加 `## N.M 对话轮次 (HH:MM)`,包含 用户原话 / 本轮做了 / 本轮决定 / 本轮待办 / 状态更新 五段。规则详见 `docs/agent/task/README.md`。-20260624
* PATH 中存在 DevEco-Studio 自带 Node v18.20.1 干扰 MCP 启动,处理 npx 类 MCP 时必须用绝对路径 node 绕开,详见 [[mcp-node-version]] -20260701