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
//
// 2026-07-18 改:root-SKILL.md 仓库支持。
// 修复 https://github.com/Vi7QY/screenwriter-skill 这种 SKILL.md 在根目录的仓库
// 报"branch master not found"的根因:旧版 anchorPrefix 硬拼 skillPath + "/",
// ROOT 场景会让 tree 过滤空、触发 branch fallback、把无关的 master 404 误报上来。
// 新版:
//   1) 加 isRootSkill(skillPath, repoName) 判定 ROOT 语义
//   2) 加 branchNotFoundError sentinel,Download 循环只对"真分支不存在"走 fallback
//   3) fetchTreePaths / downloadFromTree / parseZipball 在 ROOT 时把锚点置 ""
package github

import (
	"archive/zip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"ginp-api/internal/skilladapter"
	"ginp-api/internal/skillmarket"
	"ginp-api/pkg/httpx"

	"github.com/go-git/go-git/v5/plumbing/transport"
)

const (
	// 2026-07-09:GitHub 源官方站点(github.com),用于「前往官网」按钮
	defaultSourceHomepage = "https://github.com"
	// 2026-07-09:兜底 author(实际从 remoteID 拆 owner 更准)
	defaultAuthor = "GitHub"
	// 2026-07-10 改:zipball 缓存目录(每次下载一个独立 zip,装完清)
	zipCacheDir = "/tmp/skillbox-github-zips"
	// 2026-07-10:zipball 清理阈值(避免 /tmp 撑爆,5 分钟前的全清)
	zipCleanupThreshold = 5 * time.Minute
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
//
// 2026-07-10 调:timeout 60s。raw.githubusercontent.com 单文件 ~300ms,
	// Trees API ~1.2s,极端慢网络 + 12 文件并发仍 <30s,60s 留余量。
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

// Download 走 GitHub Trees API + raw 并发下载(2026-07-10 重写,关键)。
//
// 历史(用户偏好记录):
//   - 2026-07-09 早起走 raw.githubusercontent.com 单文件 URL,只能下 SKILL.md
//   - 中期切 codeload.github.com /zip/refs/heads/{branch} 拉 zipball,能下全部附属
//   - 中后期切 go-git PlainClone 走 smart HTTPS,认为更通用
//   - 2026-07-10 实测 go-git POST git-upload-pack 在用户网络下 90s EOF,
//     codeload zipball 3.8MB 完整下载要 30s+
//
// 当前方案(实测 3s 端到端):
//   1) api.github.com/repos/{owner}/{repo}/git/trees/{branch}?recursive=1
//      → 1.2s 拿全仓库文件树(元数据,不含文件内容)
//   2) 过滤出 skillPath 前缀的 blob 条目(SKILL.md + scripts/ + references/ 等)
//   3) 并发拉 raw.githubusercontent.com/{owner}/{repo}/{branch}/{path}
//      → 每个 ~300ms,6 并发下 12 个文件 ~2s
//   4) SKILL.md 走 ParseSkillMD 出 Manifest,其它作 files
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

	// file:// 测试模式:留给将来 hook(目前测试用 parseZipball 路径)
	if strings.HasPrefix(baseURL, "file://") {
		zipPath := strings.TrimPrefix(baseURL, "file://")
		return parseZipball(zipPath, owner, repoName, skillPath, remoteID)
	}

	// 2026-07-10 改:支持 main / master 自动 fallback。
	// anthropics/skills 这种只有 main,有些老仓库只有 master。
	//
	// 2026-07-18 改(关键):fallback 只对"真分支不存在"(branchNotFoundError)
	// 触发。anchor 错(ROOT 仓库 skillPath 被当子目录走)、parse 错、网络 5xx 等
	// 都直接返,不浪费一次 master 请求,也不会把 master 404 误报成"ROOT 仓库
	// 下载失败"的根因(参见 Vi7QY/screenwriter-skill 误报 master not found 修复)。
	branches := []string{"main", "master"}
	var lastErr error
	for _, branch := range branches {
		if cerr := ctx.Err(); cerr != nil {
			if lastErr != nil {
				return nil, fmt.Errorf("%w: %v (ctx cancelled: %v)", skillmarket.ErrRemoteFetchFail, lastErr, cerr)
			}
			return nil, fmt.Errorf("%w: ctx cancelled before branch %s: %v", skillmarket.ErrRemoteFetchFail, branch, cerr)
		}

		can, err := a.downloadFromTree(ctx, owner, repoName, branch, skillPath, remoteID)
		if err == nil {
			return can, nil
		}
		lastErr = err
		// rate-limit 立即终止(避免无效重试)
		if isRateLimitedErr(err) {
			return nil, fmt.Errorf("%w: GitHub rate limited on branch %s", skillmarket.ErrRemoteFetchFail, branch)
		}
		// 2026-07-18 改:仅 branchNotFoundError 触发 fallback;其他错(anchor 空、
		// SKILL.md 缺失、parse 错、网络错)直接返,不再吞掉真相去 fallback master。
		var bne *branchNotFoundError
		if !errors.As(err, &bne) {
			return nil, fmt.Errorf("%w: %v", skillmarket.ErrRemoteFetchFail, err)
		}
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("no branch matched")
	}
	return nil, fmt.Errorf("%w: %v", skillmarket.ErrRemoteFetchFail, lastErr)
}

