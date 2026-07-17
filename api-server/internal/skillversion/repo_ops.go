package skillversion

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"

	"ginp-api/pkg/logger"
)

// CommitInput 自动 commit 的入参。
//
// 2026-07-17:skillstore.Save / Delete / Move 末尾调 AutoCommitAndPush,传 (msg, paths)。
// paths 是相对 repo root 的文件/目录列表(传 nil/空 = add all modified)。
type CommitInput struct {
	Message string
	// Paths 相对 repo root 的文件/目录列表(用正斜杠);nil = add all。
	// 2026-07-17 决策:Paths 仅作 commit message 注释,不参与 add — 始终走 Add All 简化逻辑。
	Paths []string
}

// AutoCommitAndPush 同步 commit + 异步 push。
//
// 2026-07-17 流程:
//  1. mu 串行(避免与 HTTP API 并发)
//  2. InitIfNotExists(首次启动兜底)
//  3. Worktree.AddWithOptions(All: true)
//  4. Worktree.Commit(msg, author)
//  5. enqueuePush(hash, msg) — 不阻塞
//
// 失败兜底:任何 git 错误都只写 logger.Error,不抛(因为调用方是 store.Save,
// 业务上写盘已经成功,版本管理失败不能反向回滚数据)。
// AutoCommitAndPush 同步 commit + 异步 push。
//
// 2026-07-17 流程:
//  1. mu 串行(避免与 HTTP API 并发)
//  2. InitIfNotExists(首次启动兜底)
//  3. Worktree.AddWithOptions(All: true)
//  4. Worktree.Commit(msg, author)
//  5. enqueuePush(hash, msg) — 不阻塞
//
// 失败兜底:任何 git 错误都只写 logger.Error,不抛(因为调用方是 store.Save,
// 业务上写盘已经成功,版本管理失败不能反向回滚数据)。
//
// 2026-07-18 大改:之前走 go-git Worktree.Commit 在 wails v3 alpha.60 webview
// 子进程里死锁(跟 Diff() 同根因 — go-git IO 在 Chromium 多进程架构下调度异常)。
// 跟 Diff() 一样改走系统 git CLI(exec.CommandContext + 3s 超时):
//   - `git -C <path> add -A` → `git -C <path> commit -m <msg> --author=<a>`
//   - 拿 stdout 解析 commit hash(`git rev-parse HEAD`)
//   - push 仍走 go-git(异步,可容忍失败)
//
// 新策略是"先 git CLI 跑通、go-git 只负责 read-only 跟 push",
// 这样 store.Save 链路的 commit 不再会被 webview 子进程 IO 调度阻塞。
func (r *Repo) AutoCommitAndPush(in CommitInput) (plumbing.Hash, error) {
	logger.Warn("skillversion: AutoCommitAndPush ENTER: msg=%q paths=%v", in.Message, in.Paths)
	r.mu.Lock()
	logger.Warn("skillversion: AutoCommitAndPush mu acquired")
	defer r.mu.Unlock()

	if err := r.InitIfNotExists(); err != nil {
		logger.Warn("skillversion: AutoCommitAndPush InitIfNotExists: %v", err)
		return plumbing.ZeroHash, err
	}
	logger.Warn("skillversion: AutoCommitAndPush after init, calling git CLI commit")

	msg := strings.TrimSpace(in.Message)
	if len(in.Paths) > 0 {
		// paths 拼到 commit message 末尾方便 grep
		msg = msg + "\n\nfiles: " + strings.Join(in.Paths, ", ")
	}

	author := resolveAuthor()
	hash, err := r.commitViaCLI(msg, author)
	if err != nil {
		logger.Warn("skillversion: AutoCommitAndPush CLI commit failed: %v", err)
		return plumbing.ZeroHash, err
	}
	logger.Warn("skillversion: AutoCommitAndPush CLI commit OK hash=%v", hash.String())

	// 异步 push,失败入重试队列
	r.enqueuePush(hash, msg)
	return hash, nil
}

