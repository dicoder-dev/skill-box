# SkillHub API 接口文档

> 本文档基于 https://skillhub.cn 网站实际抓包分析得出
> 抓包日期：2026-07-01
> 所有接口均通过浏览器 Network 面板验证

---

## 通用说明

### Base URL
```
https://api.skillhub.cn
```

### 通用请求头
```
Accept: */*
Accept-Encoding: gzip, deflate, br, zstd
Accept-Language: zh-CN,zh;q=0.9
Origin: https://skillhub.cn
Referer: https://skillhub.cn/
User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 ...
```

### 通用 CORS 响应头
```
Access-Control-Allow-Credentials: true
Access-Control-Allow-Headers: Authorization,Content-Type,X-Skillhub-Token,X-Requested-With,traceparent,tracestate,X-API-Key,X-Client-User-Id
Access-Control-Allow-Methods: GET,POST,PUT,PATCH,DELETE,OPTIONS
Access-Control-Allow-Origin: https://skillhub.cn
Access-Control-Max-Age: 600
```

### 通用响应体结构

成功响应：
```json
{
  "code": 0,
  "data": { ... },
  "message": "success"
}
```

错误响应：
```json
{
  "code": 400,
  "data": null,
  "message": "参数错误：xxx"
}
```

---

## 接口列表

### 列表与搜索

| 序号 | 接口 | Method | Path |
|------|------|--------|------|
| 1 | 技能列表（搜索） | GET | /api/skills |
| 2 | 技能分类列表 | GET | /api/v1/categories |
| 3 | 当前用户信息 | GET | /api/v1/auth/me |

### 首页

| 序号 | 接口 | Method | Path |
|------|------|--------|------|
| 4 | 首页轮播图 | GET | /api/v1/banners |
| 5 | 热门下载榜单 | GET | /api/v1/showcase/hot |
| 6 | 推荐榜单 | GET | /api/v1/showcase/recommended |
| 7 | 最新榜单 | GET | /api/v1/showcase/newest |
| 8 | 飙升榜单 | GET | /api/v1/showcase/trending |

### 技能详情与下载

| 序号 | 接口 | Method | Path |
|------|------|--------|------|
| 9 | 技能详情 | GET | /api/v1/skills/{slug} |
| 10 | 技能评测报告 | GET | /api/v1/skills/{slug}/evaluation |
| 11 | 版本列表 | GET | /api/v1/skills/{slug}/versions |
| 12 | 版本文件清单 | GET | /api/v1/skills/{slug}/files?version={ver} |
| 13 | 单个文件（302 跳转） | GET | /api/v1/skills/{slug}/file?path=...&version=... |
| 14 | 整包下载（302 跳转） | GET | /api/v1/download?slug={slug} |
| 15 | 数字签名 | GET | /api/v1/open/skills/{slug}/versions/{ver}/signature |
| 16 | 版本 diff 总览 | GET | /api/v1/skills/{slug}/diff?base=...&target=... |
| 17 | 版本 diff 文件内容 | GET | /api/v1/skills/{slug}/diff/file?base=...&target=...&path=... |
| 18 | 相关推荐 | GET | /api/v1/skills/{slug}/recommendations?pageSize=... |

### 批量 / 大赛

| 序号 | 接口 | Method | Path |
|------|------|--------|------|
| 19 | 批量查询技能 | POST | /api/v1/skills/batch |
| 20 | 大赛信息 | GET | /api/v1/contest/info |
| 21 | 技能参赛状态 | GET | /api/v1/contest/skills/{slug}/joined |
| 22 | 大赛 Top 榜 | GET | /api/v1/contest/top |

### 认证

| 序号 | 接口 | Method | Path |
|------|------|--------|------|
| 23 | 微信扫码登录 | GET | /api/v1/auth/wechat/qrcode |

---

## 1. 技能列表（搜索）

### 基本信息
- **Method**: `GET`
- **Path**: `/api/skills`
- **用途**: 获取技能列表（支持搜索、筛选、排序、分页）

### Query 参数

| 参数名 | 类型 | 必填 | 默认值 | 说明 |
|--------|------|------|--------|------|
| page | integer | 否 | 1 | 页码，从 1 开始 |
| pageSize | integer | 否 | 24 | 每页条数，最大 100 |
| keyword | string | 否 | - | 搜索关键词（模糊匹配 name/description） |
| category | string | 否 | - | 一级分类 key，见 `/api/v1/categories` |
| subCategory | string | 否 | - | 二级分类 key |
| source | string | 否 | - | 来源：`community`（社区）、`enterprise`（企业）、`clawhub`（ClawHub） |
| sortBy | string | 否 | score | 排序字段：`updated_at`/`downloads`/`stars`/`installs`/`score` |
| order | string | 否 | desc | 排序方向：`asc`/`desc` |
| labels | string | 否 | - | 标签过滤，如 `requires_api_key:true`（仅需 API Key 的） |

> 说明：`sortBy` 传入不支持的字段时，后端会返回 400 错误并提示支持的字段列表。

### Request 示例
```
GET /api/skills?page=1&pageSize=24&sortBy=score&order=desc
GET /api/skills?page=1&pageSize=24&sortBy=downloads&order=desc&keyword=股票&category=professional
GET /api/skills?page=1&pageSize=5&sortBy=score&order=desc&requiresApiKey=true
GET /api/skills?page=1&pageSize=24&sortBy=downloads&order=desc&keyword=PPT&category=office-efficiency&source=community&labels=requires_api_key%3Atrue
```