// downloadFromTree 走 Trees API + raw 并发,完整流程。
func (a *Adapter) downloadFromTree(ctx context.Context, owner, repoName, branch, skillPath, remoteID string) (*skilladapter.Canonical, error) {
	// 1. 拿 tree 元数据
	paths, err := a.fetchTreePaths(ctx, owner, repoName, branch, skillPath)
	if err != nil {
		return nil, err
	}

	// 2. 锚点路径前缀(用于过滤 tree 里的所有 blob)
	// 2026-07-18 改:root SKILL.md 仓库(例 Vi7QY/screenwriter-skill@screenwriter-skill,
	// SKILL.md 在 repo 根)锚点为空字符串,所有 blob 都收。
	anchorPrefix := strings.Trim(skillPath, "/") + "/"
	if isRootSkill(skillPath, repoName) {
		anchorPrefix = ""
	}
	skillMDP := strings.Trim(skillPath, "/")
	if !isRootSkill(skillPath, repoName) {
		// 子目录场景:锚点下的 SKILL.md = "{skillPath}/SKILL.md"
		skillMDP = strings.Trim(skillPath, "/") + "/SKILL.md"
	} else {
		// ROOT 场景:整个 repo 根目录下任意 SKILL.md 都算(通常只有 1 个)
		skillMDP = "SKILL.md"
	}

	// 3. 并发下载所有 raw 文件
	type fileResult struct {
		path    string
		content string
		err     error
	}
	results := make([]fileResult, len(paths))
	sem := make(chan struct{}, 6) // 6 并发
	var wg sync.WaitGroup
	for i, p := range paths {
		wg.Add(1)
		go func(i int, p string) {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				results[i] = fileResult{path: p, err: ctx.Err()}
				return
			}
			content, err := a.fetchRawFile(ctx, owner, repoName, branch, p)
			results[i] = fileResult{path: p, content: content, err: err}
		}(i, p)
	}
	wg.Wait()

	// 4. 收集结果,SKILL.md 单独处理
	//
	// 2026-07-18 改:ROOT 场景下 anchorPrefix=="",SKILL.md 不带任何前缀,匹配
	// 时直接用 "SKILL.md"。rel 也直接用 r.path,不要 TrimPrefix 空字符串(空
	// TrimPrefix 会把路径原本的子目录关系误展平,虽然 ROOT 仓库通常无子目录)。
	var skillMD string
	files := make([]skilladapter.File, 0, len(paths))
	for _, r := range results {
		if r.err != nil {
			continue // 单文件失败不影响整体
		}
		if r.path == skillMDP {
			skillMD = r.content
			continue
		}
		var rel string
		if anchorPrefix == "" {
			rel = r.path
		} else {
			rel = strings.TrimPrefix(r.path, anchorPrefix)
		}
		files = append(files, skilladapter.File{
			Path:    filepath.ToSlash(rel),
			Content: r.content,
		})
	}
	if skillMD == "" {
		return nil, fmt.Errorf("download: SKILL.md missing at %s", skillMDP)
	}

	can, perr := skilladapter.ParseSkillMD(skillMD)
	if perr != nil {
		return nil, fmt.Errorf("%w: parse SKILL.md: %v", skillmarket.ErrRemoteFetchFail, perr)
	}
	if can.Manifest.Name == "" {
		can.Manifest.Name = lastSegment(skillPath)
	}
	if can.Manifest.Author == "" {
		can.Manifest.Author = owner
	}
	// SKILL.md 放 files[0](惯例)
	allFiles := make([]skilladapter.File, 0, len(files)+1)
	allFiles = append(allFiles, skilladapter.File{Path: "SKILL.md", Content: skillMD})
	allFiles = append(allFiles, files...)
	can.Files = allFiles
	return can, nil
}

