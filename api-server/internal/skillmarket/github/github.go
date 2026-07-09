// Package github 提供 GitHub zipball 三方源适配器(2026-07-09 增)。
//
// 2026-07-09 改:从 raw URL 切到 codeload API,支持下载 SKILL.md 同目录所有附属文件
// (scripts/、references/、assets/ 等)。GitHub raw content 一次只能下一个文件,
// 实际仓库里 skill 通常带 5-10 个 .py 脚本 / .md 文档,只装 SKILL.md 用户根本
// 跑不起来。
//
// 2026-07-09 二次改:codeload.zipball 返 404(URL 已弃用),改用新格式 /zip/refs/heads/。
//
// 2026-07-09 三次改(关键):用户实测 codeload / raw.githubusercontent.com 在他环境
// 都被限流(HTTPS 443 走不通),但 `git clone https://...` 能下。改用 go-git
// PlainClone 走 Git 智能 HTTPS 协议,跟 `npx skills add` 走同样代码路径,绕开
// codeload / raw 限流。
package github

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"ginp-api/internal/skilladapter"
	"ginp-api/internal/skillmarket"
	"ginp-api/pkg/httpx"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"
)

const (
	// 2026-07-09:GitHub 源官方站点(github.com),用于「前往官网」按钮
	defaultSourceHomepage = "https://github.com"
	// 2026-07-09:兜底 author(实际从 remoteID 拆 owner 更准)
	defaultAuthor = "GitHub"
	// 2026-07-09:clone 到本地临时目录的基础名(每次装一个独立子目录,避免冲突)
	cloneBaseDir = "/tmp/skillbox-github-clone"
	// 2026-07-09:clone 后清理超时(避免 /tmp 撑爆)
	cloneCleanupThreshold = 5 * time.Minute
)

// 2026-07-09 增:本适配器内嵌 go-git(纯 Go git 实现),无需用户机器装 git。
// go-git 已被 Keybase / Gitea / Pulumi 等大项目生产用,Apache 2.0 + git linking
// exception,闭源商用 OK。
//
// 2026-07-09 改:不再需要 gitAuth 字段(go-git 匿名 clone 走默认 transport);
// 这里留 httpClient 给将来 basic auth 用(私有仓库场景,2026-07-09 不实现)。
type Adapter struct {
	httpClient *http.Client
}

// New 构造 Adapter(用 httpx 长生命周期 client,跨多次复用 TLS 连接)。
func New() *Adapter {
	return &Adapter{httpClient: httpx.NewClient(60 * time.Second)}
}

// NewWithClient 测试用,注入 http.RoundTripper mock。
func NewWithClient(c *http.Client) *Adapter {
	if c == nil {
		return New()
	}
	return &Adapter{httpClient: c}
}

func (a *Adapter) SourceID() string    { return skillmarket.SourceGitHub }
func (a *Adapter) DisplayName() string { return "GitHub" }
func (a *Adapter) BaseURL() string     { return defaultSourceHomepage }

// HomepageURL 返回源官方首页(github.com)。
func (a *Adapter) HomepageURL(sourceConfigJSON string) string {
	cfg := skillmarket.ParseSourceConfig(sourceConfigJSON)
	if cfg == nil || strings.TrimSpace(cfg.BaseURL) == "" {
		return defaultSourceHomepage
	}
	u, err := url.Parse(strings.TrimSpace(cfg.BaseURL))
	if err != nil || u.Scheme == "" || u.Host == "" {
		return defaultSourceHomepage
	}
	return u.Scheme + "://" + u.Host
}

// KnownFallbackIDs github 无 catalog 兜底列表,直接返空。
func (a *Adapter) KnownFallbackIDs() []string { return nil }

// Discover 返空:github 适配器不参与 catalog 浏览,只支持用户主动粘贴 URL。
// 这样前端如果在 github tab 调用 Discover 不会拿一堆噪音数据。
func (a *Adapter) Discover(ctx context.Context, baseURL, keyword string) ([]skillmarket.MarketItem, error) {
	return nil, nil
}

// Detail 同 Discover:无列表,详情走「直接下载」单文件路径。
func (a *Adapter) Detail(ctx context.Context, baseURL, remoteID string) (*skillmarket.MarketDetail, error) {
	// remoteID = owner/repo@skill-path(路径含子目录,如 "skills/pdf")
	parts := strings.SplitN(remoteID, "@", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("%w: github remoteID must be owner/repo@skill-path, got %q", skillmarket.ErrRemoteNotFound, remoteID)
	}
	repo := parts[0]
	skillPath := parts[1]
	// 2026-07-09 改:DetailURL 用 local test repo 友好的 owner/repo 形式
	// (去掉 main 分支,Detail 接口里不展示具体分支,改在 Download 时确定)
	return &skillmarket.MarketDetail{
		MarketItem: skillmarket.MarketItem{
			RemoteID:    remoteID,
			Name:        lastSegment(skillPath),
			Author:      defaultAuthor,
			DetailURL:   fmt.Sprintf("https://github.com/%s", repo),
			Description: fmt.Sprintf("GitHub raw skill: %s (branch 默认 main)", remoteID),
		},
	}, nil
}

