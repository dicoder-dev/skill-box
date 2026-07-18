package smarket

// 2026-07-09 增:用户输入框 → 后端下载 解析层。
//
// 用户在 MarketView 输入框粘贴 skill 名称 / 详情页 URL,本文件把原文解析成
// (source_type, remote_id) 元组,供 InstallFromInput 走 Orchestrator.DownloadFromSource。
//
// 支持格式:
//   - https://skillhub.cn/skills/{slug}                → (skillhub, slug)
//   - https://api.skillhub.cn/skills/{slug}             → (skillhub, slug)(同源)
//   - https://skillhub.cn/skill/{slug}                  → (skillhub, slug)(兜底)
//   - https://www.skills.sh/{owner}/{repo}/{skill}      → (skillssh, owner/repo@skill)
//   - https://skills.sh/{owner}/{repo}/{skill}          → (skillssh, owner/repo@skill)
//   - https://github.com/{owner}/{repo}/.../{skill}/SKILL.md → (skillssh, owner/repo@skill)
//   - https://raw.githubusercontent.com/{owner}/{repo}/{branch}/{skill}/SKILL.md → (skillssh, owner/repo@skill)
//   - 纯 slug(在 skillhub tab)                          → (skillhub, slug)
//   - owner/repo@skill(在 skillssh tab)                 → (skillssh, owner/repo@skill)
//
// 不支持:
//   - 任意 URL(非 skillhub / skills.sh / github 域名) → ErrInvalidInput
//   - 缺少 owner 或 repo 或 skill → ErrInvalidInput
//   - raw URL 但 path 不以 SKILL.md 结尾 → ErrInvalidInput

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"ginp-api/internal/skillmarket"
)

// ErrInvalidInput 用户输入无法识别成任何已知格式(2026-07-09 增)。
//
// 任何 source_id + input 解析失败都 wrap 这个 error;controller 层据此返 400。
var ErrInvalidInput = errors.New("market: invalid input")

// ResolvedInput 解析结果(2026-07-09 增)。
//
// - SourceType 必填,值为 skillmarket.SourceSkillhub / skillmarket.SourceSkillsSH;
// - RemoteID   必填,符合 adapter Download 接口的 remoteID 格式;
// - ResolvedURL 后端实际访问的 URL(若 input 是 URL),便于前端展示和日志追踪。
type ResolvedInput struct {
	SourceType  string `json:"source_type"`
	SourceName  string `json:"source_name"` // DisplayName(UI 文案)
	RemoteID    string `json:"remote_id"`
	ResolvedURL string `json:"resolved_url,omitempty"`
}