// fetchTreePaths 调 GitHub Trees API,过滤出 skillPath 前缀的所有 blob 路径。
func (a *Adapter) fetchTreePaths(ctx context.Context, owner, repoName, branch, skillPath string) ([]string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/git/trees/%s?recursive=1", owner, repoName, branch)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", httpx.UserAgent)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("tree api: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		// 2026-07-18 改:返 sentinel 让 Download 主循环判断走 fallback,不要被误报成
		// 上层"下载失败"的根因。
		return nil, &branchNotFoundError{branch: branch, owner: owner, repoName: repoName}
	}
	if resp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("status 403: GitHub API rate limited")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("status %d: %s", resp.StatusCode, body)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, fmt.Errorf("read tree: %w", err)
	}

	var treeResp struct {
		Tree []struct {
			Path string `json:"path"`
			Type string `json:"type"`
		} `json:"tree"`
		Truncated bool `json:"truncated"`
	}
	if err := json.Unmarshal(body, &treeResp); err != nil {
		return nil, fmt.Errorf("parse tree: %w", err)
	}
	if treeResp.Truncated {
		return nil, fmt.Errorf("tree truncated (repo too large, %d+ entries)", len(treeResp.Tree))
	}

	// 2026-07-18 改:root skill 仓库(skillPath == repoName)锚点为空,所有 blob 都算
	// skill 文件;非 root 场景保留原"锚点目录前缀"过滤。
	anchorPrefix := strings.Trim(skillPath, "/") + "/"
	if isRootSkill(skillPath, repoName) {
		anchorPrefix = ""
	}
	paths := make([]string, 0, 8)
	for _, e := range treeResp.Tree {
		if e.Type != "blob" {
			continue
		}
		if anchorPrefix != "" && !strings.HasPrefix(e.Path, anchorPrefix) {
			continue
		}
		paths = append(paths, e.Path)
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("no files in tree (skillPath=%q, root=%v)", skillPath, isRootSkill(skillPath, repoName))
	}
	return paths, nil
}

// fetchRawFile 单 GET 拉 raw.githubusercontent.com 文件内容。
func (a *Adapter) fetchRawFile(ctx context.Context, owner, repoName, branch, path string) (string, error) {
	url := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/%s", owner, repoName, branch, path)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", httpx.UserAgent)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch raw: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("status %d: %s", resp.StatusCode, path)
	}
	// 单文件 4MB cap,够大多数 skill 附属文件
	const cap = 4 << 20
	b, err := io.ReadAll(io.LimitReader(resp.Body, cap))
	if err != nil {
		return "", fmt.Errorf("read raw: %w", err)
	}
	return string(b), nil
}