// commitViaCLI 走系统 git CLI 完成 add + commit + 读 hash。
// 跟 Diff() 同一个根因修复:避免 go-git 在 wails webview 子进程里 IO 死锁。
func (r *Repo) commitViaCLI(msg string, author *object.Signature) (plumbing.Hash, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// git add -A
	addCmd := exec.CommandContext(ctx, "git", "-C", r.path, "add", "-A")
	if out, err := addCmd.CombinedOutput(); err != nil {
		return plumbing.ZeroHash, fmt.Errorf("skillversion: git add failed: %v: %s", err, string(out))
	}

	// git commit -m <msg> --author=<author>
	authorStr := fmt.Sprintf("%s <%s>", author.Name, author.Email)
	commitCmd := exec.CommandContext(ctx, "git", "-C", r.path,
		"commit", "-m", msg, "--author="+authorStr)
	if out, err := commitCmd.CombinedOutput(); err != nil {
		// 空 commit(no changes)不算错,fallback 读 HEAD
		outStr := string(out)
		if strings.Contains(outStr, "nothing to commit") || strings.Contains(outStr, "no changes added") {
			hashOut, herr := r.headHashViaCLI(ctx)
			if herr != nil {
				return plumbing.ZeroHash, herr
			}
			return plumbing.NewHash(hashOut), nil
		}
		return plumbing.ZeroHash, fmt.Errorf("skillversion: git commit failed: %v: %s", err, outStr)
	}

	return r.commitAndReadHashViaCLI(ctx)
}

// commitAndReadHashViaCLI 走 `git rev-parse HEAD` 拿当前 commit hash 字符串
// 转 plumbing.Hash。
func (r *Repo) commitAndReadHashViaCLI(ctx context.Context) (plumbing.Hash, error) {
	hashOut, err := r.headHashViaCLI(ctx)
	if err != nil {
		return plumbing.ZeroHash, err
	}
	return plumbing.NewHash(hashOut), nil
}

// headHashViaCLI 走 `git rev-parse HEAD` 拿当前 commit hash。
func (r *Repo) headHashViaCLI(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", r.path, "rev-parse", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("skillversion: git rev-parse failed: %v", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// enqueuePush 异步 push;不阻塞调用方。
//
// 2026-07-17:不走 goroutine 池(简单方案,Push 频率低)。
// 失败错误存到 lastPushErr,前端 Status 接口读取。
func (r *Repo) enqueuePush(hash plumbing.Hash, msg string) {
	go func() {
		if err := r.pushLocked(); err != nil {
			r.recordPushError(err)
			pushQueue.Add(hash.String(), msg, err.Error())
			return
		}
		r.recordPushError(nil)
		pushQueue.Remove(hash.String())
	}()
}

// pushLocked 内部:假定 caller 已持有 r.mu。
func (r *Repo) pushLocked() error {
	auth, err := loadAuthFromConfig()
	if err != nil {
		return err
	}
	repo, err := r.open()
	if err != nil {
		return err
	}
	return repo.Push(&git.PushOptions{
		RemoteName: "origin",
		Auth:       auth,
		Force:      false,
	})
}

// loadAuthFromConfig 从 gitconfig 拿远端 URL + token 文件,组装 BasicAuth。
func loadAuthFromConfig() (*githttp.BasicAuth, error) {
	cfg := getGitConfig()
	if cfg.RemoteURL == "" {
		return nil, ErrRemoteNotConfigured
	}
	if cfg.TokenFile == "" {
		return nil, fmt.Errorf("skillversion: token_file not configured")
	}
	token, err := readTokenFile(cfg.TokenFile)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(token) == "" {
		return nil, fmt.Errorf("skillversion: token file is empty")
	}
	return &githttp.BasicAuth{
		Username: "skill-box", // GitHub PAT 模式 username 任意,只校验 token
		Password: token,
	}, nil
}

// readTokenFile 读 token 文件(单行 trim)。
func readTokenFile(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("skillversion: read token file %s: %w", path, err)
	}
	return strings.TrimSpace(string(b)), nil
}

// recordPushError / LastPushError 跨 goroutine 同步最后一次 push 错误。
var lastPushErrMu sync.Mutex
var lastPushErr string

