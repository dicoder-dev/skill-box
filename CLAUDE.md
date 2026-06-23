

# 项目简介

`skill-box` 是基于 Wails v3 + Gin + Vue 3 + Pinia 的桌面/Web 双形态应用,
本仓库为 Go 1.25 工作区,业务代码全部在 `api-server/` 与 `frontend/`。

wails3 文档地址：https://v3.wails.io/quick-start/why-wails/
---

# 项目说明
* docs/project/项目架构.md 这个文档是项目的架构说明，有需要时可以进行阅读和理解
* 请求日志存放在家目录/.skill-box/logs 里面遇到问题时不要瞎猜看看请求数据和响应数据
* 必要时可以调用相关 skill 或者 mcp 来联网搜索查询相关数据或者文档，不要瞎猜

# 必须遵守
* 前端界面开发时，禁止使用 emoji 作为项目图标，优先使用 ui-ux-pro-max skill作为界面开发指导
* 没完成一个功能点或者修复完成一个 bug，都要提交一次 git，自行生成 commit 信息
* 只有 **删除文件 / 强推 / 强 reset / 跳过 hooks / 强 --force 等不可逆操作**需要先跟用户确认。其他任何操作(写文件、跑测试、跑构建、调 MCP、git add/commit)**直接做,无需确认**。
* docs/agent/memory 这个目录保存与用户对话过程中用户偏好和项目开发注意事项，你需要实时更新和动态维护该目录下的文件，自行解析用户的话并分析，如果属于偏好则自行更新该目录下的文件。因此每次执行任务前都需要阅读一次该文件以知晓用户的偏好，再进行开发，同理对话过程中如果遇到用户偏好也要更新维护该文件
* docs/agent/task文件夹中记录每次对话，在对话过程中你要实时维护该文件，具体要求可以查看docs/agent/task/README.md文件说明
* docs/agent/project文件夹保存项目的开发规范，docs/agent/project/README.md是目录说明，开发前你也要先阅读该文件，并按照规范要求进行开发。