// parseZipball 从 zipball 文件里找 SKILL.md + 收锚点目录下文件。
//
// 2026-07-10 重写:从 clone 目录扫描改为 zipball 流式解压。
// zipball 顶层目录形如 "{owner}-{repo}-{sha}/",锚点路径 = 顶层目录 + skillPath。
//
// 2026-07-18 改:root skill 仓库(例 Vi7QY/screenwriter-skill@screenwriter-skill,
// 即 SKILL.md 在 repo 根)走另一种定位:wantSuffix="SKILL.md",wantAnchorPrefix=
// 顶层目录 + "/"(所有顶层目录下的文件都算 skill 自带)。
func parseZipball(zipPath, owner, repo, skillPath, remoteID string) (*skilladapter.Canonical, error) {
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, fmt.Errorf("open zip: %w", err)
	}
	defer zr.Close()

	rootSkill := isRootSkill(skillPath, repo)
	// 锚点后缀:子目录场景 skillPath 形如 "skills/pdf" → "skills/pdf/SKILL.md";
	// root 场景只有 "SKILL.md"。
	wantSuffix := "SKILL.md"
	if !rootSkill {
		wantSuffix = strings.Trim(skillPath, "/") + "/SKILL.md"
	}
	// 锚点目录前缀:在 zip 内顶层目录下的锚点路径,所有 file 都要在这个前缀下
	wantAnchorPrefix := ""

	// 找 SKILL.md(在 zip 内任意匹配 suffix 的 entry)
	var skillMDEntry *zip.File
	for _, f := range zr.File {
		// 跳过目录条目
		if strings.HasSuffix(f.Name, "/") {
			continue
		}
		// 文件名形如 "anthropics-skills-{sha}/skills/pdf/SKILL.md"
		// 拆分顶层目录(包含 sha 的那层)
		idx := strings.Index(f.Name, "/")
		if idx < 0 {
			continue
		}
		// 顶层 + skillPath 之后才是锚点
		rel := f.Name[idx+1:]
		if rel == wantSuffix {
			skillMDEntry = f
			if rootSkill {
				// ROOT:锚点目录前缀 = 顶层目录 + "/"(整个顶层目录下全收)
				wantAnchorPrefix = f.Name[:idx+1]
			} else {
				// 子目录:锚点目录前缀 = 顶层目录 + skillPath + "/"
				wantAnchorPrefix = f.Name[:idx+1+len(wantSuffix)-len("/SKILL.md")]
			}
			break
		}
	}

	if skillMDEntry == nil {
		return nil, fmt.Errorf("download: SKILL.md not found at %q in zipball", wantSuffix)
	}

	// 读 SKILL.md
	rc, err := skillMDEntry.Open()
	if err != nil {
		return nil, fmt.Errorf("open SKILL.md: %w", err)
	}
	skillMDBytes, err := io.ReadAll(rc)
	rc.Close()
	if err != nil {
		return nil, fmt.Errorf("read SKILL.md: %w", err)
	}

	can, perr := skilladapter.ParseSkillMD(string(skillMDBytes))
	if perr != nil {
		return nil, fmt.Errorf("%w: parse SKILL.md: %v", skillmarket.ErrRemoteFetchFail, perr)
	}

	// 收锚点目录下所有 file
	files := make([]skilladapter.File, 0, 8)
	for _, f := range zr.File {
		if strings.HasSuffix(f.Name, "/") {
			continue
		}
		if !strings.HasPrefix(f.Name, wantAnchorPrefix) {
			continue
		}
		// 相对路径 = f.Name 去掉 wantAnchorPrefix 前缀
		rel := strings.TrimPrefix(f.Name, wantAnchorPrefix)
		// 去掉前导 "/" 让路径以 SKILL.md / scripts/foo.py 形式存
		rel = strings.TrimPrefix(rel, "/")
		// SKILL.md 已作 manifest,不重复
		if rel == "SKILL.md" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			continue
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			continue
		}
		files = append(files, skilladapter.File{
			Path:    filepath.ToSlash(rel),
			Content: string(data),
		})
	}

	if can.Manifest.Name == "" {
		can.Manifest.Name = lastSegment(skillPath)
	}
	if can.Manifest.Author == "" {
		can.Manifest.Author = owner
	}

	// SKILL.md 放 files[0](惯例)
	allFiles := make([]skilladapter.File, 0, len(files)+1)
	allFiles = append(allFiles, skilladapter.File{Path: "SKILL.md", Content: string(skillMDBytes)})
	allFiles = append(allFiles, files...)
	can.Files = allFiles
	return can, nil
}

// cleanupOldZipFiles 清理超过阈值的旧 zipball 文件。
// 防止 /tmp/skillbox-github-zips 撑爆(用户装过 100 个 skill 就 100 个 zip)。
func cleanupOldZipFiles() {
	entries, err := os.ReadDir(zipCacheDir)
	if err != nil {
		return // 目录不存在是正常
	}
	now := time.Now()
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if now.Sub(info.ModTime()) > zipCleanupThreshold {
			_ = os.Remove(filepath.Join(zipCacheDir, e.Name()))
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

// branchNotFoundError 2026-07-18 增:哨兵错误,标识"指定分支不存在"。
//
// GitHub API 在拿不到指定 branch 的 tree 时返 404,Download 主循环要用 sentinel
// 区分"分支真不存在(继续 fallback)"和"其他错(直接返)",否则会把 anchor 错、网络
// 错等无关错误诱导到下一个分支、最后用 master 的 404 误报成"下载失败根因"。
type branchNotFoundError struct {
	branch   string
	owner    string
	repoName string
}

func (e *branchNotFoundError) Error() string {
	return fmt.Sprintf("branch %q not found on %s/%s", e.branch, e.owner, e.repoName)
}

// isRootSkill 2026-07-18 增:判定"根目录单文件仓库"语义。
//
// resolver 在 URL 末段是 SKILL.md 且路径里无其他子段时,会把 skill 设成 repo 名
// (例 Vi7QY/screenwriter-skill@screenwriter-skill),此时 SKILL.md 实际在 repo
// 根目录。Download 走 tree API 时不应再用 skillPath 作"锚点目录前缀"过滤,
// 否则会把 SKILL.md 自身过滤掉、抛出"no files under"误判。
//
// 规则:skillPath 去掉前后空白与斜杠后与 repoName 完全相等。
// 容错 TrimSpace 是防御:splitRemoteID 自身不会产出含空格路径,但日后若 resolver
// 演进传入 trimmed 值,这里不应误判。
func isRootSkill(skillPath, repoName string) bool {
	return strings.TrimSpace(strings.Trim(skillPath, "/")) == strings.TrimSpace(repoName)
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

func init() {
	skillmarket.Register(New())
}