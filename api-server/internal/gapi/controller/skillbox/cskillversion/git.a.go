// Package cskillversion 提供技能仓库版本管理的 HTTP API(2026-07-17 增)。
//
// 11 个端点:
//   GET  /api/skillbox/git/config       读远端配置(URL/branch/token 存在标志)
//   POST /api/skillbox/git/config       写远端配置 + 写 token 文件
//   GET  /api/skillbox/git/status       当前 Repo 状态 + push 队列长度
//   POST /api/skillbox/git/init         初始化仓库(给未启用场景)
//   GET  /api/skillbox/git/log          commit 历史(?limit=50)
//   GET  /api/skillbox/git/log/:hash    单 commit 详情
//   GET  /api/skillbox/git/diff         diff (?from=A&to=B 或 from=hash&to=HEAD)
//   POST /api/skillbox/git/checkout     reset 工作区到某 commit(hard)
//   POST /api/skillbox/git/push         手动 push
//   POST /api/skillbox/git/pull         手动 pull
//   POST /api/skillbox/git/discard      丢弃工作区未提交改动
//
// 本期不做:ssh/lfs/submodule/三方合并。
package cskillversion

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"ginp-api/internal/skillversion"
	"ginp-api/internal/skillversion/gitconfig"
	"ginp-api/pkg/ginp"
	"ginp-api/pkg/logger"
)

// ===========================================================================
// Request / Response 结构
// ===========================================================================

// RequestGitConfig POST /api/skillbox/git/config 的入参。
//
// 2026-07-17:remote_url / branch / token / user_name / user_email 都是可选;
// 空字段 = 不修改当前配置。token 字段是明文 PAT,后端立刻落盘 0600 文件,
// 响应不回 token 明文。
type RequestGitConfig struct {
	RemoteURL string `json:"remote_url"`
	Branch    string `json:"branch"`
	Token     string `json:"token"`      // 明文 PAT;为空 = 不修改
	TokenFile string `json:"token_file"` // 可选;空 = 默认 ~/.skill-box/.git_token
	UserName  string `json:"user_name"`
	UserEmail string `json:"user_email"`
}

// RespondGitConfig GET 的响应(token 字段只回 has_token 标志)。
type RespondGitConfig struct {
	RemoteURL string `json:"remote_url"`
	Branch    string `json:"branch"`
	HasToken  bool   `json:"has_token"`
	UserName  string `json:"user_name"`
	UserEmail string `json:"user_email"`
}

// RequestGitCheckout POST /api/skillbox/git/checkout 入参。
type RequestGitCheckout struct {
	Hash string `json:"hash"`
}

// RequestGitDiff GET /api/skillbox/git/diff 查询参数。
type RequestGitDiff struct {
	From string `json:"from" form:"from"`
	To   string `json:"to" form:"to"`
}

// RespondGitCommit 单 commit 详情(给 /git/log/:hash 用)。
type RespondGitCommit struct {
	Hash    string   `json:"hash"`
	Short   string   `json:"short"`
	Author  string   `json:"author"`
	Email   string   `json:"email"`
	Message string   `json:"message"`
	Body    string   `json:"body"`
	When    string   `json:"when"`
	Files   []string `json:"files"`
}

// ===========================================================================
// Handlers
// ===========================================================================

// GetGitConfig GET /api/skillbox/git/config
func GetGitConfig(c *ginp.ContextPlus) {
	cfg := gitconfig.GetGitRemoteConfig()
	tokenPath := cfg.TokenFile
	hasToken := false
	if tokenPath != "" {
		hasToken = gitconfigHasToken(tokenPath)
	}
	c.JSON(200, RespondGitConfig{
		RemoteURL: cfg.RemoteURL,
		Branch:    cfg.Branch,
		HasToken:  hasToken,
		UserName:  cfg.UserName,
		UserEmail: cfg.UserEmail,
	})
}