func (r *Repo) recordPushError(err error) {
	lastPushErrMu.Lock()
	if err == nil {
		lastPushErr = ""
	} else {
		lastPushErr = err.Error()
	}
	lastPushErrMu.Unlock()
}

// LastPushError 读最后一次 push 错误(空 = 上次成功)。
func LastPushError() string {
	lastPushErrMu.Lock()
	defer lastPushErrMu.Unlock()
	return lastPushErr
}

// getGitConfig 拿远端配置(走 gitconfig 包,避免循环依赖)。
func getGitConfig() gitconfigSkillConfig {
	return gitconfigGet()
}

// gitconfigSkillConfig / gitconfigGet 是包级桥接,实际定义见 gitconfig_bridge.go。
type gitconfigSkillConfig = struct {
	RemoteURL string
	Branch    string
	TokenFile string
	UserName  string
	UserEmail string
}

func gitconfigGet() gitconfigSkillConfig {
	c := gitconfigSnapshot()
	return gitconfigSkillConfig{
		RemoteURL: c.RemoteURL,
		Branch:    c.Branch,
		TokenFile: c.TokenFile,
		UserName:  c.UserName,
		UserEmail: c.UserEmail,
	}
}

// Status 读当前仓库状态(给 HTTP API)。
func (r *Repo) Status() (Status, error) {
	cfg := getGitConfig()
	s := Status{
		Initialized: r.IsInitialized(),
		RemoteURL:   cfg.RemoteURL,
		RemoteBranch: cfg.Branch,
		HasToken:    hasTokenFile(),
		PendingPush: pushQueue.Len(),
	}
	if last := LastPushError(); last != "" {
		s.LastPushError = last
	}
	if !s.Initialized {
		return s, nil
	}
	repo, err := r.open()
	if err != nil {
		return s, err
	}
	head, err := repo.Head()
	if err == nil {
		hash := head.Hash()
		// 2026-07-17 改:go-git PlainInit 后 HEAD 可能指向 zero hash(还没
		// 任何 commit),这时不应该显示 "0000000",前端 badge 改成空 + 状态
		// 文案 "no commits yet"。分支名也只在有 commit 时才有意义。
		if !hash.IsZero() {
			s.HeadHash = hash.String()
			s.HeadShort = hash.String()[:7]
			if name := head.Name(); name != plumbing.ReferenceName("") {
				s.Branch = name.Short()
			}
			if c, cerr := repo.CommitObject(hash); cerr == nil {
				msg := strings.SplitN(c.Message, "\n", 2)[0]
				if len(msg) > 120 {
					msg = msg[:120] + "…"
				}
				s.HeadMessage = msg
			}
		}
	}
	if remote, rerr := repo.Remote("origin"); rerr == nil {
		if rc := remote.Config(); rc != nil && len(rc.URLs) > 0 && s.RemoteURL == "" {
			s.RemoteURL = rc.URLs[0]
		}
	}
	if wt, werr := repo.Worktree(); werr == nil {
		if st, serr := wt.Status(); serr == nil {
			s.WorkingClean = st.IsClean()
		}
	}
	return s, nil
}

