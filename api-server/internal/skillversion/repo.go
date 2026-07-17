// Package skillversion 提供基于 go-git 的技能仓库版本管理(2026-07-17 增)。
//
// 目标:
//   - 把 ~/.skill-box/skills/ 整个目录做成一个 git 仓库(monorepo 模式)。
//   - skillstore.Save / Delete 落盘成功后,自动 git add + commit + 异步 push。
//   - 提供 init / log / diff / checkout / reset / push / pull / status 等 HTTP API。
//
// 不做:
//   - SSH / LFS / sub-module / 三方合并(go-git 不支持或本期不做)。
//   - DB 表存储 commit 历史(全部靠 git log 实时拉)。
//
// 调用层次:
//
//	skillstore.Save / Delete
//	  └─→ skillversion.AutoCommitAndPush(msg, paths)   ← store 末尾 hook
//	         └─→ skillversion.Repo.Commit / Push        ← 走 go-git
//
//	HTTP API(cskillversion)
//	  └─→ skillversion.Repo.*                          ← 直接调,不走 hook
package skillversion

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"ginp-api/configs"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// 2026-07-17:错误集,前端可以 errors.Is 判断。
var (
	// ErrRepoNotInitialized 仓库尚未 PlainInit,需要先调 InitIfNotExists。
	ErrRepoNotInitialized = errors.New("skillversion: repo not initialized")

	// ErrRemoteNotConfigured 没有配置远端 URL,push/pull 无意义。
	ErrRemoteNotConfigured = errors.New("skillversion: remote not configured")

	// ErrWorkingTreeDirty 工作区有未提交改动,checkout / reset --hard 必须先确认。
	ErrWorkingTreeDirty = errors.New("skillversion: working tree has uncommitted changes")

	// ErrPushFailed push 失败(网络 / 认证 / 远端拒绝等),原始 err 在 message 里。
	ErrPushFailed = errors.New("skillversion: push failed")

	// ErrInvalidURL URL 不合法(非 https / 域名不在白名单)。
	ErrInvalidURL = errors.New("skillversion: invalid remote url")

	// 2026-07-18 增:diff 在当前进程环境不可用(wails webview sandbox 拦截
	// exec.Command("git") 或 go-git IO 卡死)。前端 GitDiff handler 检测
	// 到这个错误时返 stub hint,让用户用 CLI 查看。
	ErrSandboxUnavailable = errors.New("skillversion: git diff unavailable in current sandbox")
)

// Repo 封装单个 git 仓库(目前永远指 ~/.skill-box/skills)。
//
// 进程内全局只允许一个 Repo 实例,通过 Default() 拿;mu 串行所有 git 操作,
// 避免与 skillstore.Save 的 per-skill flock 并发打架。
type Repo struct {
	path string
	mu   sync.Mutex
}

var (
	defaultOnce sync.Once
	defaultRepo *Repo
	defaultErr  error
)

// Default 拿全局单例 Repo。
//
// 2026-07-17:Repo 持有 mutex,go-git 同一进程多 goroutine 访问
// Worktree 不安全(go-git 5.x 文档未保证),所以全局只允许一个。
func Default() (*Repo, error) {
	defaultOnce.Do(func() {
		root, err := resolveRoot()
		if err != nil {
			defaultErr = err
			return
		}
		defaultRepo = &Repo{path: root}
	})
	return defaultRepo, defaultErr
}

// Root 返回仓库绝对路径(等于 skillstore 的 StoreRoot,可能 EvalSymlinks 过)。
func (r *Repo) Root() string { return r.path }

// resolveRoot 解析仓库路径:优先 configs.Skillbox.StoreRoot,否则 ~/.skill-box/skills。
//
// 2026-07-17:跟 skillstore.New() 内部逻辑保持一致 — 改造完后两者一定指向同一目录。
func resolveRoot() (string, error) {
	if root := strings.TrimSpace(configs.Skillbox.StoreRoot); root != "" {
		return root, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("skillversion: cannot resolve home: %w", err)
	}
	return filepath.Join(home, ".skill-box", "skills"), nil
}

// IsInitialized 判断 .git/ 是否存在。
func (r *Repo) IsInitialized() bool {
	_, err := os.Stat(filepath.Join(r.path, ".git"))
	return err == nil
}