// ResolveInstallInput 把用户原文解析成 (source_type, remote_id)(2026-07-09 增)。
//
// 2026-07-09 改:各 source 仅支持自己的 URL 形态(用户要求「统一仅支持一个来源」)
//   - skillhub:仅 https://skillhub.cn/skills/{slug} 详情页 URL
//   - skills.sh:仅 https://skills.sh/{owner}/{repo}/{skill} 详情页 URL
//   - github:仅 https://github.com/.../blob/.../SKILL.md 详情页 URL
//   - 不再支持 raw.githubusercontent.com(那是 zipball 的事,不是 skill 入口)
//   - 不再支持纯 slug(用户要求"必须粘 URL")
//
// 参数:
//   - input      用户原文(已 TrimSpace,必须含 URL scheme)
//   - sourceHint 前端传下来的当前 tab source_id("skillhub" / "skillssh" / "github");
//     缺省 = auto(由 URL 域名推断);非空时只接受该 source 的 URL。
//
// 失败:返 ErrInvalidInput 的 wrap 错误,UI 上展示 err.Error() 即可。
func ResolveInstallInput(input, sourceHint string) (*ResolvedInput, error) {
	raw := strings.TrimSpace(input)
	if raw == "" {
		return nil, fmt.Errorf("%w: 输入为空", ErrInvalidInput)
	}

	hint := strings.ToLower(strings.TrimSpace(sourceHint))

	// 2026-07-09 改:不再支持非 URL 输入。所有 source 都要求用户粘详情页 URL。
	if !strings.Contains(raw, "://") {
		return nil, fmt.Errorf("%w: 必须粘贴详情页 URL(以 https:// 开头),得到 %q", ErrInvalidInput, raw)
	}

	// 解析 URL
	u, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("%w: URL 解析失败 %q: %v", ErrInvalidInput, raw, err)
	}
	if u.Scheme == "" || u.Host == "" {
		return nil, fmt.Errorf("%w: URL 缺少 scheme/host %q", ErrInvalidInput, raw)
	}

	host := strings.ToLower(u.Host)
	host = strings.TrimPrefix(host, "www.")
	hostNoAPIPrefix := strings.TrimPrefix(host, "api.")

	// 2026-07-09 改:按 hint 收窄域名(用户要求"统一仅支持一个来源")
	switch hint {
	case skillmarket.SourceSkillhub:
		// skillhub tab:只接受 skillhub 域名
		if host != "skillhub.cn" && hostNoAPIPrefix != "skillhub.cn" {
			return nil, fmt.Errorf("%w: 当前是 SkillHub tab,只支持 https://skillhub.cn/skills/{{slug}} URL,得到 %q", ErrInvalidInput, raw)
		}
		return resolveSkillhubURL(u, raw)

	case skillmarket.SourceSkillsSH:
		// skills.sh tab:只接受 skills.sh 域名
		if host != "skills.sh" {
			return nil, fmt.Errorf("%w: 当前是 Skills.sh tab,只支持 https://skills.sh/{{owner}}/{{repo}}/{{skill}} URL,得到 %q", ErrInvalidInput, raw)
		}
		return resolveSkillsSHURL(u, raw)

	case skillmarket.SourceGitHub:
		// github tab:只接受 github.com 域名
		if host != "github.com" {
			return nil, fmt.Errorf("%w: 当前是 GitHub tab,只支持 https://github.com/{{owner}}/{{repo}}/blob/{{branch}}/{{path}}/SKILL.md URL,得到 %q", ErrInvalidInput, raw)
		}
		return resolveGitHubTreeURL(u, raw)

	case "":
		// auto:按域名分发(URL 必须带明确域名)
		switch {
		case host == "skillhub.cn" || hostNoAPIPrefix == "skillhub.cn":
			return resolveSkillhubURL(u, raw)
		case host == "skills.sh":
			return resolveSkillsSHURL(u, raw)
		case host == "github.com":
			return resolveGitHubTreeURL(u, raw)
		default:
			return nil, fmt.Errorf("%w: 不支持的 URL 域名 %q(目前支持 skillhub.cn / skills.sh / github.com 详情页 URL)", ErrInvalidInput, host)
		}

	default:
		return nil, fmt.Errorf("%w: 未知 source_hint %q", ErrInvalidInput, hint)
	}
}

// resolveSkillhubURL 从 skillhub URL 提取 slug(2026-07-09 增)。
//
// 支持:
//   - /skills/{slug}
//   - /skill/{slug}(兜底)
//   - 末尾空 slug / 路径不匹配 → ErrInvalidInput
func resolveSkillhubURL(u *url.URL, raw string) (*ResolvedInput, error) {
	parts := splitPathParts(u.Path)
	if len(parts) == 0 {
		return nil, fmt.Errorf("%w: skillhub URL 缺少路径 %q", ErrInvalidInput, raw)
	}
	var slug string
	// 模式:/skills/{slug}
	if parts[0] == "skills" || parts[0] == "skill" {
		if len(parts) < 2 {
			return nil, fmt.Errorf("%w: skillhub URL 路径缺少 slug %q", ErrInvalidInput, raw)
		}
		slug = parts[1]
	} else {
		// 兜底:把最后一段当 slug(兼容 /{slug} 或 /pskoett/{slug} 等奇怪路径)
		slug = parts[len(parts)-1]
	}
	slug = sanitizeSlug(slug)
	if slug == "" {
		return nil, fmt.Errorf("%w: skillhub slug 非法 %q", ErrInvalidInput, raw)
	}
	return &ResolvedInput{
		SourceType:  skillmarket.SourceSkillhub,
		SourceName:  "SkillHub",
		RemoteID:    slug,
		ResolvedURL: "https://skillhub.cn/skills/" + slug,
	}, nil
}

