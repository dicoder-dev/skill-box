// Package github 提供 GitHub zipball 三方源适配器(2026-07-09 增)。
//
// 2026-07-09 改:从 raw URL 切到 zipball API,支持下载 SKILL.md 同目录的所有附属文件
// (scripts/、references/、assets/ 等)。GitHub raw content 一次只能下一个文件,
// 实际仓库里 skill 通常带 5-10 个 .py 脚本 / .md 文档,只装 SKILL.md 用户根本
// 跑不起来。
//
// URL 形态:
//   - https://github.com/{owner}/{repo}/blob/{branch}/{path}/SKILL.md
//     → 内部转 https://codeload.github.com/{owner}/{repo}/zipball/{branch}
//     → 解压 zip,找到 {path}/SKILL.md 所在目录(锚点),收该目录所有 file
//     → SKILL.md 作 Manifest 解析,其它作 files
//
// 不支持 catalog 浏览 / 搜索(那是 skillssh 的事)。
package github

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"ginp-api/internal/skilladapter"
	"ginp-api/internal/skillmarket"
	"ginp-api/pkg/httpx"
)

const (
	// 2026-07-09:源官方站点(github.com),用于「前往官网」按钮
	defaultSourceHomepage = "https://github.com"
	// 2026-07-09:兜底 author(实际从 remoteID 拆 owner 更准)
	defaultAuthor = "GitHub"
	// 2026-07-09:zip 大小上限 50MB(防恶意超大仓库)
	zipballMaxBytes = 50 << 20
)

// 2026-07-09:GitHub codeload.zipball host(用户公开 API,无需鉴权)。
// zipball 返回的 zip 顶层目录是 "{owner}-{repo}-{commit_sha}/...",不是仓库根,
// 解压时需要识别这层包裹目录并剥掉,只保留 user 视角的相对路径。
// var 而非 const:单测用 httptest 替换 base URL,跑完恢复。
var defaultZipballBase = "https://codeload.github.com"

// Adapter github 适配器。
type Adapter struct {
	httpClient *http.Client
}

// New 构造 Adapter。
func New() *Adapter {
	return &Adapter{httpClient: httpx.NewClient(15 * time.Second)}
}

// NewWithClient 测试用,注入 http.RoundTripper。
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
	return &skillmarket.MarketDetail{
		MarketItem: skillmarket.MarketItem{
			RemoteID:    remoteID,
			Name:        lastSegment(skillPath),
			Author:      defaultAuthor,
			DetailURL:   fmt.Sprintf("https://github.com/%s/tree/main/%s", repo, skillPath),
			Description: fmt.Sprintf("GitHub raw skill: %s (branch 默认 main)", remoteID),
		},
	}, nil
}

// Download 走 GitHub zipball API 拉仓库 zip,解压取 SKILL.md + 同目录所有附属文件。
//
// 2026-07-09 改(关键 bug):早期实现走 raw URL,只下一个 SKILL.md 文件。
// 实际仓库里 skill 通常带 5-10 个附属文件(pdf 仓库有 9 个 .py + LICENSE + reference.md),
// 只装 SKILL.md 用户根本跑不起来。改走 codeload.github.com zipball API。
//
// 流程:
//   1) 拼 zipball URL: codeload.github.com/{owner}/{repo}/zipball/{branch}
//   2) GET 拉 zip 字节流(50MB cap)
//   3) archive/zip 解压;zip 顶层是 {owner}-{repo}-{commit_sha} 包裹目录,先识别后剥掉
//   4) 在剥掉后的路径里找 SKILL.md 作为锚点(锚点目录 = SKILL.md 所在目录)
//   5) 收锚点目录下所有 file,SKILL.md 走 ParseSkillMD 出 Manifest,其它作 files
//   6) branch 默认 main,失败试 master
func (a *Adapter) Download(ctx context.Context, baseURL, remoteID string) (*skilladapter.Canonical, error) {
	repo, skillPath, ok := splitRemoteID(remoteID)
	if !ok {
		return nil, fmt.Errorf("%w: invalid github remote id %q", skillmarket.ErrRemoteNotFound, remoteID)
	}
	// 拆 owner / repo
	slash := strings.Index(repo, "/")
	if slash <= 0 || slash >= len(repo)-1 {
		return nil, fmt.Errorf("%w: invalid repo %q", skillmarket.ErrRemoteNotFound, repo)
	}
	owner := repo[:slash]
	repoName := repo[slash+1:]

	branches := []string{"main", "master"}
	var lastErr error
	for _, branch := range branches {
		can, err := a.downloadViaZipball(ctx, owner, repoName, branch, skillPath, remoteID)
		if err == nil && can != nil {
			return can, nil
		}
		lastErr = err
		// 命中 429 立即终止,跟 skillhub / skillssh 一致
		if isRateLimited(err) {
			return nil, fmt.Errorf("%w: GitHub rate limited on branch %s", skillmarket.ErrRemoteFetchFail, branch)
		}
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("no branch matched")
	}
	return nil, fmt.Errorf("%w: %v", skillmarket.ErrRemoteFetchFail, lastErr)
}

