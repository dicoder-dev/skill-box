package skillversion

import (
	"fmt"
	"os"
	"path/filepath"
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
		s.HeadHash = head.Hash().String()
		s.HeadShort = head.Hash().String()[:7]
		s.Branch = head.Name().Short()
		if c, cerr := repo.CommitObject(head.Hash()); cerr == nil {
			msg := strings.SplitN(c.Message, "\n", 2)[0]
			if len(msg) > 120 {
				msg = msg[:120] + "…"
			}
			s.HeadMessage = msg
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
func (r *Repo) Log(limit int) ([]CommitEntry, error) {
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
	cIter, err := repo.Log(&git.LogOptions{From: head.Hash()})
	if err != nil {
		return nil, err
	}
	var out []CommitEntry
	count := 0
	_ = cIter.ForEach(func(c *object.Commit) error {
		if count >= limit {
			return errStop
		}
		entry := CommitEntry{
			Hash:    c.Hash.String(),
			Short:   c.Hash.String()[:7],
			Author:  c.Author.Name,
			Email:   c.Author.Email,
			Message: strings.SplitN(c.Message, "\n", 2)[0],
			When:    c.Author.When,
		}
		if files, ferr := commitFiles(repo, c); ferr == nil {
			entry.Files = files
		}
		out = append(out, entry)
		count++
		return nil
	})
	if out == nil {
		out = []CommitEntry{}
	}
	return out, nil
}

// errStop 内部 sentinel,ForEach 用它提前终止。
type errStopSentinel struct{}

func (errStopSentinel) Error() string { return "stop" }

var errStop = errStopSentinel{}

// commitFiles 拿 commit 涉及的文件路径列表(走 Parent Tree diff)。
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
	parent, err := c.Parent(0)
	if err != nil {
		return nil, err
	}
	patch, err := parent.Patch(c)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, fstat := range patch.Stats() {
		files = append(files, fstat.Name)
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
func (r *Repo) Diff(from, to string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	repo, err := r.open()
	if err != nil {
		return "", err
	}
	fromCommit, err := resolveCommit(repo, from)
	if err != nil {
		return "", err
	}
	toCommit, err := resolveCommit(repo, to)
	if err != nil {
		return "", err
	}
	fromTree, err := fromCommit.Tree()
	if err != nil {
		return "", err
	}
	toTree, err := toCommit.Tree()
	if err != nil {
		return "", err
	}
	patch, err := fromTree.Patch(toTree)
	if err != nil {
		return "", err
	}
	return patch.Message(), nil
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