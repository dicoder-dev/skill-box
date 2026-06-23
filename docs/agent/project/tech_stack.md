# Tech Stack(版本与约束)

## 后端

| 依赖 | 版本 | 备注 |
| --- | --- | --- |
| Go | 1.25+ | 根 `go.mod` 与 `api-server/go.mod` 都要满足 |
| Wails | v3 | `go install github.com/wailsapp/wails/v3/cmd/wails3@latest` |
| Gin | 通过 `api-server/pkg/ginp` 封装 | 不要直接 import `github.com/gin-gonic/gin` 的 Context,统一用 `ginp.ContextPlus` |
| GORM | v2 | 多驱动(sqlite / mysql / pgsql),通过 `db.use_type` 切换 |
| Viper | 最新 | 配置加载;`cfg.GetBool` 已支持 `true/false/yes/no/ok/ng/on/off/0/1` |
| jwx | 最新 | JWKS 鉴权(`AuthUserCenterMiddleware`) |

## 前端

| 依赖 | 版本 | 备注 |
| --- | --- | --- |
| Node | 18+ | |
| Vue | 3.x | `<script setup>` SFC,2 空格缩进 |
| Pinia | 最新 | store 在 `frontend/src/store/`,**不要散落别处** |
| Vite | 最新 | dev 默认 5173;`WAILS_VITE_PORT` 可覆盖 |
| 平台能力 | `@/platform` 抽象 | Web / Desktop 双实现,**业务侧只 import 这一份** |
| HTTP 层 | `@/core/utils/requests` | 自带拦截器栈 + 业务码剥离 |

## 构建

| 工具 | 安装 | 用途 |
| --- | --- | --- |
| Task | `go install github.com/go-task/task/v3/cmd/task@latest` | 跨平台任务编排 |
| Wails CLI v3 | 见上 | 桌面端 build / dev |

## 数据库驱动

- **SQLite**:开箱即用,默认 `data.db`,已 gitignore
- **MySQL**:`db.use_type=mysql` + 填连接信息
- **PostgreSQL**:`db.use_type=pgsql`

新增数据库类型需要改:
1. `api-server/pkg/dbops/base_db_new.go`(驱动初始化)
2. `api-server/internal/db/dbs`(同步 type)
3. `api-server/configs/db.go`(配置项 + use_type 支持新值)

## 禁止 / 谨慎

- 不要手动改 `frontend/bindings/` 与 `frontend/src/api/`(Wails 生成)
- 不要在 `api-server/cmd/web/frontend/dist/` 直接改文件(同步目录,会被覆盖)
- 不要绕过 `configs.yaml` 直接读环境变量(配置走 cfg 单点)
- 升级 Go 主版本前先确认 `Wails v3` 与 `GORM v2` 兼容性