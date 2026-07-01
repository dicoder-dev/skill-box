# Skills.sh API 接口文档

> 本文档基于 https://www.skills.sh 网站实际抓包分析 + 官方 API 文档整理
> 抓包日期：2026-07-01
> Skills.sh 是 Vercel Labs 维护的 AI Agent Skills 目录网站
>
> 2026-07-01:已对接 skill-box 适配器(`api-server/internal/skillmarket/skillssh/skillssh.go`)。
> 由于公共 API(`/api/audits/{page}`)字段缺 description/version/tags,实际仍走 HTML 解析(首页 + 搜索页),
> 失败时降级到 knownCatalogFallback。`keyword` 透传到 `/search?q=`(经验路径,失败时子串过滤 fallback)。

---

## 项目概览

**项目性质**：Skills.sh 是基于 Next.js (App Router + RSC) 部署的纯静态/边缘渲染网站（Vercel 托管）。
底层是开源项目 https://github.com/vercel-labs/skills（CLI 工具 + 本地文件系统）。

**数据流**：通过 GitHub Actions 定时抓取各 GitHub 仓库的 SKILL.md，构建时生成静态 RSC payload，前端直接读取。
没有传统的运行时业务 API（除下方列出的少数 endpoint）。

**下载方式**：skills.sh 网站本身**不提供 zip 下载接口**，安装通过 CLI 完成：
```bash
npx skills add <owner/repo> --skill <skill-slug>
```

---

## 通用说明

### Base URL
```
https://skills.sh
```

### 公共响应头（Open API）
```
Access-Control-Allow-Origin: *
Content-Type: application/json; charset=utf-8
Cache-Control: public, max-age=...（不同接口 TTL 不同）
Server: Vercel
Strict-Transport-Security: max-age=63072000
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
```

### 鉴权相关响应头（需鉴权接口）
```
X-RateLimit-Limit: integer
X-RateLimit-Remaining: integer
X-RateLimit-Reset: integer（秒，窗口过期时间）
```

### 通用错误结构
```json
{
  "error": "error_code",
  "message": "Human-readable description."
}
```

### 错误码

| 状态码 | 含义 |
|--------|------|
| 400 | 请求参数无效 |
| 401 | 缺少/无效/过期的 Vercel OIDC token |
| 404 | 资源未找到（如 skill 不存在或暂无审计） |
| 429 | 触发限流，响应带 `Retry-After` 头 |
| 503 | 暂时不可用，需退避重试 |

---

## 接口清单

### 公开 API（无需鉴权）

| # | 接口 | Method | Path | 说明 |
|---|------|--------|------|------|
| 1 | 排行榜分页 | GET | `/api/audits/{page}` | 按安装量倒序分页返回所有 skills 及其审计结果 |

### 需 Vercel OIDC 鉴权

| # | 接口 | Method | Path | 说明 |
|---|------|--------|------|------|
| 2 | 技能列表（排行榜） | GET | `/api/v1/skills` | 分页排行榜，支持 all-time / trending / hot 三种视图 |
| 3 | 技能搜索 | GET | `/api/v1/skills/search` | 按 name/source/description 搜索 |
| 4 | 官方精选 | GET | `/api/v1/skills/curated` | 一方厂商认证的技能集 |
| 5 | 技能详情 | GET | `/api/v1/skills/{source}/{skill}` | 单个技能完整信息 + 文件树 |
| 6 | 安全审计 | GET | `/api/v1/skills/audit/{source}/{skill}` | 多家审计结果（Agent Trust Hub / Socket / Snyk / Runlayer / ZeroLeaks） |

### 网站路由（Next.js 页面，非 API）

| # | 路径 | 说明 |
|---|------|------|
| 7 | `/` | 首页（All Time 排行榜） |
| 8 | `/trending` | Trending 24h 排行榜 |
| 9 | `/hot` | Hot 当前小时排行榜 |
| 10 | `/topic` | 主题列表 |
| 11 | `/topic/{topic}` | 单个主题下的技能 |
| 12 | `/agent` | Agent 列表 |
| 13 | `/agent/{agent}` | 单个 Agent 的技能 |
| 14 | `/{owner}` | 单个 owner 的技能（owner 页） |
| 15 | `/{owner}/{repo}` | 单个 repo 的技能 |
| 16 | `/{owner}/{repo}/{skill}` | 单个技能详情页 |
| 17 | `/{owner}/{repo}/{skill}/security/{provider}` | 单个审计报告（provider: snyk/socket/agent-trust-hub） |
| 18 | `/official` | 官方精选 |
| 19 | `/audits` | 审计汇总 |
| 20 | `/docs` / `/docs/api` / `/docs/cli` 等 | 文档页 |
| 21 | `/about` `/privacy` `/terms` `/contact` | 项目信息页 |

