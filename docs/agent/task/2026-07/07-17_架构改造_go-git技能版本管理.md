# go-git 技能版本管理架构改造

**日期:** 2026-07-17
**状态:** 进行中(方案设计阶段)

---

## 1. 需求

### 1.1 用户原始诉求
> 当前项目我想使用 go-git 的一个 Git 库,主要是用于版本管理,包括同步技能等多项功能。
> 我想把当前的方案全部替换掉,全部换成基于文件的、基于 go-git 的版本管理,而不是基于数据库的版本管理。
> 同时也可以把整个项目同步到 GitHub 等远端仓库,这样方便我们去同步我们的技能。
> ~/.skill-box/skills  把这个目录作为一整个 git 仓库。

### 1.2 需求细化

1. **彻底替换**:`skill_tags` / `skill_file_snapshots` 两张 DB 表 + `ctag/*` 5 个 API + `internal/skillaudit` 包 + 前端 `tags.js` 客户端,**全部废弃**,改用 `go-git` 直接操作 `~/.skill-box/skills/.git/`。
2. **单仓**:整个 `~/.skill-box/skills/` 是**一个** git 仓库(用户选 monorepo 模式),不是 per-skill 独立仓。
3. **自动同步**:Create / Update / Delete / Move / Rename / Group 任意操作,落盘成功后自动 `git add + commit + push`;push 失败不阻断业务流程,异步重试。
4. **远端认证**:HTTPS + GitHub PAT(BasicAuth),token 加密存在 `~/.skill-box/.git_token`(0600 权限)。
5. **前端完全替换**:Tags 弹窗整个删掉,改为新的 "Versions / History" 弹窗(commit 时间线 + diff + checkout + push 状态)。

### 1.3 与现有功能的关系

| 现状 | 改造后 | 备注 |
|---|---|---|
| `skillstore.Store` 写 `~/.skill-box/skills/<group>/<name>/` | 不变,只是新加一层 git hook | `store.Save` 末尾触发 git commit |
| `ctag/create_tag` API | 删除 | 用 commit message 替代 tag name |
| `ctag/list_tags` | 删除 | 用 `git log` 替代 |
| `ctag/diff_tag` | 删除(或者保留 URL,内部换 git diff) | UI 改成 commit vs HEAD |
| `ctag/rollback_tag` | 删除(或者保留 URL,内部换 git checkout --hard) | 加 confirmation |
| `ctag/delete_tag` | 删除(或者 git reset 替代) | 改用 revert commit |
| `skill_tags` / `skill_file_snapshots` 表 | 表保留数据可读,但新代码不写 | DB migration 不 drop(避免破坏性) |
| `internal/skillaudit` 包 | 整个废弃 | 仅 `internal/skillversion` 替代 |

---

## 2. 架构设计

### 2.1 目录布局

```
~/.skill-box/                                       ← 数据根
├── skills/                                         ← git 仓库根
│   ├── .git/                                       ← go-git 初始化在这里
│   ├── .gitignore                                  ← 忽略临时文件(见下)
│   ├── .skillbox-meta/                             ← 不入库的本地元数据
│   │   └── last_commit_<group>_<name>.txt          ← 每次提交后记录 head hash
│   ├── group-a/
│   │   ├── skill-1/
│   │   │   ├── SKILL.md
│   │   │   └── scripts/
│   │   └── skill-2/
│   └── group-b/
│       └── skill-3/
├── config.yaml                                     ← git remote / branch / token_file
└── .git_token                                      ← GitHub PAT,0600
```

### 2.2 核心包结构

新增包:

| 包 | 路径 | 职责 |
|---|---|---|
| `internal/skillversion` | `api-server/internal/skillversion/` | go-git 封装:Init / Commit / Log / Diff / Checkout / Reset / Push / Pull / Fetch |
| `internal/skillversion/gitconfig` | 同上子包 | 远端配置加载/保存/校验(token 路径、URL 合法性) |
| `internal/skillversion/hook` | 同上子包 | 在 `skillstore.Save` 末尾注入 commit 回调 |

### 2.3 关键 API 边界

```go
// internal/skillversion/repo.go
type Repo struct {
    path string  // ~/.skill-box/skills
    mu   sync.Mutex  // 进程内 git 操作串行
}

func Open(root string) (*Repo, error)
func Init(root string) (*Repo, error)  // PlainInit + 默认 branch + .gitignore
func (r *Repo) Status() (git.Status, error)
func (r *Repo) Commit(msg string, files []string) (plumbing.Hash, error)
func (r *Repo) Log(opt LogOptions) ([]CommitEntry, error)
func (r *Repo) Diff(a, b string) (string, error)  // a..b 之间的 diff
func (r *Repo) Checkout(commit string) error  // git checkout <commit> -- .
func (r *Repo) Reset(commit string, mode ResetMode) error
func (r *Repo) Push(ctx context.Context, auth *http.BasicAuth) error
func (r *Repo) Pull(ctx context.Context, auth *http.BasicAuth) error
func (r *Repo) Fetch(ctx context.Context, auth *http.BasicAuth) error
func (r *Repo) RemoteInfo() (url, branch string, ok bool)
func (r *Repo) SetRemote(url, branch string) error
func (r *Repo) IsConfigured() bool
```