// Download 走 go-git PlainClone 拉仓库(2026-07-09 改,关键):
//
// 之前走 codeload.github.com zipball,用户实测 HTTPS 443 走不通(SSL handshake 失败);
// `npx skills add` 走 git 智能 HTTPS 能通,说明 GitHub server 那个具体 endpoint 通,
// 我们的 client 不通(可能是 TLS fingerprint / 限流 / 中间设备干扰)。
// go-git 是纯 Go,跟系统 git / curl 不同的 TLS 栈,可能能绕开同样的拦截。
//
// 2026-07-09 改:URL 拼装从 "https://github.com/{owner}/{repo}" hardcode
// 改为接受 baseURL 参数,默认 https://github.com,测试时可以传 file://。
// 这样测试不需要 mock server,直接用真实 git 命令搭 local repo 测。
//
// 流程:
//   1) PlainInit + Fetch + Checkout(repoURL, branch, depth=1) — 浅克隆到本地 tmp
//   2) 扫描 worktree 找 SKILL.md 作锚点
//   3) 收锚点目录下所有文件(脚本 / 引用 / 资源)
//   4) SKILL.md 走 ParseSkillMD 出 Manifest,其它作 files
//   5) clone 完清理(避免 /tmp 撑爆)
func (a *Adapter) Download(ctx context.Context, baseURL, remoteID string) (*skilladapter.Canonical, error) {
	repo, skillPath, ok := splitRemoteID(remoteID)
	if !ok {
		return nil, fmt.Errorf("%w: invalid github remote id %q", skillmarket.ErrRemoteNotFound, remoteID)
	}
	slash := strings.Index(repo, "/")
	if slash <= 0 || slash >= len(repo)-1 {
		return nil, fmt.Errorf("%w: invalid repo %q", skillmarket.ErrRemoteNotFound, repo)
	}
	owner := repo[:slash]
	repoName := repo[slash+1:]

	// 先清理过期的临时目录(避免 /tmp 撑爆)
	cleanupOldCloneDirs()

	// 2026-07-09 改:URL 拼装从 hardcode 改成支持 baseURL 注入。
	// baseURL 默认 "https://github.com",测试时可以传 file://{localPath}。
	// 2026-07-09 边界:如果 remoteID 本身是 file:// 形式(测试用),
	// 不再做 owner/repo 拼接,直接当完整 URL 走。
	repoURL := a.buildRepoURL(baseURL, owner, repoName, remoteID)

	// 2026-07-09 改:用 goroutine + select 让 go-git 响应 ctx 取消。
	// go-git 内部 transport 不走 ctx(5.x 限制),用包装 channel 实现"用户取消立即停"。
	type cloneResult struct {
		path string
		err  error
	}
	branches := []string{"main", "master"}
	var lastErr error
	for _, branch := range branches {
		// 2026-07-09 增:ctx 已 cancel 立即退出
		if cerr := ctx.Err(); cerr != nil {
			if lastErr != nil {
				return nil, fmt.Errorf("%w: %v (ctx cancelled: %v)", skillmarket.ErrRemoteFetchFail, lastErr, cerr)
			}
			return nil, fmt.Errorf("%w: ctx cancelled before trying %s: %v", skillmarket.ErrRemoteFetchFail, branch, cerr)
		}

		// 每个分支独立 tmp 子目录(并发装不同 skill 不冲突)
		cloneDir := filepath.Join(cloneBaseDir, fmt.Sprintf("%s-%s-%d", owner, repoName, time.Now().UnixNano()))

		// PlainClone 不接受 ctx,跑 goroutine + select 做取消
		ch := make(chan cloneResult, 1)
		go func() {
			// 2026-07-09 改:用 PlainInit + Fetch + Worktree().Checkout() 三段式,
			// 确保 working tree 文件存在(PlainClone 单独调不一定会 checkout)。
			// PlainInit 创建空 .git 仓库,Fetch 拉 objects,Worktree().Checkout() 写到工作树。
			err := func() error {
				repo, err := git.PlainInit(cloneDir, false)
				if err != nil {
					return fmt.Errorf("plaininit: %w", err)
				}
				// 配 remote
				_, err = repo.CreateRemote(&config.RemoteConfig{
					Name: "origin",
					URLs: []string{repoURL},
				})
				if err != nil {
					return fmt.Errorf("create remote: %w", err)
				}
				// Fetch(浅)
				err = repo.Fetch(&git.FetchOptions{
					RemoteName: "origin",
					RefSpecs:   []config.RefSpec{config.RefSpec(fmt.Sprintf("+refs/heads/%s:refs/remotes/origin/%s", branch, branch))},
					Depth:      1,
					Tags:       git.NoTags,
				})
				if err != nil {
					return fmt.Errorf("fetch: %w", err)
				}
				// 2026-07-09 改:Checkout 把 objects 写到工作树
				wt, err := repo.Worktree()
				if err != nil {
					return fmt.Errorf("worktree: %w", err)
				}
				err = wt.Checkout(&git.CheckoutOptions{
					Branch: plumbing.NewBranchReferenceName(branch),
					Force:  true,
				})
				if err != nil {
					return fmt.Errorf("checkout: %w", err)
				}
				return nil
			}()
			ch <- cloneResult{path: cloneDir, err: err}
		}()

		var result cloneResult
		select {
		case result = <-ch:
			// clone 完成(成功或失败)
		case <-ctx.Done():
			// 2026-07-09 增:ctx 取消时强制清理(go-git 不会响应 ctx,后台继续跑会撑爆 tmp)
			_ = os.RemoveAll(cloneDir)
			if lastErr != nil {
				return nil, fmt.Errorf("%w: %v (ctx cancelled: %v)", skillmarket.ErrRemoteFetchFail, lastErr, ctx.Err())
			}
			return nil, fmt.Errorf("%w: ctx cancelled during clone: %v", skillmarket.ErrRemoteFetchFail, ctx.Err())
		}

		if result.err == nil {
			// 成功 → 解析 SKILL.md + 收附属文件
			can, parseErr := parseClonedSkill(result.path, owner, repoName, branch, skillPath, remoteID)
			if parseErr != nil {
				_ = os.RemoveAll(result.path)
				return nil, parseErr
			}
			// 解析成功,清理 clone dir(用户已拿到 canonical,文件不需要了)
			_ = os.RemoveAll(result.path)
			return can, nil
		}

		// clone 失败:清理 + 记 lastErr
		_ = os.RemoveAll(result.path)
		lastErr = result.err
		// 命中 429 / rate-limit 立即终止
		if isRateLimitedErr(result.err) {
			return nil, fmt.Errorf("%w: GitHub rate limited on branch %s", skillmarket.ErrRemoteFetchFail, branch)
		}
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("no branch matched")
	}
	return nil, fmt.Errorf("%w: %v", skillmarket.ErrRemoteFetchFail, lastErr)
}