---

## 1. 排行榜分页（公开 API）

### 基本信息
- **Method**: `GET`
- **Path**: `/api/audits/{page}`
- **用途**: 按安装量倒序分页返回所有 skills 及其审计结果（**无需鉴权**）
- **认证**: 不需要

### Path 参数

| 参数 | 类型 | 说明 |
|------|------|------|
| page | integer | 页码，从 0 开始。`audits/0` = 排名 1-50，`audits/1` = 排名 51-100 |

> 每页固定 50 条。`audits/{page}` 中 page 即为页索引。

### Request 示例
```
GET /api/audits/0
GET /api/audits/1
```

### Response 示例
```json
{
  "skills": [
    {
      "rank": 1,
      "source": "vercel-labs/skills",
      "skillId": "find-skills",
      "name": "find-skills",
      "agentTrustHub": {
        "source": "vercel-labs/skills",
        "slug": "find-skills",
        "skillFolderHash": "3013fdeb8a11b10b1eb795ec3ae8bfca38f7c26d",
        "partner": "Agent Trust Hub",
        "result": {
          "content_analysis": {
            "urls_found": ["https://skills.sh/", "..."],
            "total_urls": 2,
            "detected_commands": [],
            "total_commands": 0,
            "security_credentials": {
              "env_exports": [],
              "file_writes": [],
              "hardcoded_keys": [],
              "api_key_patterns": [],
              "env_file_operations": [],
              "plaintext_credentials": []
            },
            "total_credentials": 0,
            "dependency_analysis": {
              "python_packages": [],
              "node_packages": [],
              "remote_code_executions": [],
              "total_python_packages": 0,
              "total_node_packages": 0,
              "total_remote_executions": 0
            },
            "skill_md_sha256": "54b44dc9539df865fbb060f62fb062e8232e765852a0cf14c38301fe0c1eb264",
            "url_to_files": {
              "https://skills.sh/": ["SKILL.md"],
              "https://skills.sh/vercel-labs/agent-skills/vercel-react-best-practices": ["SKILL.md"]
            }
          },
          "urlite_analysis": {
            "results": [
              {"url": "https://skills.sh/", "verdict": "unknown", "malicious_type": null, "status": "success"}
            ],
            "summary": {"total_urls_checked": 2, "malicious": 0, "clean": 0, "unknown": 2, "errors": 0}
          },
          "av_analysis": {
            "results": [{"filename": "skill/SKILL.md", "verdict": "clean", "threat_name": null, "status": "success"}],
            "summary": {"total_files_scanned": 1, "clean": 1, "infected": 0, "errors": 0}
          },
          "gemini_analysis": {
            "verdict": "SAFE",
            "summary": "..."
          }
        }
      },
      "socket": null,
      "snyk": null
    }
  ]
}
```

### 字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| skills | array | 当前页的 skills 列表（每页 50 条） |
| skills[].rank | integer | 当前 rank（1-based，在所有页中连续） |
| skills[].source | string | GitHub `{owner}/{repo}` 或 well-known domain |
| skills[].skillId | string | skill slug（文件夹名） |
| skills[].name | string | 技能显示名 |
| skills[].agentTrustHub | object\|null | Agent Trust Hub 审计结果 |
| skills[].socket | object\|null | Socket 审计结果 |
| skills[].snyk | object\|null | Snyk 审计结果 |

> 三个审计对象都为 `null` 表示该 skill 尚未被相应审计商扫描。
> 三个审计对象内含具体的审计详细数据，结构取决于审计商（详见各审计对象内字段）。

---

## 2. 技能列表（排行榜，需鉴权）