// Log 读最近 N 条 commit 历史(默认 50,上限 500)。
//
// 2026-07-17 增:可选 pathPrefix 参数过滤 — 非空时只返回涉及该路径前缀的
// commit(用 go-git LogOptions.FileName + 跳过未涉及文件,O(N) walk 树)。
// 这是 per-skill 修改历史的实现核心:传 "<group>/<name>/" 即可只显示该 skill 的 commit。
func (r *Repo) Log(limit int, pathPrefix string) (out []CommitEntry, err error) {
	// 2026-07-17 加 recover 兜底 — go-git 内部 commitFiles 走 parent.Patch
	// 在某些 commit shape 下会 panic(runtime error: slice bounds out of range),
	// 这里兜住返空,让接口返 200 而不是 500。
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "[skillversion] Log panic: %v\n%s\n", r, debug.Stack())
			err = fmt.Errorf("skillversion: Log panic: %v", r)
			out = nil
		}
	}()
	if limit <= 0 || limit > 500 {
		limit = 50
	}
	repo, err := r.open()
	if err != nil {
		return nil, err
	}
	head, err := repo.Head()
	if err != nil {
		return nil, err
	}
	// 2026-07-17 改:per-skill 过滤走 go-git 原生 PathFilter(内部走
	// commit-tree diff 索引),不再在 ForEach 里手工 commitFiles —
	// 之前手工 commitFiles 走 parent.Patch 在某些 commit shape 下会
	// panic(slice bounds out of range),改成 PathFilter 让 go-git 自己处理。
	// 2026-07-18 改:把 prefix 提到 if 分支外面,这样 else 分支也是空字符串,
	// commitFiles 调用一致地拿到 prefix(空 = 不过滤)。
	var cIter object.CommitIter
	prefix := pathPrefix
	if pathPrefix != "" {
		cIter, err = repo.Log(&git.LogOptions{
			From:       head.Hash(),
			PathFilter: func(p string) bool { return strings.HasPrefix(p, prefix) },
		})
	} else {
		cIter, err = repo.Log(&git.LogOptions{From: head.Hash()})
	}
	if err != nil {
		return nil, err
	}
	count := 0
	_ = cIter.ForEach(func(c *object.Commit) error {
		if count >= limit {
			return errStop
		}
		// 2026-07-17 改:per-path 过滤已由 git.LogOptions.PathFilter 接管,
		// 这里不再手工 commitFiles(可能 panic)。commitFiles 仅在
		// 给 entry.Files 字段用,且独立包 recover 兜底。
		entry := CommitEntry{
			Hash:    c.Hash.String(),
			Short:   c.Hash.String()[:7],
			Author:  c.Author.Name,
			Email:   c.Author.Email,
			Message: strings.SplitN(c.Message, "\n", 2)[0],
			When:    c.Author.When,
		}
		// 2026-07-17 增:第一 parent hash(供前端 diff 用,避免发 "<hash>^"
		// 让 go-git ResolveRevision 卡 15s)。root commit 留空。
		if c.NumParents() > 0 {
			entry.ParentHash = c.ParentHashes[0].String()
		}
		// 2026-07-17 改:commitFiles 用 recover + error 双重兜底 —
		// go-git c.Stats() / Tree() 在合并 / squash / 孤儿 commit
		// 上会返 "object not found" 错误(不是 panic),这里捕获到
		// 错误时 entry.Files 留空但 entry 仍然 append,不让单条
		// commit 阻断整次 Log。前端需要的 hash/msg/author/when 都
		// 在 entry 里,Files 是锦上添花。
		func() {
			defer func() {
				if pv := recover(); pv != nil {
					fmt.Fprintf(os.Stderr, "[skillversion] commitFiles panic on %s: %v\n", c.Hash.String()[:7], pv)
				}
			}()
			// 2026-07-18 改:传 pathPrefix 给 commitFiles — 让 commit 涉及
			// 的文件列表在源头就按 skill 过滤,前端 modal 展开时不再
			// 看到其他 skill 的文件。
			if files, ferr := commitFiles(c, prefix); ferr == nil {
				entry.Files = files
			}
		}()
		out = append(out, entry)
		count++
		return nil
	})
	if out == nil {
		out = []CommitEntry{}
	}
	return out, nil
}

// LogAll 是 Log 的全量便捷包装(无 path filter,跟阶段一 API 兼容)。
//
// 2026-07-17 增:HTTP API 之前绑定的是 Log(limit) 单参,改成 (limit, path) 后
// 旧调用方会编译失败,所以保留 LogAll 给需要"全仓 log"的端点用。
func (r *Repo) LogAll(limit int) ([]CommitEntry, error) {
	return r.Log(limit, "")
}

// errStop 内部 sentinel,ForEach 用它提前终止。
type errStopSentinel struct{}

func (errStopSentinel) Error() string { return "stop" }

var errStop = errStopSentinel{}

