* 前端界面设计时，用户偏好简洁一点的界面，并且很反感AI感很强的界面 -20260623更新
* Claude Code 用户日常用的 skill 在 `~/.claude/skills/` 下(以 symlink 形式存在,目标在 `~/.agents/skills/` 等),不是 `~/.claude/plugins/marketplaces/`。adapter 必须同时扫这两个根,否则会漏掉用户的日常 skill。-20260623
* 桌面端请求/响应日志在 `~/.skill-box/logs/2026-06/06-DD-request.txt`(每请求一段,含响应体)。bug 复现不出来时第一件事是翻这个文件,不要瞎猜。-20260623