### 基本信息
- **Method**: `GET`
- **Path**: `/api/v1/skills`
- **用途**: 排行榜分页，支持 all-time / trending / hot 三种视图
- **认证**: Vercel OIDC Token（详见下方"鉴权"章节）

### Query 参数

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| view | string | 否 | `all-time` | `all-time` / `trending` / `hot` |
| page | integer | 否 | 0 | 页码（0-indexed） |
| per_page | integer | 否 | 100 | 每页条数，1-500 |

### Request 示例
```
GET /api/v1/skills?view=trending&per_page=10
Authorization: Bearer $VERCEL_OIDC_TOKEN
```

### Response 示例
```json
{
  "data": [
    {
      "id": "vercel-labs/skills/find-skills",
      "slug": "find-skills",
      "name": "find-skills",
      "source": "vercel-labs/skills",
      "installs": 24531,
      "sourceType": "github",
      "installUrl": "https://github.com/vercel-labs/skills",
      "url": "https://skills.sh/vercel-labs/skills/find-skills"
    }
  ],
  "pagination": {
    "page": 0,
    "perPage": 10,
    "total": 8420,
    "hasMore": true
  }
}
```

### Hot 视图扩展字段
`view=hot` 时每条记录额外包含：
| 字段 | 类型 | 说明 |
|------|------|------|
| installsYesterday | integer | 昨日同一小时的安装量 |
| change | integer | 当前小时 - 昨日同时刻 |

### V1Skill 基础对象字段

| 字段 | 类型 | 说明 |
|------|------|------|
| id | string | 稳定唯一标识，格式 `{source}/{slug}` |
| slug | string | URL 安全 slug |
| name | string | 显示名 |
| source | string | GitHub: `owner/repo`；well-known: `domain.com` |
| installs | integer | 去重后的总安装量 |
| sourceType | string | `github` / `well-known` |
| installUrl | string\|null | GitHub URL 或 well-known base URL，配合 `npx skills add <url>` 使用 |
| url | string | skills.sh 上的详情页 URL |
| isDuplicate | boolean | 是否被识别为 fork/复制（仅 true 时返回） |

### Pagination 字段

| 字段 | 类型 | 说明 |
|------|------|------|
| page | integer | 当前页码 |
| perPage | integer | 每页条数 |
| total | integer | 总条数 |
| hasMore | boolean | 是否还有下一页 |

---

## 3. 技能搜索（需鉴权）

### 基本信息
- **Method**: `GET`
- **Path**: `/api/v1/skills/search`
- **用途**: 按 name/source/description 搜索；单词模糊匹配，多词语义搜索
- **认证**: Vercel OIDC Token

### Query 参数

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| q | string | 是 | - | 搜索关键词，最少 2 个字符 |
| limit | integer | 否 | 50 | 最大返回数，1-200 |
| owner | string | 否 | - | 限定 GitHub owner（跨仓库搜索） |

### 搜索类型自动判断
- **单词查询**（无空格）：`fuzzy` 模糊匹配
- **多词查询**（含空格）：`semantic` 语义搜索

### Request 示例
```
GET /api/v1/skills/search?q=react%20native&owner=expo&limit=5
Authorization: Bearer $VERCEL_OIDC_TOKEN
```

### Response 示例
```json
{
  "data": [
    {
      "id": "expo/skills/react-native",
      "slug": "react-native",
      "name": "React Native",
      "source": "expo/skills",
      "installs": 3842,
      "sourceType": "github",
      "installUrl": "https://github.com/expo/skills",
      "url": "https://skills.sh/expo/skills/react-native"
    }
  ],
  "query": "react native",
  "searchType": "semantic",
  "count": 5,
  "durationMs": 142
}
```

### 字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| data | array | 搜索结果（V1Skill 对象数组） |
| query | string | 实际执行的查询字符串 |
| searchType | string | `fuzzy` / `semantic` |
| count | integer | 实际返回条数 |
| durationMs | integer | 查询耗时（毫秒） |

---

## 4. 官方精选（需鉴权）

### 基本信息
- **Method**: `GET`
- **Path**: `/api/v1/skills/curated`
- **用途**: 一方厂商认证的技能集（与 `/official` 页面同数据源）
- **认证**: Vercel OIDC Token

### Query 参数
无