```go
// internal/skillversion/gitconfig/config.go
type Config struct {
    RemoteURL string `yaml:"remote_url"`
    Branch    string `yaml:"branch"`  // 默认 "main"
    TokenFile string `yaml:"token_file"`
}

func Load(dataDir string) (*Config, error)
func Save(dataDir string, cfg *Config) error
func (c *Config) Validate() error
func (c *Config) LoadAuth() (*http.BasicAuth, error)  // 读 token 文件
```

### 2.4 自动 commit + push 流程

```
skillstore.Save(name, files)
  ├── 加文件锁
  ├── 写 tmp dir
  ├── atomic rename
  ├── 释放锁
  └── skillversion.AutoCommitAndPush(group, name, files)
        ├── go-git PlainOpen(skills/)
        ├── Worktree.Status() 判断是否真有改动(防 rename 失败)
        ├── w.AddGlob("<group>/<name>/**") 或 w.AddWithOptions(All: true)
        ├── w.Commit(msg, author)
        │     msg 格式: "skill(store): update <group>/<name>" / "create / delete"
        ├── 异步 Push(走 goroutine + retry queue)
        └── return  // 不阻塞 store.Save 的响应
```

**并发保证:**
- `skillstore.Save` 的 per-skill 文件锁继续保留,保证单 skill 写串行
- 新增 `skillversion.Repo.mu`(全局锁),保证 git 操作不会跟 store.Save 同时跑
- Push 失败进重试队列(本地内存 + 进程退出前 flush 到 `~/.skill-box/git_push_queue.json`),启动时扫一次重试

**commit author 设定:**
- 优先读 `~/.skill-box/config.yaml` 的 `git.user_name` / `git.user_email`
- 否则 fallback 到 `os.Getenv("GIT_AUTHOR_NAME")` / `os.Getenv("GIT_AUTHOR_EMAIL")`
- 否则用占位 `"skill-box" <skill-box@local>`(只在未配 remote 时用,避免 commit 失败)

### 2.5 数据迁移(从 DB tag 到 git commit)

**目标**:`skill_tags` / `skill_file_snapshots` 表的数据保留(只读),但新代码不再读写。

**迁移步骤**(一次性,放在新方案上线的 commit):
1. 新增 `skillversion.InitIfNotExists(skills/)` 函数,首次启动时检测:
   - `skills/.git/` 不存在 → `PlainInit(skills/, false)` + 写 `.gitignore` + `git commit --allow-empty` 初始化
   - 已存在 → skip
2. **存量技能的首次 commit**:遍历 `skills/<group>/<name>` 全部目录,写一个 `chore(skills): initial commit` 的 commit 把所有现有内容入库(2026-07-17 这个时间点之前的状态)。
3. **保留 DB 表**:`AutoMigrate` 仍然注册 `skill_tags` / `skill_file_snapshots`,但前端不再触发 CreateTag 等接口(UI 整个换)。表数据保留供日后排查。

### 2.6 新增 HTTP API

`internal/gapi/controller/skillbox/cskillversion/`(新增包):

| Path | Method | 用途 |
|---|---|---|
| `/api/skillbox/git/config` | GET | 读远端配置(URL/branch/token 是否已配,不返回 token) |
| `/api/skillbox/git/config` | POST | 写远端配置 + 写 token 文件 |
| `/api/skillbox/git/status` | GET | 当前 Repo 的 Status / Last commit / Ahead-Behind |
| `/api/skillbox/git/log` | GET | commit 历史(支持 `?limit=20&offset=0&path=<group>/<name>`) |
| `/api/skillbox/git/log/<hash>` | GET | 单 commit 详情(Author / Message / Files changed) |
| `/api/skillbox/git/diff` | GET | `?from=<hash>&to=<hash>` 或 `?from=<hash>&to=HEAD` |
| `/api/skillbox/git/checkout` | POST | 把工作区 reset 到某 commit(对应"切到旧版本") |
| `/api/skillbox/git/push` | POST | 手动 push(给"重试 push 失败"用) |
| `/api/skillbox/git/pull` | POST | 手动 pull(从远端拉) |
| `/api/skillbox/git/init` | POST | 初始化仓库(给"未配 git 但想启用"的入口) |
| `/api/skillbox/git/discard` | POST | 丢弃工作区未提交改动(对应"撤销编辑") |