// resolveSkillsSHURL 从 skills.sh URL 拆 (owner/repo, skill)(2026-07-09 增)。
//
// 支持:
//   - /{owner}/{repo}/{skill}
//   - /{owner}/{repo}/{skill}/
//   - 末尾 query / fragment 忽略
func resolveSkillsSHURL(u *url.URL, raw string) (*ResolvedInput, error) {
	parts := splitPathParts(u.Path)
	if len(parts) < 3 {
		return nil, fmt.Errorf("%w: skills.sh URL 必须形如 /{owner}/{repo}/{skill},得到 %q", ErrInvalidInput, raw)
	}
	owner := parts[0]
	repo := parts[1]
	skill := parts[2]
	if !validOwnerRepo(owner) || !validOwnerRepo(repo) {
		return nil, fmt.Errorf("%w: skills.sh owner/repo 非法 %q", ErrInvalidInput, raw)
	}
	if sanitizeSlug(skill) == "" {
		return nil, fmt.Errorf("%w: skills.sh skill 名称非法 %q", ErrInvalidInput, raw)
	}
	return &ResolvedInput{
		SourceType:  skillmarket.SourceSkillsSH,
		SourceName:  "skills.sh",
		RemoteID:    owner + "/" + repo + "@" + skill,
		ResolvedURL: "https://skills.sh/" + owner + "/" + repo + "/" + skill,
	}, nil
}

// resolveGitHubTreeURL 从 GitHub tree/blob URL 推断 (owner/repo, skill)(2026-07-09 增)。
//
// 支持:
//   - github.com/{owner}/{repo}                          → skill = repo (root,2026-07-18 增)
//   - github.com/{owner}/{repo}/                         → skill = repo (root,同上)
//   - github.com/{owner}/{repo}/blob/{branch}/{path}/SKILL.md → skill = {完整目录路径}
//   - github.com/{owner}/{repo}/tree/{branch}/{path}    → skill = {完整 path}
//   - github.com/{owner}/{repo}/raw/{branch}/{path}/SKILL.md → 同 blob
//
// 推断规则:
//   - 2 段形式({owner}/{repo} 或 +尾斜杠)视为仓库主页,默认派生 root 仓库路径,
//     让 github adapter 的 isRootSkill 接管(参见 github.go 修复)
//   - 5+ 段形式(path 末段必须是 SKILL.md):sub-skill 取 path 完整父目录
//   - path 末段是 SKILL.md 且 len(rest) < 2 → skill = repo(root 仓库)
//   - path 末段是 README → 报错
//   - 其他 >3 段且 mode 不在 blob/tree/raw → 报错
func resolveGitHubTreeURL(u *url.URL, raw string) (*ResolvedInput, error) {
	parts := splitPathParts(u.Path)
	owner := ""
	repo := ""
	switch len(parts) {
	case 2:
		// 2026-07-18 增:repo 主页 URL(只粘 https://github.com/{owner}/{repo})
		// 直接派生 root 仓库形态,后续由 github adapter 的 isRootSkill 接管。
		owner = parts[0]
		repo = parts[1]
		if !validOwnerRepo(owner) || !validOwnerRepo(repo) {
			return nil, fmt.Errorf("%w: GitHub owner/repo 非法 %q", ErrInvalidInput, raw)
		}
		// sanity check:GitHub home 长 repo 不是真仓库(404 兜底交给 adapter),
		// 这里只粗校验 owner / repo 字符不空。
		return &ResolvedInput{
			SourceType:  skillmarket.SourceGitHub,
			SourceName:  "GitHub",
			RemoteID:    owner + "/" + repo + "@" + repo,
			ResolvedURL: raw,
		}, nil
	case 0, 1:
		// 只有域名 / 只有一段(owner 没有 repo):无法定位仓库
		return nil, fmt.Errorf("%w: GitHub URL 路径过短,得到 %q(预期形如 https://github.com/{{owner}}/{{repo}})", ErrInvalidInput, raw)
	default:
		// 多段:走原 blob/tree/raw 路径
		if len(parts) < 5 {
			return nil, fmt.Errorf("%w: GitHub URL 路径过短,得到 %q", ErrInvalidInput, raw)
		}
		owner = parts[0]
		repo = parts[1]
		mode := parts[2]
		_ = parts[3] // branch,保留解析但不直接使用
		if mode != "blob" && mode != "tree" && mode != "raw" {
			return nil, fmt.Errorf("%w: GitHub URL 第 3 段必须是 blob/tree/raw,得到 %q", ErrInvalidInput, mode)
		}
		if !validOwnerRepo(owner) || !validOwnerRepo(repo) {
			return nil, fmt.Errorf("%w: GitHub owner/repo 非法 %q", ErrInvalidInput, raw)
		}
		// 剩余 path = skill 目录或 skill 文件
		rest := parts[4:]
		if len(rest) == 0 {
			return nil, fmt.Errorf("%w: GitHub URL 路径缺少 SKILL.md,得到 %q", ErrInvalidInput, raw)
		}
		last := rest[len(rest)-1]
		var skill string
		switch {
		case strings.EqualFold(last, "SKILL.md"):
			// 文件层:取完整目录路径(github adapter 需要全路径,不止末段)
			// 例 blob/main/skills/pdf/SKILL.md → skill = "skills/pdf"
			if len(rest) < 2 {
				// 根目录的 SKILL.md,skill = repo(单文件仓库)
				skill = repo
			} else {
				// 拼接 rest 中除了末段 SKILL.md 之外的所有段
				skill = strings.Join(rest[:len(rest)-1], "/")
			}
		case strings.EqualFold(last, "README.md"):
			// README 不算 skill 入口,报错
			return nil, fmt.Errorf("%w: GitHub URL 指向 README.md,不是 SKILL.md %q", ErrInvalidInput, raw)
		default:
			// 末段是目录(tree URL 常见),用完整 path 作 skill(去掉末段斜杠)
			skill = strings.Join(rest, "/")
		}
		if sanitizeSlug(skill) == "" {
			return nil, fmt.Errorf("%w: GitHub skill 名称非法 %q", ErrInvalidInput, raw)
		}
		// 原始 URL 当 ResolvedURL,前端展示/日志用。
		return &ResolvedInput{
			SourceType:  skillmarket.SourceGitHub,
			SourceName:  "GitHub",
			RemoteID:    owner + "/" + repo + "@" + skill,
			ResolvedURL: raw,
		}, nil
	}
}