// open 内部 helper:PlainOpen 仓库;未 init 时返 ErrRepoNotInitialized。
func (r *Repo) open() (*git.Repository, error) {
	if !r.IsInitialized() {
		return nil, ErrRepoNotInitialized
	}
	return git.PlainOpen(r.path)
}

// PlainInit 初始化仓库(已存在时跳过,但仍确保 .gitignore 写入)。
//
// 2026-07-17:不强制 isBare — 用户需要日常查看工作区(前端展示 commit、文件浏览),
// bare 仓库 Worktree() 失败。
func (r *Repo) InitIfNotExists() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := os.MkdirAll(r.path, 0o755); err != nil {
		return fmt.Errorf("skillversion: mkdir %s: %w", r.path, err)
	}

	// 写 .gitignore(覆盖式)— 用户每次 init 都保证有这层 ignore,
	// 即使后续手动删了也会恢复。
	if err := writeGitignore(r.path); err != nil {
		return err
	}

	if r.IsInitialized() {
		// 2026-07-17 增:已 init 但 HEAD 损坏(PlainInit 后 wt.Commit 失败导致
		// HEAD 指零 hash / 仓库里没 commit object)的自愈。
		// 这种情况 IsInitialized() 返 true,但 Log/Head() 都会"object not found"。
		// 这里检查 HEAD 是否指向有效 commit,没有就补一个 empty commit。
		if err := repairHeadIfBroken(); err != nil {
			return fmt.Errorf("skillversion: repair HEAD: %w", err)
		}
		return nil
	}

	repo, err := git.PlainInit(r.path, false)
	if err != nil {
		return fmt.Errorf("skillversion: PlainInit: %w", err)
	}

	// 2026-07-17:默认分支 main(go-git 5.7+ 支持,旧版会拿 master)。
	_ = repo.CreateBranch(&config.Branch{Name: "main"})
	// 切到 main(如果 PlainInit 落到 master)。HEAD 引用走 plumbing。
	_ = repo.Storer.SetReference(plumbing.NewHashReference(plumbing.HEAD, plumbing.NewHash("")))

	// 2026-07-17:空 init 后,做一次空 commit 让 HEAD 落到 main,后续 Log 才不会空指针。
	wt, werr := repo.Worktree()
	if werr != nil {
		return fmt.Errorf("skillversion: Worktree: %w", werr)
	}
	author := resolveAuthor()
	_, cerr := wt.Commit("chore(skills): initialize empty repository", &git.CommitOptions{
		Author:            author,
		AllowEmptyCommits: true,
	})
	if cerr != nil {
		// 2026-07-17 改:不再静默 — wt.Commit 在仓库没 commit base 时会
		// "object not found" 失败,这时退到 repairHeadIfBroken 走手工
		// 建 commit object 绕过(go-git 5.x worktree commit 要求 base tree
		// 关联到某个 commit,空仓库的 base 缺失就挂)。
		if rerr := repairHeadIfBroken(); rerr != nil {
			return fmt.Errorf("skillversion: commit + repair both failed: commit=%v repair=%v", cerr, rerr)
		}
	}
	return nil
}

// repairHeadIfBroken 自愈损坏的 HEAD — 当仓库 .git 存在但 HEAD 指零 hash
// 或 refs/heads/* 全空时,手工建一个 root commit object 并把 main / HEAD
// 指过去。
//
// 2026-07-17 背景:BootstrapInit 异步跑时,如果进程被强制终止或 IO 失败,
// 可能留下"PlainInit 完成 + 没有 commit + HEAD 零 hash"的中间状态。这时
// 任何 Log / Head() 都会"object not found" / panic。repair 通过直接调
// object.Commit.Encode + Storer.SetEncodedObject 绕开 Worktree.Commit
// 对 base tree 的依赖,稳定建一个 root commit。
func repairHeadIfBroken() error {
	root := defaultRepo.path
	repo, err := git.PlainOpen(root)
	if err != nil {
		return err
	}
	head, herr := repo.Head()
	if herr == nil && !head.Hash().IsZero() {
		// HEAD 已指向有效 commit,不需要修
		return nil
	}
	// 空 tree 的 hash 是固定常量(go-git / git 内部都用)
	const emptyTreeHash = "4b825dc642cb6eb9a060e54bf8d69288fbee4904"
	author := resolveAuthor()
	commit := &object.Commit{
		Author:       *author,
		Committer:    *author,
		Message:      "chore(skills): bootstrap empty commit (repair)\n",
		TreeHash:     plumbing.NewHash(emptyTreeHash),
		ParentHashes: nil,
	}
	obj := repo.Storer.NewEncodedObject()
	if err := commit.Encode(obj); err != nil {
		return fmt.Errorf("encode commit: %w", err)
	}
	hash, err := repo.Storer.SetEncodedObject(obj)
	if err != nil {
		return fmt.Errorf("store commit: %w", err)
	}
	if err := repo.Storer.SetReference(plumbing.NewHashReference("refs/heads/main", hash)); err != nil {
		return fmt.Errorf("set refs/heads/main: %w", err)
	}
	if err := repo.Storer.SetReference(plumbing.NewSymbolicReference("HEAD", "refs/heads/main")); err != nil {
		return fmt.Errorf("set HEAD: %w", err)
	}
	return nil
}