### Request 示例
```
GET /api/v1/skills/curated
Authorization: Bearer $VERCEL_OIDC_TOKEN
```

### Response 示例
```json
{
  "data": [
    {
      "owner": "vercel-labs",
      "totalInstalls": 89240,
      "featuredRepo": "skills",
      "featuredSkill": "find-skills",
      "skills": [
        {
          "id": "vercel-labs/skills/find-skills",
          "slug": "find-skills",
          "name": "find-skills",
          "source": "vercel-labs/skills",
          "installs": 24531,
          "sourceType": "github",
          "installUrl": "https://github.com/vercel-labs/skills",
          "url": "https://skills.sh/vercel-labs/skills/find-skills"
        }
      ]
    },
    {
      "owner": "supabase",
      "totalInstalls": 12084,
      "featuredRepo": "supabase",
      "featuredSkill": "Supabase",
      "skills": [...]
    }
  ],
  "totalOwners": 87,
  "totalSkills": 342,
  "generatedAt": "2026-03-31T08:00:00.000Z"
}
```

### 字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| data | array | 按 owner 分组的精选技能集 |
| data[].owner | string | GitHub owner / 组织名 |
| data[].totalInstalls | integer | 该 owner 旗下所有精选技能总安装量 |
| data[].featuredRepo | string | 推荐的 repo 名 |
| data[].featuredSkill | string | 推荐的 skill slug（主推技能） |
| data[].skills | array | V1Skill 对象数组 |
| totalOwners | integer | 精选 owner 总数 |
| totalSkills | integer | 精选 skill 总数 |
| generatedAt | string | 数据生成时间（ISO 8601） |

---

## 5. 技能详情（需鉴权）

### 基本信息
- **Method**: `GET`
- **Path**: `/api/v1/skills/{source}/{skill}`
- **用途**: 单个技能完整信息 + 文件树（SKILL.md 等所有支持文件）
- **认证**: Vercel OIDC Token

### Path 参数

| 来源类型 | 示例 |
|---------|------|
| GitHub skill | `/api/v1/skills/vercel-labs/skills/find-skills` |
| Well-known skill | `/api/v1/skills/mintlify.com/mintlify` |

> 也可以直接用 `id` 拼接：`/api/v1/skills/{id}`

### Request 示例
```
GET /api/v1/skills/vercel-labs/skills/find-skills
Authorization: Bearer $VERCEL_OIDC_TOKEN
```

### Response 示例
```json
{
  "id": "vercel-labs/skills/find-skills",
  "source": "vercel-labs/skills",
  "slug": "find-skills",
  "installs": 24531,
  "hash": "a1b2c3d4e5f6...",
  "files": [
    {
      "path": "SKILL.md",
      "contents": "---\nname: Next.js Development\n..."
    },
    {
      "path": "examples/app-router.ts",
      "contents": "// Example code..."
    }
  ]
}
```

### 字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| id | string | 稳定唯一标识，格式 `{source}/{slug}` |
| source | string | 源仓库或 provider |
| slug | string | URL 安全 slug |
| installs | integer | 总安装量 |
| hash | string\|null | skill 文件内容的 SHA-256（用于缓存失效/变更检测） |
| files | array\|null | 所有文件清单 |
| files[].path | string | 相对文件名 |
| files[].contents | string | 文件完整内容（文本） |

> `hash` 和 `files` 在没有快照时为 `null`。

---

## 6. 安全审计（需鉴权）

### 基本信息
- **Method**: `GET`
- **Path**: `/api/v1/skills/audit/{source}/{skill}`
- **用途**: 获取单个 skill 的多家审计商结果
- **认证**: Vercel OIDC Token
- **支持的审计商**: Gen Agent Trust Hub / Socket / Snyk / Runlayer / ZeroLeaks

### Path 参数
格式同技能详情：`/api/v1/skills/audit/{id}` 或 `/api/v1/skills/audit/{source}/{skill}`

### Request 示例
```
GET /api/v1/skills/audit/vercel-labs/skills/find-skills
Authorization: Bearer $VERCEL_OIDC_TOKEN
```

### Response 示例（无审计时 404）

