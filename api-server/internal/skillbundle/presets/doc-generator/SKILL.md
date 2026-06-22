---
name: doc-generator
version: 0.1.0
description: 根据代码 / diff / 接口签名自动生成结构化文档(README / API 参考 / CHANGELOG),支持中英文。
triggers:
  - doc
  - docs
  - document
  - readme
  - 文档
  - 生成文档
author: Skill Box
license: MIT
target_tools:
  - codex
  - claude
  - opencode
  - cursor
  - trae
---

# Doc Generator

从代码 / diff / 接口定义自动生成结构化文档。

## 何时触发

- 新写完一个模块,需要 README:`doc <module path>` 或 "给这个 module 写 README"
- API endpoint 写完,需要参考文档:粘 endpoint 定义
- release 前生成 CHANGELOG:跑 `git log v0.x..HEAD` + 喂结果
- 函数多但没注释:粘源码 + "补全注释"

## 行为

按"目标产物"分流:

### README

固定结构:
1. 一句话定位(<module> 是什么,解决什么问题)
2. 安装 / 引入(命令 / import 语句)
3. 快速开始(最小可跑示例)
4. API 概要(函数签名表,深度链接到详细说明)
5. 配置项(环境变量 / 参数表)
6. 常见问题(2-5 个 FAQ)
7. License / 贡献

### API 参考

每个 endpoint:
- Method + Path
- 入参 schema(类型 / 必填 / 默认值)
- 出参 schema
- 错误码表
- 1 个 curl 示例

### CHANGELOG

按 Keep a Changelog 分类:
- Added / Changed / Deprecated / Removed / Fixed / Security
- 每条 1 行,带 commit hash 或 PR 编号

## 输出格式

```text
# <Project Name>

<一句话定位>

## 安装

\`\`\`bash
<install command>
\`\`\`

## 快速开始

\`\`\`<lang>
<最小示例>
\`\`\`

## API

### <Endpoint Name>

\`\`\`http
<METHOD> <PATH>
\`\`\`

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |

## License
MIT
```

## 限制

- 不会执行代码 / 跑测试;只静态读源码
- 大项目只生成模块级 README,不在单文件内灌 README
- 不修改源代码,只产出新文件