// writeGitignore 写 .skillbox-meta/ 与 .DS_Store 等临时文件到 .gitignore。
func writeGitignore(root string) error {
	const newContent = `# skill-box local-only metadata, must not sync to remote
.skillbox-meta/
# OS junk
.DS_Store
Thumbs.db
# Editor swap
*.swp
*.swo
*~
# Go test binaries (用户偶尔在 ~/.skill-box/skills/ 下跑 go test 的兜底)
*.test
`
	p := filepath.Join(root, ".gitignore")
	if existing, err := os.ReadFile(p); err == nil {
		// 已存在 → 不覆盖,避免冲掉用户手动加的规则
		if strings.Contains(string(existing), ".skillbox-meta/") {
			return nil
		}
		// 追加(用户文件优先)
		merged := string(existing) + "\n# --- skillversion auto-append ---\n" + newContent
		return os.WriteFile(p, []byte(merged), 0o644)
	}
	return os.WriteFile(p, []byte(newContent), 0o644)
}

// resolveAuthor 拿 commit 作者,优先级:
//  1. configs.Skillbox.Git.UserName / UserEmail
//  2. env GIT_AUTHOR_NAME / GIT_AUTHOR_EMAIL
//  3. 占位 "skill-box" / "skill-box@local"
func resolveAuthor() *object.Signature {
	name := strings.TrimSpace(configs.Skillbox.Git.UserName)
	if name == "" {
		name = strings.TrimSpace(os.Getenv("GIT_AUTHOR_NAME"))
	}
	if name == "" {
		name = "skill-box"
	}
	email := strings.TrimSpace(configs.Skillbox.Git.UserEmail)
	if email == "" {
		email = strings.TrimSpace(os.Getenv("GIT_AUTHOR_EMAIL"))
	}
	if email == "" {
		email = "skill-box@local"
	}
	return &object.Signature{
		Name:  name,
		Email: email,
		When:  time.Now(),
	}
}

// CommitEntry 单条 commit 摘要(给 HTTP API 用)。
type CommitEntry struct {
	Hash    string    `json:"hash"`
	Short   string    `json:"short"`
	Author  string    `json:"author"`
	Email   string    `json:"email"`
	Message string    `json:"message"`
	When    time.Time `json:"when"`
	// 2026-07-17 增:parent 哈希(空 = root commit)— 前端 diff 用真实 parent
	// 避免发 "<hash>^" 让 go-git ResolveRevision 卡死。
	ParentHash string `json:"parent_hash,omitempty"`
	// 2026-07-17:涉及的文件路径列表(相对 repo root)。空数组 = 不展示。
	Files []string `json:"files,omitempty"`
}

// Status 仓库状态摘要,前端 Settings Tab 用。
type Status struct {
	Initialized   bool   `json:"initialized"`
	Branch        string `json:"branch,omitempty"`
	RemoteURL     string `json:"remote_url,omitempty"`
	RemoteBranch  string `json:"remote_branch,omitempty"`
	HeadHash      string `json:"head_hash,omitempty"`
	HeadShort     string `json:"head_short,omitempty"`
	HeadMessage   string `json:"head_message,omitempty"`
	WorkingClean  bool   `json:"working_clean"`
	Ahead         int    `json:"ahead,omitempty"`
	Behind        int    `json:"behind,omitempty"`
	HasToken      bool   `json:"has_token"`
	PendingPush   int    `json:"pending_push"`
	LastPushError string `json:"last_push_error,omitempty"`
}