// commitFiles 拿 commit 涉及的文件路径列表(走 Parent Tree diff)。
//
// 2026-07-17 改:从 parent.Patch 改成 go-git 内置 c.Stats() — parent.Patch
// 在合并 commit / squash / 边界 shape 上会 panic(runtime error: slice
// bounds out of range),改用 commit 自带的 stats 接口,go-git 内部已
// 处理好边界。Stats() 返 []CommitStats,每条 .Name 就是变更的文件名。
//
// 2026-07-18 增:pathPrefix 非空时只返该前缀下的文件(per-skill 过滤)。
// 前端 VersionHistoryPanel 展开 commit 时只显示当前 skill 的文件,
// 不传 pathPrefix 走原全量返回(给"全仓 log"接口用)。
//
// 对 root commit(无 parent)走 Tree().Files() 列全部文件 — Stats() 在
// root commit 上也是空 slice(没有 parent 没法 diff),所以仍走 Tree。
func commitFiles(c *object.Commit, pathPrefix string) ([]string, error) {
	hasPrefix := pathPrefix != ""
	prefixSlash := pathPrefix + "/"
	if c.NumParents() == 0 {
		tree, err := c.Tree()
		if err != nil {
			return nil, err
		}
		var files []string
		_ = tree.Files().ForEach(func(f *object.File) error {
			if hasPrefix && !strings.HasPrefix(f.Name, prefixSlash) {
				return nil
			}
			files = append(files, f.Name)
			return nil
		})
		return files, nil
	}
	// 2026-07-17 改:用 c.Stats() 替代 parent.Patch(c) — 后者在合并/squash
	// 上 panic。Stats() 走 commit object 自带的统计,内部已处理多 parent
	// 场景,不会 panic。
	stats, err := c.Stats()
	if err != nil {
		return nil, err
	}
	files := make([]string, 0, len(stats))
	for _, s := range stats {
		// 2026-07-18 增:pathPrefix 过滤 — 用 prefixSlash 而非裸 prefix,
		// 避免 "agents/demo-global" 误匹配 "agents/demo-global-extra/..."。
		if hasPrefix && !strings.HasPrefix(s.Name, prefixSlash) {
			continue
		}
		files = append(files, s.Name)
	}
	return files, nil
}

// CheckoutReset reset 工作区到指定 commit(hard 模式)。
func (r *Repo) CheckoutReset(commit string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	repo, err := r.open()
	if err != nil {
		return err
	}
	wt, err := repo.Worktree()
	if err != nil {
		return err
	}
	if st, _ := wt.Status(); !st.IsClean() {
		return ErrWorkingTreeDirty
	}
	// 解析 commit ref → hash,然后用 plumbing.HashReference 切 HEAD + HardReset。
	hash, herr := resolveHash(repo, commit)
	if herr != nil {
		return herr
	}
	if err := repo.Storer.SetReference(plumbing.NewHashReference(plumbing.HEAD, hash)); err != nil {
		return err
	}
	return wt.Reset(&git.ResetOptions{Mode: git.HardReset})
}

// DiscardChanges 丢弃工作区未提交改动(等价 git checkout -- . + git clean -fd)。
func (r *Repo) DiscardChanges() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	repo, err := r.open()
	if err != nil {
		return err
	}
	wt, err := repo.Worktree()
	if err != nil {
		return err
	}
	if st, _ := wt.Status(); st.IsClean() {
		return nil
	}
	if err := wt.AddWithOptions(&git.AddOptions{All: true}); err != nil {
		return err
	}
	return wt.Reset(&git.ResetOptions{Mode: git.HardReset})
}

// Pull 手动 pull(从远端拉取,fast-forward only)。
//
// 2026-07-17 增:go-git v5 Pull = Fetch + Merge,只能 fast-forward;若远端有
// non-ff 改动会报 ErrNonFastForwardUpdate,这时让用户先 commit 自己工作区
// 改动(或 discard),再 retry。Force: true 允许非快进更新,本地会丢失
// 未 push 的 commit,所以默认不打开。
func (r *Repo) Pull() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	auth, err := loadAuthFromConfig()
	if err != nil {
		return err
	}
	repo, err := r.open()
	if err != nil {
		return err
	}
	wt, err := repo.Worktree()
	if err != nil {
		return err
	}
	// 工作区有未提交改动时拒绝 pull,避免冲突或覆盖
	if st, _ := wt.Status(); !st.IsClean() {
		return ErrWorkingTreeDirty
	}
	return wt.Pull(&git.PullOptions{
		RemoteName: "origin",
		Auth:       auth,
		Force:      false,
	})
}

