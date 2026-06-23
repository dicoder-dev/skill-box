# Conventions(命名 / 目录 / 文件规范)

> 改代码前必读。与具体业务无关,只装硬规则。

## 目录

- 后端业务包路径:`api-server/internal/gapi/controller/<业务>/c<业务>/`、`s<业务>/`、`model/<业务>/`、`entity/`
- 前端 store:`frontend/src/store/`,**不要散落到 `views/` 或 `components/`**
- 前端 API 封装:`frontend/src/api/`(Wails 绑定由 wails3 generate,别手改)
- 前端平台抽象:`frontend/src/platform/`,业务侧只 `import '@/platform'`

## Go 命名

- 包名小写、不带分隔符:`cuser / suser / ginp / cfg / dbops`
- 一个 API 一个文件:`get_user_info.go`,**不写 `user.go` 这种聚合**
- 实体名驼峰:`User / SysRole`,表名 `users / sys_roles`(GORM 自动推导)
- 控制器方法:`func Search(c *ginp.ContextPlus, params *ReqSearch)`
- 服务方法:接收 `*Model()`,返回 `(data, error)` 或 `(data, total, error)`

## Go 风格

- `gofmt` + `goimports` 标准格式(Tab 缩进)
- 包级 doc comment(每个包首行 `// Package xxx ...`)
- 注释风格与目标文件保持一致(仓库内中英混用,跟随上下文)
- 配置结构体用 `init()` + `cfg.ParseConfigStruct(...)`,**不要在 `main.go` 里写加载逻辑**

## Vue 风格

- `<script setup>` SFC,2 空格缩进
- props / emits 显式声明
- 组件名 PascalCase,文件名 kebab-case(`UserCard.vue` → `<user-card>`)
- 业务组件放 `frontend/src/components/`,**不要散落到 `views/`**

## HTTP 风格

- 后端:`c.SuccessData(data)` / `c.Fail(msg, code?)`,**不要直接 `c.JSON`**
- 前端:`import { http } from '@/core/utils/requests'`,**不要直接 fetch**
- 业务码:成功 `code === 1`,失败抛 `BusinessError`
- 401 处理在拦截器里,业务侧不写

## 配置风格

- YAML 字段 ↔ struct 字段用 `configkey` 标签(无标签则按字段名)
- 默认值用 `default` 标签,不要在加载后写一堆 fallback
- 新增配置项:`api-server/configs/<x>.go` 加 struct → `cfg.ParseConfigStruct` 自动注册

## 数据库

- 不写 SQL,统一走 `pkg/dbops.NewBaseDb` 的 `FindOne / FindList / Create / Update / Delete`
- 复杂查询用 `pkg/where`:`where.Format(where.OptEqual(...), where.OptLike(...))`
- 读 / 写库分离:`tables.NewUser(wdb, rdb)`,业务层不感知

## 文档

- 中文为主,代码 / 接口签名保留英文
- 业务文档放 `docs/project/`,AI 协作上下文放 `docs/agent/`,**不要混**
- 改了 `docs/project/` 里的内容要在 PR 描述写清影响