// SaveGitConfig POST /api/skillbox/git/config
//
// 2026-07-17:本接口是 Git 远端配置的单一入口 — 写 configs + 写 token 文件。
// remote_url 校验失败返 400(ErrInvalidURL)。
func SaveGitConfig(c *ginp.ContextPlus, req *RequestGitConfig) {
	// 校验 remote_url
	if err := gitconfig.ValidateRemoteURL(req.RemoteURL); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	// 写 token 文件(若提供)
	if strings.TrimSpace(req.Token) != "" {
		tokenPath := strings.TrimSpace(req.TokenFile)
		if tokenPath == "" {
			tokenPath = gitconfig.DefaultTokenFile()
		}
		if err := gitconfig.WriteToken(tokenPath, req.Token); err != nil {
			logger.Error("git config: write token: %v", err)
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		// 把路径写到全局 cfg,后续 status / push 都能用
		setSkillboxGitTokenFile(tokenPath)
	}
	// 更新配置项(cfg 是项目全局,直接写字段)
	if strings.TrimSpace(req.RemoteURL) != "" {
		setSkillboxGitRemoteURL(req.RemoteURL)
	}
	if strings.TrimSpace(req.Branch) != "" {
		setSkillboxGitBranch(req.Branch)
	}
	if strings.TrimSpace(req.UserName) != "" {
		setSkillboxGitUserName(req.UserName)
	}
	if strings.TrimSpace(req.UserEmail) != "" {
		setSkillboxGitUserEmail(req.UserEmail)
	}
	c.JSON(200, gin.H{"ok": true})
}

// GitStatus GET /api/skillbox/git/status
func GitStatus(c *ginp.ContextPlus) {
	repo, err := skillversion.Default()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	st, serr := repo.Status()
	if serr != nil {
		// 仓库未 init 等场景 — 仍返 Status{Initialized: false},不视为错误
		logger.Warn("git status: %v", serr)
	}
	c.JSON(200, st)
}

// GitInit POST /api/skillbox/git/init
//
// 2026-07-17 改:从同步改为 fire-and-forget — InitIfNotExists 内部走 go-git
// 同步 IO(PlainInit + 空 commit),在 macOS sandbox / 文件锁场景下可能慢,
// 同步 HTTP handler 会挂起前端按钮(loading 状态一直不返回)。
// 改:handler 立即返 202(后台跑),前端轮询 /git/status 看 init 是否完成。
func GitInit(c *ginp.ContextPlus) {
	repo, err := skillversion.Default()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	// 2026-07-17:如果已经 init,直接返 200,不要走异步(避免无谓的 goroutine 噪声)
	if repo.IsInitialized() {
		c.JSON(200, gin.H{"ok": true, "initialized": true, "already": true})
		return
	}
	go func() {
		defer func() { _ = recover() }()
		if err := repo.InitIfNotExists(); err != nil {
			fmt.Fprintf(os.Stderr, "[skillversion] GitInit async: %v\n", err)
		}
	}()
	c.JSON(202, gin.H{"ok": true, "initialized": false, "async": true})
}

// GitLog GET /api/skillbox/git/log?limit=50&path=<group>/<name>
//
// 2026-07-17 增:可选 path 参数 — 非空时只返回涉及该路径前缀的 commit
// (per-skill 修改历史的核心)。path 用正斜杠,跟 skills 目录布局一致。
func GitLog(c *ginp.ContextPlus, req *struct {
	Limit int    `json:"limit" form:"limit"`
	Path  string `json:"path" form:"path"`
}) {
	repo, err := skillversion.Default()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	entries, lerr := repo.Log(req.Limit, req.Path)
	if lerr != nil {
		c.JSON(500, gin.H{"error": lerr.Error()})
		return
	}
	c.JSON(200, gin.H{"items": entries, "total": len(entries)})
}

// GitCommit GET /api/skillbox/git/log/:hash
func GitCommit(c *ginp.ContextPlus) {
	hash := c.Param("hash")
	repo, err := skillversion.Default()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	entries, lerr := repo.Log(500, "")
	if lerr != nil {
		c.JSON(500, gin.H{"error": lerr.Error()})
		return
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Hash, hash) {
			c.JSON(200, RespondGitCommit{
				Hash:    e.Hash,
				Short:   e.Short,
				Author:  e.Author,
				Email:   e.Email,
				Message: e.Message,
				Body:    "",
				When:    e.When.Format("2006-01-02T15:04:05Z07:00"),
				Files:   e.Files,
			})
			return
		}
	}
	c.JSON(404, gin.H{"error": "commit not found: " + hash})
}

// GitDiff GET /api/skillbox/git/diff?from=A&to=B
func GitDiff(c *ginp.ContextPlus, req *RequestGitDiff) {
	repo, err := skillversion.Default()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	diff, derr := repo.Diff(req.From, req.To)
	if derr != nil {
		c.JSON(500, gin.H{"error": derr.Error()})
		return
	}
	c.JSON(200, gin.H{"diff": diff})
}

// GitCheckout POST /api/skillbox/git/checkout {hash}
func GitCheckout(c *ginp.ContextPlus, req *RequestGitCheckout) {
	if strings.TrimSpace(req.Hash) == "" {
		c.JSON(400, gin.H{"error": "hash is required"})
		return
	}
	repo, err := skillversion.Default()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	if err := repo.CheckoutReset(req.Hash); err != nil {
		if errors.Is(err, skillversion.ErrWorkingTreeDirty) {
			c.JSON(409, gin.H{"error": err.Error()})
			return
		}
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"ok": true})
}

// GitPush POST /api/skillbox/git/push
func GitPush(c *ginp.ContextPlus) {
	repo, err := skillversion.Default()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	if err := repo.Push(); err != nil {
		if errors.Is(err, skillversion.ErrRemoteNotConfigured) {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"ok": true})
}