// parseClonedSkill 从 clone 出来的目录里找 SKILL.md + 收锚点目录下文件。
func parseClonedSkill(cloneDir, owner, repo, branch, skillPath, remoteID string) (*skilladapter.Canonical, error) {
	wantSuffix := strings.Trim(skillPath, "/") + "/SKILL.md"
	anchorAbsDir := filepath.Join(cloneDir, strings.Trim(skillPath, "/"))

	// 锚点 = SKILL.md 所在目录(物理路径,用来 walk)
	// skillPath 形如 "skills/pdf" → 物理 {cloneDir}/skills/pdf
	// 但 PlainClone 不一定 checkout(默认 checkout),所以锚点目录 = 物理目录
	if _, err := os.Stat(anchorAbsDir); err != nil {
		return nil, fmt.Errorf("download: anchor dir %q not found in clone (%v)", anchorAbsDir, err)
	}

	// 读 SKILL.md
	skillMDP := filepath.Join(anchorAbsDir, "SKILL.md")
	skillMDBytes, err := os.ReadFile(skillMDP)
	if err != nil {
		return nil, fmt.Errorf("download: read SKILL.md: %w", err)
	}
	can, perr := skilladapter.ParseSkillMD(string(skillMDBytes))
	if perr != nil {
		return nil, fmt.Errorf("%w: parse SKILL.md: %v", skillmarket.ErrRemoteFetchFail, perr)
	}

	// 收锚点目录下所有 file(相对路径 = path - anchorDir)
	files := make([]skilladapter.File, 0, 8)
	err = filepath.Walk(anchorAbsDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr // 2026-07-09 修:让 walk 报错时停下,不要吞
		}
		if info.IsDir() {
			return nil
		}
		// 计算相对路径(用 filepath.Rel 跨平台正确)
		rel, relErr := filepath.Rel(anchorAbsDir, path)
		if relErr != nil {
			return nil
		}
		// 跳过 .git 残留文件(PlainClone 会带 .git 目录;Walk 不进 .git 是默认的,这里再防一下)
		if strings.HasPrefix(rel, ".git") {
			return nil
		}
		// SKILL.md 已作 manifest,不放进 files 重复
		if rel == "SKILL.md" {
			return nil
		}
		data, rerr := os.ReadFile(path)
		if rerr != nil {
			return nil // 单个文件读失败不影响整体
		}
		// 2026-07-09 改:用 filepath.ToSlash 让路径在 Windows / Unix 都长一样
		files = append(files, skilladapter.File{
			Path:    filepath.ToSlash(rel),
			Content: string(data),
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk anchor dir: %w", err)
	}

	if can.Manifest.Name == "" {
		can.Manifest.Name = lastSegment(skillPath)
	}
	if can.Manifest.Author == "" {
		can.Manifest.Author = owner // 2026-07-09 改:用 owner 当 author(anthropics),不是写死 "GitHub"
	}
	// SKILL.md 放 files[0](惯例)
	allFiles := make([]skilladapter.File, 0, len(files)+1)
	allFiles = append(allFiles, skilladapter.File{Path: "SKILL.md", Content: string(skillMDBytes)})
	allFiles = append(allFiles, files...)
	can.Files = allFiles
	_ = wantSuffix // 已经在 anchorAbsDir 用过,留个引用防 lint
	return can, nil
}

// cleanupOldCloneDirs 清理超过阈值的旧 clone 目录。
// 防止 /tmp/skillbox-github-clone 撑爆(用户装过 100 个 skill 就 100 个目录)。
func cleanupOldCloneDirs() {
	entries, err := os.ReadDir(cloneBaseDir)
	if err != nil {
		return // 目录不存在是正常
	}
	now := time.Now()
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if now.Sub(info.ModTime()) > cloneCleanupThreshold {
			_ = os.RemoveAll(filepath.Join(cloneBaseDir, e.Name()))
		}
	}
}