成功响应：
```json
{
  "id": "vercel-labs/skills/find-skills",
  "source": "vercel-labs/skills",
  "slug": "find-skills",
  "audits": [
    {
      "provider": "Gen Agent Trust Hub",
      "slug": "agent-trust-hub",
      "status": "pass",
      "summary": "No risks detected",
      "auditedAt": "2026-04-15T12:00:00.000Z",
      "riskLevel": "LOW"
    },
    {
      "provider": "Socket",
      "slug": "socket",
      "status": "pass",
      "summary": "No alerts",
      "auditedAt": "2026-04-15T12:05:00.000Z"
    },
    {
      "provider": "Snyk",
      "slug": "snyk",
      "status": "pass",
      "summary": "Risk: LOW · No issues",
      "auditedAt": "2026-04-15T12:03:00.000Z",
      "riskLevel": "LOW"
    }
  ]
}
```

> 如果没有任何审计商审计过此 skill，返回 `404 Not Found`。
> 审计在 skill 首次安装后自动触发，可能有几分钟延迟。

### Audit Entry 字段

| 字段 | 类型 | 说明 |
|------|------|------|
| provider | string | 审计商显示名（Gen Agent Trust Hub / Socket / Snyk / Runlayer / ZeroLeaks） |
| slug | string | URL 安全 slug，链接到 `/owner/repo/skill/security/{slug}` 详情页 |
| status | string | `pass` (安全) / `warn` (建议复查) / `fail` (可能危险) |
| summary | string | 人工可读的简述 |
| auditedAt | string | 审计时间（ISO 8601） |
| riskLevel | string | `NONE` / `LOW` / `MEDIUM` / `HIGH` / `CRITICAL` |
| categories | string[] | 检测到的类别（如 `["NO_CODE", "SAFE"]`），仅 Agent Trust Hub 返回 |

---

## 鉴权（Vercel OIDC）

### Base URL
所有 `/api/v1/` 接口需要 Vercel OIDC 鉴权。

### 启用 OIDC
1. Vercel 控制台 → Project → Settings → OIDC Federation → 启用
2. 运行时 token 位于 `process.env.VERCEL_OIDC_TOKEN` 或请求头 `x-vercel-oidc-token`
3. 安装 SDK：`npm install @vercel/oidc`

### 调用示例（推荐方式）
```javascript
import { getVercelOidcToken } from '@vercel/oidc';

export async function GET() {
  const token = await getVercelOidcToken();

  const res = await fetch('https://skills.sh/api/v1/skills', {
    headers: { Authorization: `Bearer ${token}` },
  });

  return Response.json(await res.json());
}
```

> 必须在请求处理函数内部调用 `getVercelOidcToken()`，不要 hoist 到模块作用域。
> token 每 12 小时轮换，作用域是当前请求上下文。

### 直接读环境变量（不推荐）
```javascript
const token = process.env.VERCEL_OIDC_TOKEN;
await fetch('https://skills.sh/api/v1/skills', {
  headers: { Authorization: `Bearer ${token}` },
});
```

> 本地开发需要先 `vercel link` + `vercel env pull`，token 写到 `.env.local`。

### 限流

| 等级 | 限制 | 范围 |
|------|------|------|
| 鉴权用户 | 600 次/分钟 | Per (team, project) |

响应头携带：
- `X-RateLimit-Limit`：窗口内最大请求数
- `X-RateLimit-Remaining`：剩余请求数
- `X-RateLimit-Reset`：窗口过期剩余秒数

被限流时返回 `429` + `Retry-After` 头。

### 审计日志
每个鉴权请求都会记录 `owner_id`（team）、`project_id`、`environment`（`production` / `preview` / `development`）。**原始 token 永不被存储**。

---

## 网站页面路由（非 API）

### 浏览路由

| 路径 | 说明 |
|------|------|
| `/` | 首页 - All Time 排行榜 |
| `/trending` | Trending（24h）排行榜 |
| `/hot` | Hot 排行榜（与上一小时对比） |
| `/topic` | 主题列表页 |
| `/topic/{topic}` | 单个主题下技能（如 `/topic/react`） |
| `/agent` | Agent 列表页 |
| `/agent/{agent}` | 单个 Agent 适用技能（如 `/agent/claude-code`） |
| `/{owner}` | 单个 owner 的所有技能页 |
| `/{owner}/{repo}` | 单个 repo 的所有技能页 |
| `/{owner}/{repo}/{skill}` | 单个技能详情页 |
| `/{owner}/{repo}/{skill}/security/{provider}` | 单个审计详情（provider: snyk/socket/agent-trust-hub） |
| `/official` | 官方精选 |
| `/audits` | 所有审计汇总 |