// splitOwnerRepoAt 拆 "owner/repo@skill" → (owner/repo, skill)(2026-07-09 增)。
//
// 与 skillssh 内部 splitRemoteID 等价,提到公共位置避免 adapter 包反向依赖。
func splitOwnerRepoAt(s string) (string, string, bool) {
	at := strings.LastIndex(s, "@")
	if at <= 0 || at == len(s)-1 {
		return "", "", false
	}
	repo := s[:at]
	name := s[at+1:]
	if !strings.Contains(repo, "/") {
		return "", "", false
	}
	return repo, name, true
}

// splitPathParts 把 URL path 按 / 拆并去掉空段(2026-07-09 增)。
func splitPathParts(p string) []string {
	p = strings.Trim(p, "/")
	if p == "" {
		return nil
	}
	parts := strings.Split(p, "/")
	out := make([]string, 0, len(parts))
	for _, seg := range parts {
		if seg == "" {
			continue
		}
		out = append(out, seg)
	}
	return out
}

// sanitizeSlug 简单 slug 校验(2026-07-09 增;2026-07-10 放宽允许 /)。
//
// 允许字母/数字/-/_/.//;空 → 返空。
// 不做完整 RFC 校验,只防呆(空 / 含空格 / 含 : 等明显异常)。
//
// 2026-07-10 改:允许 / + 用于 GitHub 嵌套 skill path(如 "skills/pdf")。
// GitHub 仓库 SKILL.md 通常在子目录(skills/pdf/SKILL.md),resolver 现在
// 返回完整路径而不是末段,这样 adapter 才能定位 tree 锚点。
func sanitizeSlug(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '-' || r == '_' || r == '.' || r == '/':
		default:
			return ""
		}
	}
	return s
}

// validOwnerRepo 校验 GitHub owner/repo 段(2026-07-09 增)。
//
// GitHub 规则:字母/数字/-/_/. ,长度 1-100,不能以 -/_/. 开头或结尾。
// 实现简化版:同 sanitizeSlug,长度 ≤ 100。
func validOwnerRepo(s string) bool {
	if s == "" || len(s) > 100 {
		return false
	}
	if s[0] == '-' || s[0] == '_' || s[0] == '.' {
		return false
	}
	if s[len(s)-1] == '-' || s[len(s)-1] == '_' || s[len(s)-1] == '.' {
		return false
	}
	return sanitizeSlug(s) != ""
}