// downloadViaZipball 走 codeload.github.com 拉 zip,解压取 SKILL.md 所在目录全部文件。
func (a *Adapter) downloadViaZipball(ctx context.Context, owner, repo, branch, skillPath, remoteID string) (*skilladapter.Canonical, error) {
	zipURL := fmt.Sprintf("%s/%s/%s/zipball/%s", defaultZipballBase, owner, repo, branch)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, zipURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "skill-box/1.0 (+https://skillbox.local)")
	req.Header.Set("Accept", "application/zip")
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("zipball: status %d", resp.StatusCode)
	}
	// 50MB cap
	body, err := io.ReadAll(io.LimitReader(resp.Body, zipballMaxBytes))
	if err != nil {
		return nil, fmt.Errorf("read zip body: %w", err)
	}
	r, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return nil, fmt.Errorf("open zip: %w", err)
	}

	// 找 SKILL.md:必须匹配 skillPath 对应位置的 SKILL.md
	// 2026-07-09 改:之前用"zip 里第一个 SKILL.md"是错的(zip 里可能有多个 skill,
	// 各自有 SKILL.md;zip 顺序也不保证 SKILL.md 在前)。改成"路径匹配 skillPath"。
	// skillPath 形如 "skills/pdf",匹配 "skills/pdf/SKILL.md"。
	wantSuffix := strings.Trim(skillPath, "/") + "/SKILL.md"
	skillMDPath := ""
	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}
		name := strings.TrimPrefix(f.Name, "./")
		// zipball 顶层目录形如 "anthropics-skills-abc1234/skills/pdf/SKILL.md"
		// 剥掉第一层包裹目录,只留仓库内路径
		parts := strings.SplitN(name, "/", 2)
		if len(parts) < 2 {
			continue
		}
		repoInternalPath := parts[1]
		if repoInternalPath == wantSuffix {
			skillMDPath = repoInternalPath
			break
		}
	}
	if skillMDPath == "" {
		return nil, fmt.Errorf("download: SKILL.md not found in zipball at %q (branch=%s)", wantSuffix, branch)
	}

	// 锚点 = SKILL.md 所在目录,只收锚点目录下的所有 file
	anchorDir := path.Dir(skillMDPath)
	if anchorDir == "." {
		anchorDir = ""
	}

	// 收所有 file(后面按锚点过滤,只留锚点目录下的)
	type entry struct {
		pathInZip string
		data      []byte
	}
	var entries []entry
	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}
		name := strings.TrimPrefix(f.Name, "./")
		parts := strings.SplitN(name, "/", 2)
		if len(parts) < 2 {
			continue
		}
		repoInternalPath := parts[1]
		// 文件必须在锚点目录下(否则跳过,避免把整仓库都装进来)
		if anchorDir != "" && !strings.HasPrefix(repoInternalPath, anchorDir+"/") {
			continue
		}
		rc, oerr := f.Open()
		if oerr != nil {
			continue
		}
		data, rerr := io.ReadAll(io.LimitReader(rc, 1<<20))
		rc.Close()
		if rerr != nil {
			continue
		}
		entries = append(entries, entry{pathInZip: repoInternalPath, data: data})
	}

	files := make([]skilladapter.File, 0, len(entries))
	var skillMDContent string
	for _, e := range entries {
		rel := e.pathInZip
		if anchorDir != "" {
			stripped := strings.TrimPrefix(e.pathInZip, anchorDir+"/")
			if stripped != e.pathInZip {
				rel = stripped
			}
		}
		// 跳过 zip 里其它非锚点目录的文件(避免把整仓库都装进来)
		if anchorDir != "" {
			// 文件必须在 anchorDir 下,否则跳过
			if !strings.HasPrefix(e.pathInZip, anchorDir+"/") {
				continue
			}
		}
		// 跳过资源分叉
		if rel == "" || strings.HasPrefix(rel, "__MACOSX/") || strings.HasSuffix(rel, "/") {
			continue
		}
		if e.pathInZip == skillMDPath {
			skillMDContent = string(e.data)
			continue
		}
		files = append(files, skilladapter.File{Path: rel, Content: string(e.data)})
	}
	if skillMDContent == "" {
		return nil, fmt.Errorf("download: SKILL.md content empty")
	}
	can, perr := skilladapter.ParseSkillMD(skillMDContent)
	if perr != nil {
		return nil, fmt.Errorf("%w: parse SKILL.md: %v", skillmarket.ErrRemoteFetchFail, perr)
	}
	if can.Manifest.Name == "" {
		can.Manifest.Name = lastSegment(skillPath)
	}
	if can.Manifest.Author == "" {
		can.Manifest.Author = owner // 2026-07-09 改:用 owner 当 author(比写死 "GitHub" 准)
	}
	// SKILL.md 放 files[0](惯例),其它按 zip 顺序
	finalFiles := make([]skilladapter.File, 0, len(files)+1)
	finalFiles = append(finalFiles, skilladapter.File{Path: "SKILL.md", Content: skillMDContent})
	finalFiles = append(finalFiles, files...)
	can.Files = finalFiles
	return can, nil
}

// splitRemoteID 拆 "owner/repo@skill-path" → (owner/repo, skill-path)。
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

// lastSegment 取路径末段(用作 skill name 兜底)。
func lastSegment(p string) string {
	p = strings.Trim(p, "/")
	if i := strings.LastIndex(p, "/"); i >= 0 {
		return p[i+1:]
	}
	return p
}

// isRateLimited 2026-07-09 增:同 skillssh adapter 的识别规则。
func isRateLimited(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	if strings.Contains(msg, "status 429") || strings.Contains(msg, "status 403") {
		return true
	}
	lower := strings.ToLower(msg)
	return strings.Contains(lower, "rate limit") ||
		strings.Contains(lower, "too many requests") ||
		strings.Contains(lower, "api rate limit exceeded")
}

func (a *Adapter) fetchBody(ctx context.Context, u string) (string, error) {
	return httpx.GetJSONWithUA(ctx, a.httpClient, u)
}

func init() {
	skillmarket.Register(New())
}