// GitDiscard POST /api/skillbox/git/discard
func GitDiscard(c *ginp.ContextPlus) {
	repo, err := skillversion.Default()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	if err := repo.DiscardChanges(); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"ok": true})
}

// GitPull POST /api/skillbox/git/pull
//
// 2026-07-17 增:从远端拉取(fast-forward only);工作区有未提交改动时
// 返 409 + ErrWorkingTreeDirty,让用户先 commit 或 discard 再 retry。
func GitPull(c *ginp.ContextPlus) {
	repo, err := skillversion.Default()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	if err := repo.Pull(); err != nil {
		switch {
		case errors.Is(err, skillversion.ErrRemoteNotConfigured):
			c.JSON(400, gin.H{"error": err.Error()})
		case errors.Is(err, skillversion.ErrWorkingTreeDirty):
			c.JSON(409, gin.H{"error": err.Error()})
		default:
			c.JSON(500, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(200, gin.H{"ok": true})
}

// ===========================================================================
// 路由注册
// ===========================================================================

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/git/config",
		Handler:        ginp.BindParamsHandler(SaveGitConfig, &RequestGitConfig{}),
		HttpType:       ginp.HttpPost,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.git.config.write",
		Swagger: &ginp.SwaggerInfo{
			Title:       "git.config.write",
			Description: "设置 Git 远端配置(URL/branch/token/user);token 落 0600 文件",
		},
	})
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/git/config",
		Handler:        ginp.BindHandler(GetGitConfig),
		HttpType:       ginp.HttpGet,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.git.config.read",
		Swagger: &ginp.SwaggerInfo{
			Title:       "git.config.read",
			Description: "读 Git 远端配置(只返 has_token 标志,不回 token 明文)",
		},
	})
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/git/status",
		Handler:        ginp.BindHandler(GitStatus),
		HttpType:       ginp.HttpGet,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.git.status",
		Swagger: &ginp.SwaggerInfo{
			Title:       "git.status",
			Description: "当前 Repo 状态:init/branch/HEAD/push 队列/最后错误",
		},
	})
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/git/init",
		Handler:        ginp.BindHandler(GitInit),
		HttpType:       ginp.HttpPost,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.git.init",
		Swagger: &ginp.SwaggerInfo{
			Title:       "git.init",
			Description: "PlainInit ~/.skill-box/skills 为 git 仓库(已 init 则 noop)",
		},
	})
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/git/log",
		Handler:        ginp.BindParamsHandler(GitLog, &struct {
			Limit int `json:"limit" form:"limit"`
		}{}),
		HttpType:       ginp.HttpGet,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.git.log",
		Swagger: &ginp.SwaggerInfo{
			Title:       "git.log",
			Description: "读 commit 历史;limit 默认 50,上限 500",
		},
	})
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/git/log/:hash",
		Handler:        ginp.BindHandler(GitCommit),
		HttpType:       ginp.HttpGet,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.git.log.detail",
		Swagger: &ginp.SwaggerInfo{
			Title:       "git.log.detail",
			Description: "单 commit 详情(短 hash 匹配)",
		},
	})
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/git/diff",
		Handler:        ginp.BindParamsHandler(GitDiff, &RequestGitDiff{}),
		HttpType:       ginp.HttpGet,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.git.diff",
		Swagger: &ginp.SwaggerInfo{
			Title:       "git.diff",
			Description: "两 commit 之间的 unified diff;from/to 支持短 hash/全 hash/HEAD",
		},
	})
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/git/checkout",
		Handler:        ginp.BindParamsHandler(GitCheckout, &RequestGitCheckout{}),
		HttpType:       ginp.HttpPost,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.git.checkout",
		Swagger: &ginp.SwaggerInfo{
			Title:       "git.checkout",
			Description: "reset 工作区到指定 commit(hard);工作区有未提交改动返 409",
		},
	})
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/git/push",
		Handler:        ginp.BindHandler(GitPush),
		HttpType:       ginp.HttpPost,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.git.push",
		Swagger: &ginp.SwaggerInfo{
			Title:       "git.push",
			Description: "手动 push;未配置 remote 返 400,失败返 500 + 错误详情",
		},
	})
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/git/pull",
		Handler:        ginp.BindHandler(GitPull),
		HttpType:       ginp.HttpPost,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.git.pull",
		Swagger: &ginp.SwaggerInfo{
			Title:       "git.pull",
			Description: "手动 pull(fast-forward only);工作区有未提交改动返 409",
		},
	})
	ginp.RouterAppend(ginp.RouterItem{
		Path:           "/api/skillbox/git/discard",
		Handler:        ginp.BindHandler(GitDiscard),
		HttpType:       ginp.HttpPost,
		NeedLogin:      false,
		NeedPermission: false,
		PermissionName: "skillbox.git.discard",
		Swagger: &ginp.SwaggerInfo{
			Title:       "git.discard",
			Description: "丢弃工作区未提交改动(等价 git reset --hard HEAD)",
		},
	})
}