### Response 示例
```json
{
  "code": 0,
  "data": {
    "skills": [
      {
        "category": "knowledge-management",
        "claim_state": "unclaimed",
        "claimable": false,
        "claimed_user_handle": null,
        "created_at": 1774842724122,
        "description": "MANDATORY before calling web_search, web_fetch, browser, or opencli...",
        "description_zh": "MANDATORY before calling web_search, web_fetch, browser, or opencli. Contains required error-handling procedures...",
        "downloads": 168190,
        "homepage": "https://api.skillhub.cn/user_ec205dbb/web-tools-guide",
        "iconUrl": "https://cloudcache.tencent-cloud.com/qcloud/ui/static/other_external_resource/7422abc7-fa86-4505-a723-0575c56e7a2d.png",
        "installs": 3459,
        "labels": {
          "requires_api_key": "false"
        },
        "last_synced_at": null,
        "name": "web-tools-guide",
        "ownerName": "user_ec205dbb",
        "score": 100000,
        "slug": "web-tools-guide",
        "source": "community",
        "stars": 86,
        "subCategories": [
          {
            "key": "knowledge-retrieval",
            "name": "信息检索"
          }
        ],
        "tags": null,
        "updated_at": 1782878868630,
        "upstream_owner_login": null,
        "upstream_url": null,
        "verified": false,
        "version": "1.0.2"
      }
    ],
    "total": 61804
  },
  "message": "success"
}
```

### Response 字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| code | integer | 状态码，0 表示成功 |
| data.skills | array | 技能列表 |
| data.total | integer | 总记录数（用于分页） |
| message | string | 消息 |

技能对象字段：

| 字段 | 类型 | 说明 |
|------|------|------|
| category | string | 一级分类 key |
| claim_state | string | 认领状态 |
| claimable | boolean | 是否可被认领 |
| claimed_user_handle | string\|null | 认领者 handle |
| created_at | integer | 创建时间（毫秒时间戳） |
| description | string | 英文描述 |
| description_zh | string | 中文描述 |
| downloads | integer | 下载量 |
| homepage | string | 主页 URL |
| iconUrl | string | 图标 URL |
| installs | integer | 安装量 |
| labels | object | 标签，如 `{"requires_api_key": "true"}` |
| last_synced_at | integer\|null | 最后同步时间 |
| name | string | 技能名称 |
| ownerName | string | 所有者名称 |
| score | number | 评分 |
| slug | string | 唯一标识 |
| source | string | 来源 |
| stars | integer | 收藏数 |
| subCategories | array | 二级分类列表 |
| tags | object\|null | 标签信息 |
| updated_at | integer | 更新时间（毫秒时间戳） |
| upstream_owner_login | string\|null | 上游所有者登录名 |
| upstream_url | string\|null | 上游 URL |
| verified | boolean | 是否已验证 |
| version | string | 当前版本 |

---

## 2. 技能分类列表

### 基本信息
- **Method**: `GET`
- **Path**: `/api/v1/categories`
- **用途**: 获取所有一级和二级分类

### Query 参数
无

### Request 示例
```
GET /api/v1/categories
```

### Response 示例
```json
{
  "count": 12,
  "items": [
    {
      "key": "office-efficiency",
      "level": 1,
      "version": 2,
      "name": "办公效率",
      "nameEn": "Office Efficiency",
      "sortOrder": 10,
      "active": true
    },
    {
      "key": "content-creation",
      "level": 1,
      "version": 2,
      "name": "内容创作",
      "nameEn": "Content Creation",
      "sortOrder": 20,
      "active": true
    },
    {
      "key": "dev-programming",
      "level": 1,
      "version": 2,
      "name": "开发编程",
      "nameEn": "Development",
      "sortOrder": 30,
      "active": true
    },
    {
      "key": "data-analysis",
      "level": 1,
      "version": 2,
      "name": "数据分析",
      "nameEn": "Data Analysis",
      "sortOrder": 40,
      "active": true
    },
    {
      "key": "design-media",
      "level": 1,
      "version": 2,
      "name": "设计多媒体",
      "nameEn": "Design & Media",
      "sortOrder": 50,
      "active": true
    },
    {
      "key": "ai-agent",
      "level": 1,
      "version": 2,
      "name": "AI Agent",
      "nameEn": "AI Agent",
      "sortOrder": 60,
      "active": true
    },
    {
      "key": "knowledge-management",
      "level": 1,
      "version": 2,
      "name": "知识管理",
      "nameEn": "Knowledge Management",
      "sortOrder": 70,
      "active": true
    },
    {
      "key": "business-ops",
      "level": 1,
      "version": 2,
      "name": "商业运营",
      "nameEn": "Business Operations",
      "sortOrder": 80,
      "active": true
    },
    {
      "key": "education",
      "level": 1,
      "version": 2,
      "name": "教育学习",
      "nameEn": "Education",
      "sortOrder": 90,
      "active": true
    },
    {
      "key": "professional",
      "level": 1,
      "version": 2,
      "name": "行业专业",
      "nameEn": "Professional",
      "sortOrder": 100,
      "active": true
    },
    {
      "key": "it-ops-security",
      "level": 1,
      "version": 2,
      "name": "IT 运维与安全",
      "nameEn": "IT Ops & Security",
      "sortOrder": 110,
      "active": true
    },
    {
      "key": "life-service",
      "level": 1,
      "version": 2,
      "name": "生活服务",
      "nameEn": "Life Service",
      "sortOrder": 120,
      "active": true
    }
  ]
}
```

### Response 字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| count | integer | 分类总数 |
| items | array | 分类列表 |
| items[].key | string | 分类 key（唯一标识） |
| items[].level | integer | 层级（1 表示一级分类） |
| items[].version | integer | 版本号 |
| items[].name | string | 中文名 |
| items[].nameEn | string | 英文名 |
| items[].sortOrder | integer | 排序值 |
| items[].active | boolean | 是否启用 |

### 完整分类 key 列表

| key | 中文 | 英文 |
|-----|------|------|
| office-efficiency | 办公效率 | Office Efficiency |
| content-creation | 内容创作 | Content Creation |
| dev-programming | 开发编程 | Development |
| data-analysis | 数据分析 | Data Analysis |
| design-media | 设计多媒体 | Design & Media |
| ai-agent | AI Agent | AI Agent |
| knowledge-management | 知识管理 | Knowledge Management |
| business-ops | 商业运营 | Business Operations |
| education | 教育学习 | Education |
| professional | 行业专业 | Professional |
| it-ops-security | IT 运维与安全 | IT Ops & Security |
| life-service | 生活服务 | Life Service |