**废弃 API**:
- `/api/skillbox/skills/tags/create` / `list` / `delete` / `diff` / `rollback` —— 整个 ctag 包废弃

### 2.7 前端改造

#### 2.7.1 SkillsView.vue 改动

**删除**:
- `import { createTag, listTags, deleteTag, diffTag, rollbackTag } from '@/api/skillbox/tags'` (`SkillsView.vue:20`)
- `tagOpen` / `openTagDialog` / `loadTagList` / `doCreateTag` / `doDeleteTag` / `doDiff` / `doRollback` / `currentTagList` (`SkillsView.vue:78,618-701`)
- 详情区的 tag chip (`SkillsView.vue:502-505`)
- 整个 `<Modal v-model="tagOpen">` 模板块 (`SkillsView.vue:1939-2022`)
- i18n keys: `skills.tag.*` 一整组

**新增**:
- 详情区顶栏 "Git: a↑ b↓" 状态徽章(`SkillScopePanel` 类似位置)
- 弹窗 `<Modal v-model="versionOpen">`: commit 时间线 + diff + checkout + push 按钮
- 快捷入口: Settings 里加 Git 配置 Tab

#### 2.7.2 SkillScopePanel.vue 改动

**新增 section** "Git 同步状态":
- Remote URL(可点击测试连接)
- Branch
- 当前 HEAD commit 短码 + message 头
- "Ahead N · Behind M" 状态
- Push 按钮(失败时变红 + tooltip 显原因)
- Pull 按钮
- "查看历史" 按钮 → 打开 versionOpen 弹窗

#### 2.7.3 SkillFileInlinePanel.vue 改动

**顶栏 badge**:
- 旧的 `skillVersion` computed(读 frontmatter 的 version)
- 改成:从 commit log 取 HEAD 短码(7 位 hash)+ tag 列表

#### 2.7.4 新 API 客户端

`frontend/src/api/skillbox/git.js`(新增,~60 行):

```js
import { http } from '@/core/utils/requests'

export const getGitConfig = () => http.get('/api/skillbox/git/config')
export const saveGitConfig = (cfg) => http.post('/api/skillbox/git/config', cfg)
export const getGitStatus = () => http.get('/api/skillbox/git/status')
export const getGitLog = (params) => http.get('/api/skillbox/git/log', params)
export const getGitCommit = (hash) => http.get(`/api/skillbox/git/log/${hash}`)
export const getGitDiff = (params) => http.get('/api/skillbox/git/diff', params)
export const checkoutGit = (hash) => http.post('/api/skillbox/git/checkout', { hash })
export const pushGit = () => http.post('/api/skillbox/git/push')
export const pullGit = () => http.post('/api/skillbox/git/pull')
export const initGit = () => http.post('/api/skillbox/git/init')
export const discardGit = () => http.post('/api/skillbox/git/discard')
```

`frontend/src/api/skillbox/tags.js`:删除整个文件。

---

## 3. 安全与边界

### 3.1 Token 存储
- 写入路径:`~/.skill-box/.git_token`
- 文件权限:`os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)`
- 读取路径:仅 `skillversion.gitconfig.LoadAuth` 内部用,不出 HTTP response
- HTTP response 中 token 字段永远不返回,只回 `has_token: true/false`

### 3.2 远端 URL 校验
- 必须 `https://` 开头(显式拒绝 `http://` 防中间人)
- 域名必须在白名单内:`github.com` / `gitlab.com` / `gitee.com` / 用户自定义(Gitea 等)
- 不支持 SSH URL(本期不做)

### 3.3 Push 失败兜底
- Push 失败不抛错给用户(仅日志)
- 写 `~/.skill-box/git_push_queue.json`:`[{hash, msg, attempted_at}]`
- 启动时 + 任意 success push 后扫一次重试
- 上限重试 5 次,失败后从队列移除(用户手动重试入口是 SkillsView 的 "Push" 按钮)

### 3.4 工作区未提交改动保护
- 任何 `checkout` / `reset --hard` 前必先 `Status()` 判断
- 如果有未提交改动,前端弹确认框("会丢失本地未保存的修改")
- 后端兜底:`Status().IsClean() == false` 时直接 409 拒绝

### 3.5 commit hash 不入库
- 不在 DB 加新表"记录 git commit hash",完全靠 `git log` 实时拉
- 唯一例外:`.skillbox-meta/last_commit_<group>_<name>.txt` 写文件,给"快速定位某 skill 当前 HEAD"用(可选,不强制)

---

## 4. 实施计划

