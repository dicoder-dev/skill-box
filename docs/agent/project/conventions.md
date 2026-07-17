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

## AI 生成代码规范(Claude 必须遵守的设计原则)

> 这部分是"AI 怎么写代码"的硬规则,与上面的命名/目录规范并列。

1. **优先封装,保证可复用性**
   - 出现 ≥2 次的逻辑,必须抽成独立函数/组件/模块,不要复制粘贴
   - 抽离时同步考虑参数化(传参 > 复制代码块)
   - 工具类/常量/枚举集中在 `core/`、`common/` 或对应业务的 `pkg/` 下,避免散落

2. **禁止硬编码,优先常量**
   - 魔法数字、状态值、错误码、URL、key 名等一律抽常量
   - 状态/枚举走 `iota` 或显式命名常量,不写裸 `0/1/2`
   - 文案/提示信息走 i18n,业务侧不写死中文/英文字符串

3. **区分全局与局部**
   - **全局**:跨业务、跨模块、多人复用的放 `core/` / `common/` / `pkg/` / `shared/`(例:http 客户端、日志、错误处理、通用工具)
   - **局部**:只在本业务/本模块用的就近放,不要往全局公共里塞
   - 判断标准:**至少 2 个业务/模块会用 → 全局;只有 1 处用 → 局部**
   - 犹豫时优先局部,等出现第二处复用再上提

4. **遵循设计模式,不要一股脑写死**
   - 复杂逻辑先想清楚再下手:**分层(Controller / Service / Repository)、单一职责、依赖注入** 是底线
   - 重复 `if-else`/类型分支 → 策略模式 / 多态 / map dispatch
   - 复杂对象构造 → 工厂 / Builder;有状态流程 → 状态机
   - 不要"打开文件直接从头写到尾",先列骨架再填肉

## 文档

- 中文为主,代码 / 接口签名保留英文
- 业务文档放 `docs/project/`,AI 协作上下文放 `docs/agent/`,**不要混**
- 改了 `docs/project/` 里的内容要在 PR 描述写清影响