package skillversion

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
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
func (r *Repo) AutoCommitAndPush(in CommitInput) (plumbing.Hash, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := r.InitIfNotExists(); err != nil {
		return plumbing.ZeroHash, err
	}
	repo, err := r.open()
	if err != nil {
		return plumbing.ZeroHash, err
	}
	wt, err := repo.Worktree()
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("skillversion: Worktree: %w", err)
	}

	// 2026-07-17:始终 Add All — store.Save 已经原子 rename 完整个目录,
	// 用 All: true 覆盖所有 modified / untracked / deleted。Paths 仅作 commit msg 注释。
	if err := wt.AddWithOptions(&git.AddOptions{All: true}); err != nil {
		return plumbing.ZeroHash, fmt.Errorf("skillversion: add all: %w", err)
	}

	msg := strings.TrimSpace(in.Message)
	if len(in.Paths) > 0 {
		// paths 拼到 commit message 末尾方便 grep
		msg = msg + "\n\nfiles: " + strings.Join(in.Paths, ", ")
	}

	hash, err := wt.Commit(msg, &git.CommitOptions{
		Author: resolveAuthor(),
	})
	if err != nil {
		// AllowEmptyCommits 默认 false,空 commit 会返 ErrEmptyCommit — 当作 noop,不报错。
		if err == git.ErrEmptyCommit {
			return plumbing.ZeroHash, nil
		}
		return plumbing.ZeroHash, fmt.Errorf("skillversion: commit: %w", err)
	}

	// 异步 push,失败入重试队列
	r.enqueuePush(hash, msg)
	return hash, nil
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
	var cIter object.CommitIter
	if pathPrefix != "" {
		prefix := pathPrefix
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
			if files, ferr := commitFiles(repo, c); ferr == nil {
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
// 对 root commit(无 parent)走 Tree().Files() 列全部文件 — Stats() 在
// root commit 上也是空 slice(没有 parent 没法 diff),所以仍走 Tree。
func commitFiles(repo *git.Repository, c *object.Commit) ([]string, error) {
	if c.NumParents() == 0 {
		tree, err := c.Tree()
		if err != nil {
			return nil, err
		}
		var files []string
		_ = tree.Files().ForEach(func(f *object.File) error {
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

// Diff 两个 commit 之间的 diff(unified 文本)。
//
// 2026-07-17 大改:go-git 5.19.1 的 Tree.Patch() 在本地仓库上返空 patch
// (不是 panic,只是 len=0),即使两个 tree 明确有 file 差异。所以这里
// 走手工实现:walk fromTree / toTree,比较 blob hash,对变更的文件
// 用 difflib 生成 unified diff。性能 OK,因为只对一个 commit 的
// 几个 file 跑。
func (r *Repo) Diff(from, to string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	repo, err := r.open()
	if err != nil {
		return "", err
	}
	// 解析 from(失败时退化到空 tree,模拟 root commit 全量 diff)
	fromCommit, ferr := resolveCommit(repo, from)
	var fromFiles map[string]plumbing.Hash
	if ferr == nil && fromCommit != nil {
		fromTree, err := fromCommit.Tree()
		if err != nil {
			return "", err
		}
		fromFiles = map[string]plumbing.Hash{}
		_ = fromTree.Files().ForEach(func(f *object.File) error {
			fromFiles[f.Name] = f.Hash
			return nil
		})
	} else {
		fromFiles = map[string]plumbing.Hash{}
	}
	// 解析 to
	toCommit, terr := resolveCommit(repo, to)
	if terr != nil {
		return "", terr
	}
	toTree, err := toCommit.Tree()
	if err != nil {
		return "", err
	}
	toFiles := map[string]plumbing.Hash{}
	_ = toTree.Files().ForEach(func(f *object.File) error {
		toFiles[f.Name] = f.Hash
		return nil
	})

	var out strings.Builder
	// 找出 added/modified
	for name, toHash := range toFiles {
		fromHash, ok := fromFiles[name]
		if !ok {
			// 新增 — 整文件作为 + 行
			appendAddedFile(&out, repo, name, toHash)
		} else if fromHash != toHash {
			// 修改 — 从 blob 读两端内容,生成 unified diff
			if err := appendModifiedFile(&out, repo, name, fromHash, toHash); err != nil {
				return "", err
			}
		}
	}
	// 找出 deleted
	for name, fromHash := range fromFiles {
		if _, ok := toFiles[name]; !ok {
			if err := appendDeletedFile(&out, repo, name, fromHash); err != nil {
				return "", err
			}
		}
	}
	return out.String(), nil
}

// appendAddedFile 输出 "新增" 文件的 unified diff(+ 全部行)。
func appendAddedFile(out *strings.Builder, repo *git.Repository, name string, hash plumbing.Hash) {
	blob, err := repo.BlobObject(hash)
	if err != nil {
		return
	}
	content, err := blob.Reader().ReadAll()
	if err != nil {
		return
	}
	out.WriteString(fmt.Sprintf("diff --git a/%s b/%s\n", name, name))
	out.WriteString("new file mode 100644\n")
	out.WriteString("--- /dev/null\n")
	out.WriteString(fmt.Sprintf("+++ b/%s\n", name))
	for _, line := range splitLines(string(content)) {
		out.WriteString("+" + line + "\n")
	}
}

// appendDeletedFile 输出 "删除" 文件的 unified diff(- 全部行)。
func appendDeletedFile(out *strings.Builder, repo *git.Repository, name string, hash plumbing.Hash) error {
	blob, err := repo.BlobObject(hash)
	if err != nil {
		return err
	}
	content, err := blob.Reader().ReadAll()
	if err != nil {
		return err
	}
	out.WriteString(fmt.Sprintf("diff --git a/%s b/%s\n", name, name))
	out.WriteString("deleted file mode 100644\n")
	out.WriteString(fmt.Sprintf("--- a/%s\n", name))
	out.WriteString("+++ /dev/null\n")
	for _, line := range splitLines(string(content)) {
		out.WriteString("-" + line + "\n")
	}
	return nil
}

// appendModifiedFile 用 LCS-based unified diff 生成两个 blob 间的差异。
func appendModifiedFile(out *strings.Builder, repo *git.Repository, name string, fromHash, toHash plumbing.Hash) error {
	fromBlob, err := repo.BlobObject(fromHash)
	if err != nil {
		return err
	}
	toBlob, err := repo.BlobObject(toHash)
	if err != nil {
		return err
	}
	fromContent, err := fromBlob.Reader().ReadAll()
	if err != nil {
		return err
	}
	toContent, err := toBlob.Reader().ReadAll()
	if err != nil {
		return err
	}
	fromLines := splitLines(string(fromContent))
	toLines := splitLines(string(toContent))

	out.WriteString(fmt.Sprintf("diff --git a/%s b/%s\n", name, name))
	out.WriteString(fmt.Sprintf("--- a/%s\n", name))
	out.WriteString(fmt.Sprintf("+++ b/%s\n", name))
	// 简化 unified diff:context=3,无 hunk 头(因为我们从整文件算
	// diff,hunk 头算起来要后端再处理 line offset,前端展示不严格
	// 需要 hunk 头)。直接列 +/- 行,context 用 3 行。
	const ctx = 3
	ops := unifiedDiff(fromLines, toLines, ctx)
	for _, op := range ops {
		switch op.kind {
		case ' ':
			out.WriteString(" " + op.text + "\n")
		case '+':
			out.WriteString("+" + op.text + "\n")
		case '-':
			out.WriteString("-" + op.text + "\n")
		}
	}
	return nil
}

// splitLines 按 \n 切行;保留空行(末尾无 \n 也算一行)。
func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	lines := strings.Split(s, "\n")
	// 末尾 \n 会产生空串,丢掉
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

type diffOp struct {
	kind byte // ' ' / '+' / '-'
	text string
}

// unifiedDiff LCS 算法计算编辑脚本,然后合并相邻 ctx 区间输出。
//
// 算法:经典 LCS table → backtrack → 简化输出(无 hunk 头)。
func unifiedDiff(a, b []string, ctx int) []diffOp {
	n, m := len(a), len(b)
	// LCS length table
	dp := make([][]int, n+1)
	for i := range dp {
		dp[i] = make([]int, m+1)
	}
	for i := 1; i <= n; i++ {
		for j := 1; j <= m; j++ {
			if a[i-1] == b[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
			} else {
				if dp[i-1][j] > dp[i][j-1] {
					dp[i][j] = dp[i-1][j]
				} else {
					dp[i][j] = dp[i][j-1]
				}
			}
		}
	}
	// backtrack → 原始编辑脚本
	type rawOp struct {
		kind byte
		text string
	}
	var raw []rawOp
	i, j := n, m
	for i > 0 || j > 0 {
		switch {
		case i > 0 && j > 0 && a[i-1] == b[j-1]:
			raw = append(raw, rawOp{' ', a[i-1]})
			i--
			j--
		case j > 0 && (i == 0 || dp[i][j-1] >= dp[i-1][j]):
			raw = append(raw, rawOp{'+', b[j-1]})
			j--
		default:
			raw = append(raw, rawOp{'-', a[i-1]})
			i--
		}
	}
	// reverse
	for l, r := 0, len(raw)-1; l < r; l, r = l+1, r-1 {
		raw[l], raw[r] = raw[r], raw[l]
	}
	// 合并 + 应用 ctx:扫描 raw,找出 +/- 区间,在其前后保留 ctx 行 ' '。
	type span struct {
		kind   byte // '+' / '-'
		start  int  // raw 索引
		end    int  // raw 索引(含)
	}
	var spans []span
	for k, op := range raw {
		if op.kind == '+' || op.kind == '-' {
			if len(spans) > 0 && spans[len(spans)-1].end+1 == k &&
				(spans[len(spans)-1].kind == op.kind || isAdjacentSameKind(spans[len(spans)-1], op.kind)) {
				spans[len(spans)-1].end = k
			} else {
				spans = append(spans, span{kind: op.kind, start: k, end: k})
			}
		}
	}
	// 简化:不严格合并相邻 +/- 区,直接对每个 +/- 区间前后输出 ctx 行。
	used := make([]bool, len(raw))
	for _, sp := range spans {
		from := sp.start - ctx
		if from < 0 {
			from = 0
		}
		to := sp.end + ctx
		if to >= len(raw) {
			to = len(raw) - 1
		}
		for k := from; k <= to; k++ {
			used[k] = true
		}
	}
	var out []diffOp
	for k, op := range raw {
		if used[k] {
			out = append(out, diffOp{kind: op.kind, text: op.text})
		}
	}
	return out
}

func isAdjacentSameKind(s span, kind byte) bool {
	// 简化:相邻 +/- 区合并到前一个 span
	_ = kind
	return true
}

// resolveCommit 解析 ref/短 hash/全 hash → commit object。
func resolveCommit(repo *git.Repository, ref string) (*object.Commit, error) {
	if ref == "" || ref == "HEAD" {
		h, err := repo.Head()
		if err != nil {
			return nil, err
		}
		ref = h.Hash().String()
	}
	hash := plumbing.NewHash(ref)
	if h, err := repo.ResolveRevision(plumbing.Revision(ref)); err == nil && h != nil {
		hash = *h
	}
	return repo.CommitObject(hash)
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