// resolveHash 解析 ref/短 hash/全 hash → plumbing.Hash。
func resolveHash(repo *git.Repository, ref string) (plumbing.Hash, error) {
	if ref == "" || ref == "HEAD" {
		h, err := repo.Head()
		if err != nil {
			return plumbing.ZeroHash, err
		}
		return h.Hash(), nil
	}
	if h, err := repo.ResolveRevision(plumbing.Revision(ref)); err == nil && h != nil {
		return *h, nil
	}
	hash := plumbing.NewHash(ref)
	return hash, nil
}

// CheckoutRestore 切 HEAD 到指定 commit(detached,不动工作区)。
func (r *Repo) CheckoutRestore(commit string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	repo, err := r.open()
	if err != nil {
		return err
	}
	return repo.Storer.SetReference(plumbing.NewHashReference(plumbing.HEAD, plumbing.NewHash(commit)))
}

// Diff 两个 commit 之间的 unified diff(走系统 git CLI,不走 go-git)。
//
// 2026-07-18 重写:之前用 go-git Tree walk + 手写 LCS(原 repo_ops.go 注释
// 里说"go-git 自己内部读 .git/ 不需要 fork"),实测在 wails v3 alpha.60
// webview 子进程里 hang 15s+ — webview 是 Chromium 多进程架构,go-git
// 大量小对象读 + ObjectIter 在子进程里 IO 调度异常,go run 单测秒返,
// webview 内卡死。
//
// 现在优先走 `git -C <path> diff A B`,3s ctx 超时:
//   - 系统装了 git 且 wails sandbox 没拦 fork → 秒返,正常 diff
//   - wails sandbox 拦 fork / exec.Command hang / git 不在 PATH → 返
//     ErrSandboxUnavailable,GitDiff handler 转成 stub + hint 提示用户
//     用 CLI。
//
// 为什么不再尝试手写 LCS:它本身是对的,但 webview 内 IO 卡死的根因是
// go-git 库本身而非 walk 逻辑。LCS 代码删除避免误导后人"以为是这里卡"。
//
// from 传空 = 不传 from 参数,git 会从 to 的第一个 parent 开始 diff
// (等价于 `git diff <to>`,适合 root commit 场景)。
func (r *Repo) Diff(from, to string) (string, error) {
	if !r.IsInitialized() {
		return "", ErrRepoNotInitialized
	}
	// 2026-07-18:不持 r.mu — exec.Command 是外部进程,跟 go-git 仓库对
	// 象访问无关,不需要序列化 git 命令。
	args := []string{"-C", r.path, "diff"}
	if from != "" {
		args = append(args, from)
	}
	args = append(args, to)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", args...)
	out, err := cmd.Output()
	if err != nil {
		// 2026-07-18:任何错误(ENOENT / EACCES / 沙箱拦 / 超时 / git
		// 进程非零退出)统一转 ErrSandboxUnavailable,GitDiff handler
		// 据此返 stub。stderr 信息丢 logger 方便排查。
		if ee, ok := err.(*exec.ExitError); ok {
			logger.Warn("skillversion: git diff failed: stderr=%s err=%v", string(ee.Stderr), err)
		} else {
			logger.Warn("skillversion: git diff exec failed: %v", err)
		}
		return "", ErrSandboxUnavailable
	}
	return string(out), nil
}

// Push 手动 push(给"重试失败"用)。
func (r *Repo) Push() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.pushLocked(); err != nil {
		r.recordPushError(err)
		return err
	}
	r.recordPushError(nil)
	return nil
}

// hasTokenFile 读 TokenFile 是否存在且非空。
func hasTokenFile() bool {
	cfg := getGitConfig()
	if cfg.TokenFile == "" {
		return false
	}
	fi, err := os.Stat(cfg.TokenFile)
	if err != nil {
		return false
	}
	return fi.Size() > 0
}

// _ 抑制 unused import 警告
var _ = filepath.Join
var _ = time.Second
var _ = plumbing.HEAD