### 4.1 阶段一:基础设施(本次提交)
1. `internal/skillversion/repo.go` —— go-git 封装
2. `internal/skillversion/gitconfig/config.go` —— 配置 + token
3. `internal/skillversion/init.go` —— PlainInit + .gitignore
4. `internal/skillstore/store.go` 末尾加 hook(默认 noop,后续阶段接入)
5. `api-server/internal/gapi/controller/skillbox/cskillversion/` —— 11 个 HTTP API
6. 前端 `git.js` 客户端
7. 前端 `SkillScopePanel.vue` 加 Git section
8. 前端 `SettingsView`(如有) 加 Git config tab
9. **保留**所有现有 API/前端,旧 ctag 暂时不动,新旧并存

### 4.2 阶段二:前端替换(独立 PR)
1. `SkillsView.vue` 删除整个 tag 弹窗
2. 新增 `<Modal v-model="versionOpen">` version 弹窗
3. `tags.js` 删除
4. `ctag` 后端路由 + controller 注释 TODO:下一阶段删除
5. `skillaudit` 包 注释 TODO

### 4.3 阶段三:DB 清理(独立 PR,需用户确认)
1. `entity.SkillTag.GenConfig().AutoMigrate` 标记 false
2. `entity.SkillFileSnapshot.GenConfig().AutoMigrate` 标记 false
3. 前端灰度开关:GUI 弹"是否迁移 DB tag → git tag"功能(可选)
4. 等所有用户迁完,真正 drop table

---

## 5. 待决项

- [ ] 是否支持 sub-module? (答案:否,单仓足够)
- [ ] .gitignore 内容? (临时: `.skillbox-meta/`, `*.swp`, `.DS_Store`)
- [ ] push queue 的存储格式? (JSON 单文件,够用)
- [ ] commit message 是否 i18n? (否,用英文 + 路径后缀,grep 友好)
- [ ] 是否支持 git tag(轻量 / 注释)? (阶段四考虑,本期不做)
- [ ] 三个 init / config / status 等 API 的 permission 名怎么命名? (`skillbox.git.read` / `skillbox.git.write`)

---

## 6. 测试报告

> 本任务当前为方案设计阶段,未进入自测。下一轮实施阶段开始填写。

## 7. 总结

> 待阶段一完成后填。

## 8. 改动的文件

### 8.1 新增
> 本轮仅写 task 文档,未改代码。

### 8.2 修改
> 本轮仅写 task 文档,未改代码。

### 8.3 删除
> 本轮仅写 task 文档,未改代码。

## 9. 工具与用途

### 9.1 MCP 工具
- `MCP MiniMax::web_search` — 查 go-git PlainClone / Pull / Push 错误特征(第二轮)
- (WebFetch 通过 Skill 工具实现,不计入 MCP)

### 9.2 Skill
- 无

### 9.3 CLI
- `Bash ls` — 列项目目录结构(第一轮)
- `Bash mkdir -p` — 建 task 月份目录(第三轮)

---

## N.M 对话轮次记录

> 时间正序追加,每轮一段。

### 1.1 对话轮次 (2026-07-17 第一轮)

> 用户原话:"前项目我想使用 go-git 的一个 Git 库,主要是用于版本管理,包括同步技能等多项功能。我想把当前的方案全部替换掉,全部换成基于文件的、基于 go-git 的版本管理,而不是基于数据库的版本管理。同时也可以把整个项目同步到 GitHub 等远端仓库,这样方便我们去同步我们的技能。~/.skill-box/skills  把这个目录作为一整个 git 仓库。"

- **本轮做了:**
  - 读完项目 CLAUDE.md + 项目架构文档
  - 启动 Explore agent 拉基线报告(技能版本管理现状扫描)
  - 启动 WebSearch / WebFetch 拉 go-git v5 API 文档
  - 通过 AskUserQuestion 收齐 4 个关键架构决策
  - 写 task 文档(本文件)覆盖需求/架构/API/前端/迁移
- **本轮决定:**
  - 仓库粒度 = 单仓(monorepo)
  - 同步策略 = 全自动 commit + push(push 异步 + 重试队列)
  - 远端认证 = HTTPS + PAT,token 文件 0600
  - 过渡策略 = 完全替换,旧 ctag API/前端/包全部废弃,DB 表保留只读
- **本轮待办:**
  - 等用户审阅方案 → 确认后进入阶段一实施
  - 阶段一: skillversion 包 + 11 个 API + 前端 git.js + SkillScopePanel Git section
- **本轮工具:**
  - `Bash ls` — 列 docs/agent 子目录
  - `Bash mkdir -p` — 建 task 月份目录
  - `Agent Explore` — 拉代码侧基线
  - `MCP MiniMax::web_search` — 查 go-git 错误特征
  - `WebFetch` pkg.go.dev — go-git v5 API
  - `WebFetch` COMPATIBILITY.md — go-git 能力边界
  - `AskUserQuestion` — 4 个架构决策
- **状态更新:** 任务列表勾选 task #1 / #2 完成,task #3 进行中