---

## 3. 当前用户信息

### 基本信息
- **Method**: `GET`
- **Path**: `/api/v1/auth/me`
- **用途**: 获取当前登录用户信息
- **认证**: 需要登录态（未登录返回 401）

### Request 示例
```
GET /api/v1/auth/me
```

### Response 示例

未登录：
```json
{
  "code": 401,
  "data": null,
  "message": "Unauthorized"
}
```

---

## 4. 飙升榜单

### 基本信息
- **Method**: `GET`
- **Path**: `/api/v1/showcase/trending`
- **用途**: 获取"近期飙升"分类的技能列表（首页侧边栏展示）

### Query 参数
无

### Request 示例
```
GET /api/v1/showcase/trending
```

### Response 示例
```json
{
  "section": "trending",
  "skills": [
    {
      "category": "ai-agent",
      "created_at": 1780476469390,
      "description": "Agently Mail 是 QQ 邮箱团队为 Agent 打造的专属邮箱服务...",
      "description_zh": "Agently Mail 是 QQ 邮箱团队为 Agent 打造的专属邮箱服务...",
      "downloads": 15838,
      "homepage": "https://api.skillhub.cn/u_d95b6787/agently-mail",
      "iconUrl": "https://cloudcache.tencent-cloud.com/qcloud/ui/static/other_external_resource/401a454b-4b16-4aee-a18c-4e3467ca48f1.png",
      "installs": 0,
      "labels": { "requires_api_key": "false" },
      "name": "Agently Mail",
      "ownerName": "u_d95b6787",
      "score": 14258.58521918025,
      "slug": "agently-mail",
      "source": "enterprise",
      "stars": 19,
      "subCategories": [{ "key": "agent-tool-use", "name": "工具调用" }],
      "updated_at": 1782879265944,
      "version": "1.0.8"
    }
  ]
}
```

### Response 字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| section | string | 区块标识，固定为 `trending` |
| skills | array | 飙升技能列表（数据结构同 `/api/skills` 返回的 skills 元素） |

---

## 5. 批量查询技能

### 基本信息
- **Method**: `POST`
- **Path**: `/api/v1/skills/batch`
- **用途**: 根据 slug 列表批量查询技能详细信息（含 owner、publisher、latestVersion、securityReports 等扩展字段）

### Request Body
```json
{
  "slugs": ["agently-mail", "ima-skills", "lingyi-wx-video-decomposer-exp"]
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| slugs | array[string] | 是 | slug 列表，最大 50 个 |

### Request 示例
```
POST /api/v1/skills/batch
Content-Type: application/json