### 文档路由

| 路径 | 说明 |
|------|------|
| `/docs` | 文档首页 |
| `/docs/api` | API 参考（本文档对应页面） |
| `/docs/cli` | CLI 使用说明 |
| `/docs/customize` | 自定义页面 |
| `/docs/faq` | 常见问题 |

### 项目路由

| 路径 | 说明 |
|------|------|
| `/about` | 关于 |
| `/privacy` | 隐私政策 |
| `/terms` | 服务条款 |
| `/contact` | 联系我们 |

### 已确认的 topic 列表

| slug | 中文名 |
|------|--------|
| `react` | React |
| `nextjs` | Next.js |
| `design` | Design & UI |
| `mobile` | Mobile |
| `agent-workflows` | Agent workflows |
| `databases` | Databases |
| `testing` | Testing |
| `marketing` | Marketing |

### 已确认的 agent 列表

| slug | 名称 |
|------|------|
| `claude-code` | Claude Code |
| `cursor` | Cursor |
| `codex` | Codex |
| `github-copilot` | GitHub Copilot |
| `windsurf` | Windsurf |
| `gemini` | Gemini |
| `cline` | Cline |
| `amp` | AMP |
| `antigravity` | Antigravity |
| `clawdbot` | ClawdBot |
| `droid` | Droid |
| `goose` | Goose |
| `kilo` | Kilo |
| `kiro-cli` | Kiro CLI |
| `nous-research` | Nous Research |
| `opencode` | OpenCode |
| `roo` | Roo |
| `trae` | Trae |
| `vscode` | VS Code |
| `zed` | Zed |

### 已确认的 source 类型

| sourceType | 示例 | 说明 |
|------------|------|------|
| `github` | `vercel-labs/skills` | GitHub 仓库 |
| `well-known` | `open.feishu.cn` | 通过 `/.well-known/` 协议发现的服务 |

---

## 安装方式（无下载接口）

Skills.sh 网站本身**不提供 zip 包下载接口**。安装方式：

### 通过 CLI 安装
```bash
npx skills add https://github.com/<owner>/<repo> --skill <skill-slug>
# 或
npx skills add <owner>/<repo> --skill <skill-slug>
```

### 安装来源
- **GitHub skill**：`installUrl` = `https://github.com/<owner>/<repo>`
- **Well-known skill**：`installUrl` = 对应 base URL

### 查找命令
```bash
npx skills find <query>            # 交互式搜索
npx skills list                    # 已安装技能
npx skills remove <name>           # 卸载
npx skills update                  # 更新
```

---

## 缓存策略

| 接口 | Cache-Control TTL |
|------|--------------------|
| `/api/audits/{page}` | 30-60 秒 |
| `/api/v1/skills`（排行榜） | 30-60 秒 |
| `/api/v1/skills/search` | 30-60 秒 |
| `/api/v1/skills/curated` | 5 分钟 |
| `/api/v1/skills/{source}/{skill}` | 5 分钟 |

> 轮询时建议尊重 `Cache-Control`，避免触发限流。

---

## 备注

1. **`/api/v1/` 全部需要 Vercel OIDC 鉴权**，仅 `/api/audits/{page}` 完全公开
2. 网站本身是 **Next.js RSC 静态网站**，所有页面骨架通过 `?_rsc=` 后缀拉取 RSC payload
3. 数据来源于 **GitHub Actions 定时抓取 + 构建时生成**，无传统意义上的运行时业务 API
4. **没有 zip 下载接口**，必须通过 `npx skills add` 命令行安装
5. skill 文件内容通过 `/api/v1/skills/{source}/{skill}` 一次返回完整 `files` 树，无需单独下载接口
6. 抓包时间：2026-07-01
7. 项目开源仓库：https://github.com/vercel-labs/skills
8. CLI 包名：`skills`（`npx skills`）
9. skills.sh 与 skillhub.cn / skills.sh 没有关联，是 Vercel Labs 独立运营项目