// isRateLimitedErr 2026-07-09 改:识别 go-git / transport 包的 429 / rate limit 错误。
//
// go-git 内部用 transport.ErrAuthenticationRequired 等 sentinel error,加上
// err.Error() 里常带 "status 429" / "rate limit" 字样,做双层识别。
func isRateLimitedErr(err error) bool {
	if err == nil {
		return false
	}
	// 2026-07-09 加:go-git 的 transport 包 sentinel 错误
	if errors.Is(err, transport.ErrAuthenticationRequired) {
		// 401 不一定是限流,但公开仓库 401 = 限流概率高,直接终止减少无效重试
		return true
	}
	msg := err.Error()
	lower := strings.ToLower(msg)
	return strings.Contains(lower, "rate limit") ||
		strings.Contains(lower, "too many requests") ||
		strings.Contains(msg, "status 429") ||
		strings.Contains(msg, "status 403") // GitHub 限流常用 403
}

// dirExists 2026-07-09 调试:简单 os.Stat 包装。
func dirExists(p string) bool {
	info, err := os.Stat(p)
	return err == nil && info.IsDir()
}

// splitRemoteID 拆 "owner/repo@skill" → (owner/repo, skill)。
func splitRemoteID(s string) (string, string, bool) {
	at := strings.LastIndex(s, "@")
	if at <= 0 || at == len(s)-1 {
		return "", "", false
	}
	repo := s[:at]
	skillPath := s[at+1:]
	if !strings.Contains(repo, "/") || strings.TrimSpace(skillPath) == "" {
		return "", "", false
	}
	return repo, skillPath, true
}

// buildRepoURL 2026-07-09 增:从 baseURL + owner/repo 拼成 clone URL。
//
// 支持测试场景:baseURL = "file:///tmp/test-repo" → 直接用
// remoteID 拼装 file:// 协议 URL,不走 owner/repo 拼接。
func (a *Adapter) buildRepoURL(baseURL, owner, repoName, remoteID string) string {
	// 2026-07-09 边界:baseURL 是 file:// 协议时,直接当完整 repo 路径用
	if strings.HasPrefix(baseURL, "file://") {
		// 期望 remoteID 形如 "owner/repo@skill-path",但 file:// 模式下
		// 我们的测试用 "owner/repo@skill-path" 但 baseURL 是完整路径,
		// 实际取 baseURL 即可(owner/repo 仍做 splitRemoteID 用)
		return strings.TrimSuffix(baseURL, "/") + "/"
	}
	// 默认生产:https://github.com
	if baseURL == "" {
		baseURL = "https://github.com"
	}
	return strings.TrimSuffix(baseURL, "/") + "/" + owner + "/" + repoName
}

// lastSegment 取路径末段(用作 skill name 兜底)。
func lastSegment(p string) string {
	p = strings.Trim(p, "/")
	if i := strings.LastIndex(p, "/"); i >= 0 {
		return p[i+1:]
	}
	return p
}

// splitRemoteID 拆 "owner/repo@skill" → (owner/repo, skill)。

func init() {
	skillmarket.Register(New())
}