{"slugs": ["agently-mail", "ima-skills", "westock-data", "cloudbase", "wxa-skills-generate", "tencent-agent-storage"]}
```

### Response 示例
```json
{
  "count": 6,
  "items": [
    {
      "latestVersion": {
        "changelog": "New version",
        "createdAt": 1782451182715,
        "version": "1.0.8"
      },
      "owner": {
        "displayName": "frankgqpeng(彭光前)",
        "handle": "u_d95b6787",
        "image": null
      },
      "publisher": {
        "name": "QQ邮箱",
        "logoUrl": null,
        "verified": true,
        "certifiedName": "腾讯科技（深圳）有限公司",
        "orgId": "org-bv6b8qcb"
      },
      "securityReports": {
        "keen": { "reportUrl": "https://tix.qq.com/...", "status": "benign", "statusText": "安全，无风险" },
        "sanbu": { "reportUrl": "https://static.cloudsec.tencent.com/...", "status": "benign", "statusText": "安全，无风险" }
      },
      "skill": {
        "authorVerifiedHandle": null,
        "category": "ai-agent",
        "claim_state": "unclaimed",
        "claimable": false,
        "claimed_user_handle": null,
        "createdAt": 1780476469390,
        "displayName": "Agently Mail",
        "iconUrl": "https://cloudcache.tencent-cloud.com/qcloud/ui/static/other_external_resource/401a454b-4b16-4aee-a18c-4e3467ca48f1.png",
        "isAuthorVerified": false,
        "labels": { "requires_api_key": "false" },
        "slug": "agently-mail",
        "source": "enterprise",
        "stats": { "comments": 0, "downloads": 15838, "installs": 0, "stars": 19, "versions": 0 },
        "subCategories": [{ "key": "agent-tool-use", "name": "工具调用" }],
        "summary": "Agently Mail 是 QQ 邮箱团队为 Agent 打造的专属邮箱服务...",
        "summary_zh": "",
        "tags": {},
        "updatedAt": 1782879265944
      }
    }
  ]
}
```

### Response 字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| count | integer | 返回条目数 |
| items | array | 技能详情列表 |

每个 item 包含：

| 字段 | 类型 | 说明 |
|------|------|------|
| latestVersion | object | 最新版本信息（changelog、version、createdAt） |
| owner | object | 所有者信息（displayName、handle、image） |
| publisher | object\|null | 发布者信息（name、logoUrl、verified、certifiedName、orgId） |
| securityReports | object | 安全报告（keen、sanbu），各含 reportUrl、status、statusText |
| skill | object | 技能基础信息（与 `/api/skills` 返回元素字段类似，但字段名采用 camelCase，如 createdAt、displayName、stats 等） |

---

## 排序与筛选完整示例

### 筛选：仅社区来源 + 仅需 API Key
```
GET /api/skills?page=1&pageSize=24&sortBy=rank&source=community&labels=requires_api_key%3Atrue
```

### 排序：按下载量降序 + 关键词搜索
```
GET /api/skills?page=1&pageSize=24&sortBy=downloads&order=desc&keyword=PPT
```

### 多条件：分类 + 子分类 + 关键词
```
GET /api/skills?page=1&pageSize=24&sortBy=downloads&order=desc&keyword=PPT&category=office-efficiency&subCategory=office-ppt
```

### 分页
```
GET /api/skills?page=2&pageSize=10&sortBy=updated_at&order=desc
```

---

## 支持的 sortBy 字段

| 字段 | 说明 |
|------|------|
| `updated_at` | 按更新时间排序 |
| `downloads` | 按下载量排序 |
| `stars` | 按收藏数排序 |
| `installs` | 按安装量排序 |
| `score` | 按评分排序（默认） |

> 传入其他字段（如 `trending`）会返回 400 错误：
> ```json
> {"code":400,"data":null,"message":"参数错误：sortBy 不支持（updated_at/downloads/stars/installs/score）"}
> ```

---

## source 字段可选值

| 值 | 含义 |
|----|------|
| `community` | 社区（用户上传） |
| `enterprise` | 企业（官方认证） |
| `clawhub` | ClawHub 来源 |

---

## 错误码说明

| code | 含义 |
|------|------|
| 0 | 成功 |
| 400 | 参数错误 |
| 401 | 未授权（需要登录） |

---

## 4. 首页轮播图

### 基本信息
- **Method**: `GET`
- **Path**: `/api/v1/banners`
- **用途**: 获取首页 Banner 轮播图

### Request 示例
```
GET /api/v1/banners
```

### Response 示例
```json
{
  "items": [
    {
      "id": 1,
      "title": "精选榜单专区",
      "subtitle": "热度飙升 · 实用精选 · Trace 测评 · 沙箱模拟对话",
      "imageUrl": "https://skillhub-1388575217.cos.accelerate.myqcloud.com/banners/1782471970286_july.jpg",
      "refType": "link",
      "refSlug": "",
      "linkTitle": "7月精选，有趣、实用的优质 Skill",
      "linkUrl": "https://skillhub.cn/areas/featured",
      "sortOrder": 0
    }
  ]
}
```

### 字段说明
| 字段 | 类型 | 说明 |
|------|------|------|
| id | integer | Banner ID |
| title | string | 标题 |
| subtitle | string | 副标题 |
| imageUrl | string | 图片 URL |
| refType | string | 跳转类型 |
| refSlug | string | 关联 slug |
| linkTitle | string | 链接标题 |
| linkUrl | string | 链接 URL |
| sortOrder | integer | 排序值 |

---

## 5. 热门下载榜单

### 基本信息
- **Method**: `GET`
- **Path**: `/api/v1/showcase/hot`
- **用途**: 获取"热门下载"区块的技能列表

### Response 示例
```json
{
  "section": "hot_downloads",
  "skills": [
    {
      "name": "self-improving agent",
      "slug": "self-improving-agent",
      "downloads": 921076,
      "stars": 4147,
      "score": 100000,
      "category": "ai-agent",
      "source": "clawhub",
      "labels": {"requires_api_key": "false"},
      "subCategories": [{"key": "agent-context", "name": "上下文管理"}],
      "version": "3.0.24",
      "ownerName": "pskoett",
      "iconUrl": "https://...",
      "homepage": "https://api.skillhub.cn/pskoett/self-improving-agent",
      "description": "...",
      "description_zh": "..."
    }
  ]
}
```

字段与 `/api/skills` 元素结构一致。

---

## 6. 推荐榜单

### 基本信息
- **Method**: `GET`
- **Path**: `/api/v1/showcase/recommended`
- **用途**: 获取"推荐"区块的技能列表

### Response
结构同 `/api/v1/showcase/hot`，`section` 字段为 `"recommended"`。

---

## 7. 最新榜单

### 基本信息
- **Method**: `GET`
- **Path**: `/api/v1/showcase/newest`
- **用途**: 获取"最新"区块的技能列表

### Response
结构同 `/api/v1/showcase/hot`，`section` 字段为 `"newest"`。

---

## 8. 飙升榜单

### 基本信息
- **Method**: `GET`
- **Path**: `/api/v1/showcase/trending`
- **用途**: 获取"近期飙升"区块的技能列表

### Response
结构同 `/api/v1/showcase/hot`，`section` 字段为 `"trending"`。

---

## 9. 技能详情

### 基本信息
- **Method**: `GET`
- **Path**: `/api/v1/skills/{slug}`
- **用途**: 获取单个技能的完整详情（与 `/api/v1/skills/batch` 元素结构类似）

### Request 示例
```
GET /api/v1/skills/web-tools-guide
```

### Response 示例
```json
{
  "contentZhAvailable": false,
  "latestVersion": {
    "changelog": "Initial release",
    "createdAt": 1776139672761,
    "version": "1.0.2"
  },
  "owner": {
    "displayName": "user_ec205dbb",
    "handle": "user_ec205dbb",
    "image": null
  },
  "securityReports": {
    "keen": {"reportUrl": "https://...", "status": "benign", "statusText": "安全，无风险"},
    "sanbu": {"reportUrl": "https://...", "status": "benign", "statusText": "安全，无风险"}
  },
  "skill": {
    "authorVerifiedHandle": null,
    "category": "knowledge-management",
    "claim_state": "unclaimed",
    "claimable": false,
    "claimed_user_handle": null,
    "createdAt": 1774842724122,
    "displayName": "web-tools-guide",
    "githubAuthorLogin": null,
    "iconUrl": "https://...",
    "isAuthorVerified": false,
    "labels": {"requires_api_key": "false"},
    "last_synced_at": null,
    "slug": "web-tools-guide",
    "source": "community",
    "sourceUrl": null,
    "stats": {"comments": 0, "downloads": 168194, "installs": 3459, "stars": 86, "versions": 2},
    "subCategories": [{"key": "knowledge-retrieval", "name": "信息检索"}],
    "summary": "",
    "summary_zh": "...",
    "tags": {"latest": "1.0.2"},
    "updatedAt": 1782879265914,
    "upstream_owner_login": null,
    "upstream_url": null,
    "verified": false
  }
}
```

### 字段说明
| 字段 | 类型 | 说明 |
|------|------|------|
| contentZhAvailable | boolean | 是否有中文描述 |
| latestVersion | object | 最新版本信息 |
| owner | object | 所有者信息 |
| securityReports | object | 安全报告（keen / sanbu） |
| skill | object | 技能基础信息 |

> 与 `/api/v1/skills/batch` 元素结构完全一致（返回字段使用 camelCase）。

---

## 10. 技能评测报告

### 基本信息
- **Method**: `GET`
- **Path**: `/api/v1/skills/{slug}/evaluation`
- **用途**: 获取 AI 评测报告（含多维度评分、详细点评、用户摘要）

### Request 示例
```
GET /api/v1/skills/web-tools-guide/evaluation
```

### Response 示例
```json
{
  "createdAt": 1779320593747,
  "dimensions": {
    "adaptability": {
      "items": {
        "boundary": {
          "reason": "该 Skill 清晰定义了四个工具的能力边界...",
          "score": 4.5,
          "userReason": "工具分工明确..."
        },
        "trigger": {
          "reason": "触发方式定义在 SKILL.md 的 description 字段...",
          "score": 4,
          "userReason": "触发条件基本合理..."
        }
      },
      "reason": "该 Skill 在能力边界定义方面表现较好...",
      "userReason": "整体质量良好..."
    },
    "convention": { "...": "..." },
    "effectiveness": { "...": "..." },
    "reliability": { "...": "..." },
    "trust": { "...": "..." }
  },
  "skillId": 26725,
  "summary": "这是一个结构完整、覆盖面广的 Web 工具策略指南...",
  "updatedAt": 1779320617165,
  "userSummary": "这个 Skill 质量较好，工具使用策略清晰...",
  "versionId": 39699
}
```

### 字段说明
| 字段 | 类型 | 说明 |
|------|------|------|
| createdAt | integer | 创建时间（毫秒时间戳） |
| updatedAt | integer | 更新时间 |
| skillId | integer | 技能 ID |
| versionId | integer | 版本 ID |
| summary | string | 整体评测总结 |
| userSummary | string | 用户角度摘要 |
| dimensions | object | 多维度详细评分（含 adaptability / convention / effectiveness / reliability / trust，每个维度下有 items、reason、userReason） |

---

## 11. 版本列表

### 基本信息
- **Method**: `GET`
- **Path**: `/api/v1/skills/{slug}/versions`
- **用途**: 获取技能所有历史版本

### Request 示例
```
GET /api/v1/skills/web-tools-guide/versions
```

### Response 示例
```json
{
  "slug": "web-tools-guide",
  "source": "community",
  "versions": [
    {
      "changelog": "Initial release",
      "createdAt": 1776139672761,
      "securityReports": {
        "keen": {"status": "benign", "statusText": "安全，无风险", "reportUrl": "https://..."},
        "sanbu": {"status": "benign", "statusText": "安全，无风险", "reportUrl": "https://..."}
      },
      "version": "1.0.2",
      "versionId": 39699
    },
    {
      "changelog": "Synced by skillhub pipeline",
      "createdAt": 1774842724122,
      "securityReports": {
        "keen": {"status": "benign", "statusText": "安全，无风险", "reportUrl": ""},
        "sanbu": {"status": "suspicious", "statusText": "可疑，存在潜在风险", "reportUrl": ""}
      },
      "version": "1.0.0",
      "versionId": 30665
    }
  ]
}
```

### 字段说明
| 字段 | 类型 | 说明 |
|------|------|------|
| slug | string | 技能 slug |
| source | string | 来源 |
| versions | array | 版本列表 |
| versions[].version | string | 版本号 |
| versions[].versionId | integer | 版本 ID |
| versions[].changelog | string | 变更说明 |
| versions[].createdAt | integer | 发布时间（毫秒时间戳） |
| versions[].securityReports | object | 各家安全报告 |

---

## 12. 版本文件清单

### 基本信息
- **Method**: `GET`
- **Path**: `/api/v1/skills/{slug}/files`
- **用途**: 获取指定版本下的所有文件清单（含 sha256、size）

### Query 参数
| 参数 | 必填 | 说明 |
|------|------|------|
| version | 是 | 版本号 |

### Request 示例
```
GET /api/v1/skills/web-tools-guide/files?version=1.0.2
```

### Response 示例
```json
{
  "count": 5,
  "files": [
    {"path": "SKILL.md", "sha256": "8efcac4e1dc8c4e2e4d5a3af4c283f20fa2c2e25d9e8020e21aff5034ba22f1a", "size": 6255},
    {"path": "scripts/setup-opencli.sh", "sha256": "8881aebc3ec4cabafa9e0e89df515bc5e0534a76a284bfa832bd6bdf968a41c7", "size": 12494},
    {"path": "references/opencli-guide.md", "sha256": "c754293ba14e4134e9ec7e64fc829c6b9f7acecb90742616e8c7348d6396a2bf", "size": 3303},
    {"path": "references/well-known-sites.json", "sha256": "5ce4563ee8e68385beeeb5db891262be7c60016e41af89e84ce6626c092a0791", "size": 8100},
    {"path": "references/web-search-config.md", "sha256": "4434bf31eb4cc27bd1d224c9ecc52be04fac2179aab38647f7a8d57cedd5db90", "size": 4227}
  ],
  "version": "1.0.2"
}
```

### 字段说明
| 字段 | 类型 | 说明 |
|------|------|------|
| count | integer | 文件总数 |
| version | string | 版本号 |
| files[].path | string | 相对路径 |
| files[].sha256 | string | 文件 SHA-256 |
| files[].size | integer | 文件字节数 |

---

## 13. 单个文件（302 跳转）

### 基本信息
- **Method**: `GET`
- **Path**: `/api/v1/skills/{slug}/file`
- **用途**: 获取指定版本下的单个文件内容，**返回 302 跳转到 COS 真实地址**

### Query 参数
| 参数 | 必填 | 说明 |
|------|------|------|
| path | 是 | 相对路径（如 `SKILL.md`） |
| version | 是 | 版本号 |

### Request 示例
```
GET /api/v1/skills/web-tools-guide/file?path=SKILL.md&version=1.0.2
```

### Response
```
HTTP/1.1 302 Found
Location: https://skillhub-1388575217.cos.accelerate.myqcloud.com/skills/web-tools-guide/1.0.2/files/SKILL.md
Content-Disposition: inline
Cache-Control: public, immutable, max-age=31536000
X-Content-Sha256: 8efcac4e1dc8c4e2e4d5a3af4c283f20fa2c2e25d9e8020e21aff5034ba22f1a
X-Content-Size: 6255
```

### 响应头说明
| 头 | 说明 |
|------|------|
| Location | COS 真实下载地址 |
| Content-Disposition | 内联显示（`inline`） |
| Cache-Control | 公共、不可变、缓存 1 年 |
| X-Content-Sha256 | 内容 SHA-256 |
| X-Content-Size | 内容字节数 |

---

## 14. 整包下载（302 跳转）

### 基本信息
- **Method**: `GET`
- **Path**: `/api/v1/download`
- **用途**: 下载整个技能包（zip），**返回 302 跳转到 COS 真实地址**

### Query 参数
| 参数 | 必填 | 说明 |
|------|------|------|
| slug | 是 | 技能 slug |

### Request 示例
```
GET /api/v1/download?slug=web-tools-guide
```

### Response
```
HTTP/1.1 302 Found
Location: https://skillhub-1388575217.cos.accelerate.myqcloud.com/skills/web-tools-guide/1.0.2.zip
Content-Disposition: attachment; filename="web-tools-guide-1.0.2.zip"
```

> 实际文件名格式：`{slug}-{version}.zip`，使用最新版本。
> **注意**：此接口会自增该技能的下载量统计。

---

## 15. 数字签名

### 基本信息
- **Method**: `GET`
- **Path**: `/api/v1/open/skills/{slug}/versions/{ver}/signature`
- **用途**: 获取指定版本的数字签名（用于验证包完整性和来源）

### Request 示例
```
GET /api/v1/open/skills/web-tools-guide/versions/1.0.2/signature
```

### Response 示例
```json
{
  "content_hash": "2ab93d0014fcc1ef1c31ffd23e8ddc52cc4ce052493304849bbbda8cf1cfbb58",
  "hash_version": 1,
  "key_id": "skillhub-platform-v1",
  "payload": "{\"content_hash\":\"...\",\"file_count\":5,\"issued_at\":1781786869776,\"issuer\":\"skillhub.cn\",\"package_md5\":\"27490156f984a12b77b3c0dbe66995be\",\"publisher_id\":433976,\"publisher_real_name_verified\":true,\"publisher_user_name\":\"user_ec205dbb\",\"skill_slug\":\"web-tools-guide\",\"skill_version\":\"1.0.2\",\"v\":1,\"version_id\":39699}",
  "signature": "y0dufbraRDxaDSnYI6QUHHbUAe9wyxQfYHNZiv1v/7qTZ8XNZ4Vcc/X4uXOuSol0ApDQT8KIMinC9dz3FKcLBg==",
  "signed": true,
  "signed_at": 1781786869776
}
```

### 字段说明
| 字段 | 类型 | 说明 |
|------|------|------|
| content_hash | string | 包内容 SHA-256 |
| hash_version | integer | 哈希版本 |
| key_id | string | 签名密钥 ID |
| payload | string | 载荷 JSON（base64 安全字符） |
| signature | string | 签名值 |
| signed | boolean | 是否已签名 |
| signed_at | integer | 签名时间戳 |

> 验证文档参考：https://skillhub.cn/docs/verify-signature?slug=...&version=...

---

## 16. 版本 diff 总览

### 基本信息
- **Method**: `GET`
- **Path**: `/api/v1/skills/{slug}/diff`
- **用途**: 对比两个版本的文件变更总览（不返回内容）

### Query 参数
| 参数 | 必填 | 说明 |
|------|------|------|
| base | 是 | 基准版本 |
| target | 是 | 目标版本 |

### Request 示例
```
GET /api/v1/skills/web-tools-guide/diff?base=1.0.0&target=1.0.2
```

### Response 示例
```json
{
  "base": "1.0.0",
  "target": "1.0.2",
  "slug": "web-tools-guide",
  "summary": {
    "added": 2,
    "removed": 1,
    "changed": 2,
    "unchanged": 1
  },
  "files": [
    {
      "path": "SKILL.md",
      "status": "changed",
      "baseSize": 4767,
      "targetSize": 6255,
      "baseSha256": "404cd9470dbf632c8e6001fca2c78d42dfc264622e37e6060bb99dd550b5e7e9",
      "targetSha256": "8efcac4e1dc8c4e2e4d5a3af4c283f20fa2c2e25d9e8020e21aff5034ba22f1a"
    },
    {
      "path": "_meta.json",
      "status": "removed",
      "baseSize": 134,
      "targetSize": null,
      "baseSha256": "ddcb2d1170795ec9f6b26e2ec931bda009c85b063d68dec98d862b17477386fe",
      "targetSha256": null
    }
  ]
}
```

### 字段说明
| 字段 | 类型 | 说明 |
|------|------|------|
| base | string | 基准版本号 |
| target | string | 目标版本号 |
| slug | string | 技能 slug |
| summary | object | 变更统计（added/removed/changed/unchanged） |
| files[].path | string | 相对路径 |
| files[].status | string | 状态：`added` / `removed` / `changed` / `unchanged` |
| files[].baseSize | integer\|null | 基准版本文件大小 |
| files[].targetSize | integer\|null | 目标版本文件大小 |
| files[].baseSha256 | string\|null | 基准版本 SHA-256 |
| files[].targetSha256 | string\|null | 目标版本 SHA-256 |

---

## 17. 版本 diff 文件内容

### 基本信息
- **Method**: `GET`
- **Path**: `/api/v1/skills/{slug}/diff/file`
- **用途**: 获取两个版本指定文件的 diff 详情（含完整文本内容）

### Query 参数
| 参数 | 必填 | 说明 |
|------|------|------|
| base | 是 | 基准版本 |
| target | 是 | 目标版本 |
| path | 是 | 相对路径 |

### Request 示例
```
GET /api/v1/skills/web-tools-guide/diff/file?base=1.0.0&target=1.0.2&path=SKILL.md
```

### Response 示例
```json
{
  "slug": "web-tools-guide",
  "base": "1.0.0",
  "target": "1.0.2",
  "path": "SKILL.md",
  "status": "changed",
  "tooLarge": false,
  "baseFile": {
    "exists": true,
    "size": 4767,
    "sha256": "404cd9470dbf632c8e6001fca2c78d42dfc264622e37e6060bb99dd550b5e7e9",
    "text": "完整文本内容..."
  },
  "targetFile": {
    "exists": true,
    "size": 6255,
    "sha256": "8efcac4e1dc8c4e2e4d5a3af4c283f20fa2c2e25d9e8020e21aff5034ba22f1a",
    "text": "完整文本内容..."
  }
}
```

### 字段说明
| 字段 | 类型 | 说明 |
|------|------|------|
| slug | string | 技能 slug |
| base | string | 基准版本号 |
| target | string | 目标版本号 |
| path | string | 文件路径 |
| status | string | 变更状态 |
| tooLarge | boolean | 内容是否过大（过大可能不返回 text 字段） |
| baseFile | object\|null | 基准版本文件（exists/size/sha256/text） |
| targetFile | object\|null | 目标版本文件 |

---

## 18. 相关推荐

### 基本信息
- **Method**: `GET`
- **Path**: `/api/v1/skills/{slug}/recommendations`
- **用途**: 获取与该技能相关的其他技能推荐（用于"相关推荐"区块）

### Query 参数
| 参数 | 必填 | 说明 |
|------|------|------|
| pageSize | 否 | 每页条数 |
| cursor | 否 | 分页游标 |

### Request 示例
```
GET /api/v1/skills/web-tools-guide/recommendations?pageSize=3
```

### Response 示例
```json
{
  "hasMore": true,
  "items": [
    {
      "category": "knowledge-management",
      "displayName": "Baidu web search",
      "downloads": 108822,
      "homepage": "https://api.skillhub.cn/ide-rea/baidu-search",
      "iconUrl": "https://...",
      "ownerHandle": "ide-rea",
      "slug": "baidu-search",
      "stars": 246,
      "summary": "Search the web using Baidu AI Search Engine (BDSE)...",
      "summaryZh": "使用百度AI搜索引擎(BDSE)进行网络搜索..."
    }
  ],
  "nextCursor": "eyJvZmZzZXQiOjMsInNlZWQiOjE3ODI4Nzk0NzQ5NjYwNzk4Mjd9",
  "traceId": "97c1e4724acef63744a9a83102244147"
}
```

### 字段说明
| 字段 | 类型 | 说明 |
|------|------|------|
| hasMore | boolean | 是否还有更多 |
| items | array | 推荐技能列表（精简版字段） |
| nextCursor | string | 下一页游标（base64 编码 JSON） |
| traceId | string | 追踪 ID |

---

## 19. 批量查询技能

### 基本信息
- **Method**: `POST`
- **Path**: `/api/v1/skills/batch`
- **用途**: 根据 slug 列表批量查询技能详细信息（含 owner、publisher、latestVersion、securityReports 等扩展字段）

### Request Body
```json
{
  "slugs": ["agently-mail", "ima-skills", "lingyi-wx-video-decomposer-exp"]
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| slugs | array[string] | 是 | slug 列表，最大 50 个 |

### Request 示例
```
POST /api/v1/skills/batch
Content-Type: application/json

{"slugs": ["agently-mail", "ima-skills", "westock-data", "cloudbase", "wxa-skills-generate", "tencent-agent-storage"]}
```

### Response 示例
```json
{
  "count": 6,
  "items": [
    {
      "latestVersion": {"changelog": "New version", "createdAt": 1782451182715, "version": "1.0.8"},
      "owner": {"displayName": "frankgqpeng(彭光前)", "handle": "u_d95b6787", "image": null},
      "publisher": {"name": "QQ邮箱", "logoUrl": null, "verified": true, "certifiedName": "腾讯科技（深圳）有限公司", "orgId": "org-bv6b8qcb"},
      "securityReports": {
        "keen": {"reportUrl": "https://tix.qq.com/...", "status": "benign", "statusText": "安全，无风险"},
        "sanbu": {"reportUrl": "https://static.cloudsec.tencent.com/...", "status": "benign", "statusText": "安全，无风险"}
      },
      "skill": {
        "category": "ai-agent",
        "displayName": "Agently Mail",
        "iconUrl": "https://...",
        "isAuthorVerified": false,
        "labels": {"requires_api_key": "false"},
        "slug": "agently-mail",
        "source": "enterprise",
        "stats": {"comments": 0, "downloads": 15838, "installs": 0, "stars": 19, "versions": 0},
        "subCategories": [{"key": "agent-tool-use", "name": "工具调用"}],
        "summary": "...",
        "summary_zh": "",
        "tags": {},
        "updatedAt": 1782879265944
      }
    }
  ]
}
```

### 字段说明
每个 item 包含：

| 字段 | 类型 | 说明 |
|------|------|------|
| latestVersion | object | 最新版本信息（changelog / version / createdAt） |
| owner | object | 所有者信息（displayName / handle / image） |
| publisher | object\|null | 发布者信息（name / logoUrl / verified / certifiedName / orgId） |
| securityReports | object | 安全报告（keen / sanbu） |
| skill | object | 技能基础信息（camelCase 字段） |

---

## 20. 大赛信息

### 基本信息
- **Method**: `GET`
- **Path**: `/api/v1/contest/info`
- **用途**: 获取当前"一人公司挑战赛"基本信息

### Request 示例
```
GET /api/v1/contest/info
```

### Response 示例
```json
{
  "code": 0,
  "data": {
    "awards": {
      "top10": {"count": 10, "description": "TRASCE 评分 Top 10"},
      "top30": {"count": 30, "description": "TRASCE 评分 Top 30"}
    },
    "contestId": "2026-s1",
    "endDate": 1780632000000,
    "name": "一人公司挑战赛",
    "nextSeasonStart": 0,
    "resultDate": 1780632000000,
    "seasonNumber": 1,
    "sponsors": ["腾讯轻量云", "腾讯新闻", "腾讯玄武实验室"],
    "startDate": 1779336000000,
    "stats": {"entryCount": 447, "totalAuthors": 311},
    "status": "result",
    "submissionEndDate": 1780372800000
  },
  "message": "success"
}
```

### 字段说明
| 字段 | 类型 | 说明 |
|------|------|------|
| contestId | string | 大赛 ID |
| name | string | 大赛名称 |
| seasonNumber | integer | 赛季数 |
| startDate | integer | 开始时间 |
| endDate | integer | 结束时间 |
| submissionEndDate | integer | 投稿截止时间 |
| resultDate | integer | 公布时间 |
| nextSeasonStart | integer | 下届开始时间 |
| status | string | 状态 |
| sponsors | array[string] | 赞助商 |
| awards | object | 奖项设置 |
| stats | object | 统计 |

---

## 21. 技能参赛状态

### 基本信息
- **Method**: `GET`
- **Path**: `/api/v1/contest/skills/{slug}/joined`
- **用途**: 查询某技能是否已加入当前大赛

### Request 示例
```
GET /api/v1/contest/skills/web-tools-guide/joined
```

### Response 示例
```json
{
  "code": 0,
  "data": {
    "joined": false,
    "slug": "web-tools-guide"
  },
  "message": "success"
}
```

### 字段说明
| 字段 | 类型 | 说明 |
|------|------|------|
| joined | boolean | 是否已参赛 |
| slug | string | 技能 slug |

---

## 22. 大赛 Top 榜

### 基本信息
- **Method**: `GET`
- **Path**: `/api/v1/contest/top`
- **用途**: 获取当前大赛的 Top 10 / Top 30 排行榜

### Request 示例
```
GET /api/v1/contest/top
```

### Response 示例
```json
{
  "code": 0,
  "data": {
    "top10": [
      {
        "rank": 1,
        "slug": "tencent-novnc-chromium-cdp",
        "name": "🔥控制浏览器做任何自动化🔥...",
        "ownerName": "user_f72e84b4",
        "ownerNickname": "乐涩辞",
        "category": "ai-agent",
        "iconUrl": "https://...",
        "downloads": 4881,
        "stars": 160,
        "joinedAt": 1779534683888,
        "trasceScore": 95.86,
        "summary": "...",
        "summaryZh": "..."
      }
    ],
    "top30": [
      {"...": "..."}
    ]
  },
  "message": "success"
}
```

### 字段说明
| 字段 | 类型 | 说明 |
|------|------|------|
| rank | integer | 排名 |
| slug | string | 技能 slug |
| name | string | 技能名称 |
| ownerName | string | 所有者 handle |
| ownerNickname | string | 所有者昵称 |
| category | string | 一级分类 key |
| iconUrl | string | 图标 |
| downloads | integer | 下载量 |
| stars | integer | 收藏数 |
| joinedAt | integer | 参赛时间 |
| trasceScore | number | TRASCE 评分 |
| summary / summaryZh | string | 描述 |

---

## 23. 微信扫码登录

### 基本信息
- **Method**: `GET`
- **Path**: `/api/v1/auth/wechat/qrcode`
- **用途**: 获取微信扫码登录二维码 URL 和 state 标识
- **触发场景**: 点击"收藏"、"发布 Skill"、"登录"等需要登录的操作

### Request 示例
```
GET /api/v1/auth/wechat/qrcode
Cookie: sid=...
```

### Response 示例
```json
{
  "authUrl": "https://open.weixin.qq.com/connect/qrconnect?appid=wx15f09c62e99e2b64&redirect_uri=https%3A%2F%2Fapi.skillhub.cn%2Fapi%2Fv1%2Fauth%2Fwechat%2Flogin%2Fcallback&response_type=code&scope=snsapi_login&state=st_8993518d2e8994399106c9f96dd647bb#wechat_redirect",
  "state": "st_8993518d2e8994399106c9f96dd647bb"
}
```

### 字段说明
| 字段 | 类型 | 说明 |
|------|------|------|
| authUrl | string | 微信开放平台授权 URL（含 state 参数） |
| state | string | 状态标识，登录回调时携带 |

> 实际登录流程：浏览器跳转到 authUrl → 用户扫码 → 微信回调到 `redirect_uri` → 设置 cookie `sid`。
> 后续请求需在 cookie 中携带 `sid` 才能通过 `/api/v1/auth/me` 鉴权。

---

## 备注

1. 所有接口均支持 CORS，仅允许 `https://skillhub.cn` 来源
2. `/api/v1/auth/me` 和需要登录的接口（如收藏）依赖 cookie `sid`，未登录返回 401
3. `/api/v1/skills/batch` 单次最多 50 个 slug
4. `labels` 字段过滤支持任意自定义 key/value 对（`requires_api_key` 是已知 key）
5. **文件下载流程**：
   - 调用 `/api/v1/download?slug=...` 拿到 302 Location（zip 完整包）
   - 或调用 `/api/v1/skills/{slug}/file?path=...&version=...` 拿到 302 Location（单文件）
   - 也可直接用 `/api/v1/skills/{slug}/files?version=...` 拿到文件 sha256 自取 COS（`https://skillhub-{cosId}.cos.accelerate.myqcloud.com/skills/{slug}/{ver}/files/{path}`）
6. **签名验证流程**：
   - 拉取 `/api/v1/open/skills/{slug}/versions/{ver}/signature`
   - 用 `key_id="skillhub-platform-v1"` 的公钥校验 `signature` 签名的 `payload` 字符串
   - payload 内 `package_md5` = `hash(md5(sorted(file.path+file.sha256)))`，与下载包 md5 一致
7. 抓包时间：2026-07-01，接口行为可能随版本变化
8. COS 域名格式：`https://skillhub-1388575217.cos.accelerate.myqcloud.com/...`
9. `slug` 路径中可能含中文等多字节字符，需 URL encode
