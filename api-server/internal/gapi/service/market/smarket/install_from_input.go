package smarket

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"ginp-api/internal/gapi/entity"
	"ginp-api/internal/gapi/service/skill/sskill"
	"ginp-api/internal/skilladapter"
	"ginp-api/internal/skillmarket"
	mmarketskill "ginp-api/internal/gapi/model/skillbox/mmarketskill"
	mmarketsource "ginp-api/internal/gapi/model/skillbox/mmarketsource"
	"ginp-api/pkg/where"
)

// 2026-07-09 增:同名 skill 已存在错误,前端用它判 409 并弹覆盖确认。
var ErrSkillAlreadyExists = errors.New("market: skill already exists")

// InstallFromInputInput 用户输入框 → 下载入参(2026-07-09 增)。
//
// 与 PullInput 的差别:不要求 caller 提前知道 sourceID + remoteID,而是把"用户原文"
// (slug / 详情页 URL / owner/repo@skill / GitHub URL)交给 service 解析。
type InstallFromInputInput struct {
	// SourceHint 前端传下来的当前 tab source_type("skillhub" / "skillssh" / "github");
	// 空 = auto(由 service 解析 input 推断);非空时只接受该 source 的 URL。
	SourceHint string
	Input      string
	Scope      string // global(默认) / project
	ProjectID  uint
	// 2026-07-09 增:冲突解决模式
	//   - "" 或 "prompt":service 内部检查同名,存在时直接返 ErrSkillAlreadyExists
	//     (前端弹 Modal 询问「覆盖/另存为/取消」,选覆盖时再调一次传 "overwrite")
	//   - "overwrite":跳过冲突检查,直接覆盖同名 skill
	//   - "rename":跳过冲突检查,自动加 -2 / -3 后缀避免冲突
	ConflictMode string
	// 2026-07-09 增:ConflictMode=rename 时使用的目标名(可选,空 = 自动生成)
	RenameTo string
}

// InstallFromInputResult 落盘结果(2026-07-09 增)。
type InstallFromInputResult struct {
	SourceType    string `json:"source_type"`
	SourceName    string `json:"source_name"`
	RemoteID      string `json:"remote_id"`
	ResolvedURL   string `json:"resolved_url,omitempty"`
	SkillName     string `json:"skill_name"`
	SkillVersion  string `json:"skill_version"`
	Scope         string `json:"scope"`
	MarketSkillID uint   `json:"market_skill_id,omitempty"`
	// 2026-07-09 增:实际写入的分组路径(空 = 根);前端用来刷新 tree 时定位
	// 跳转首页后会自动展开到该 group。
	GroupPath string `json:"group_path,omitempty"`
	// 2026-07-09 增:同名已存在时返 409,前端弹「覆盖/另存为/取消」对话框
	ConflictExistingVersion string `json:"conflict_existing_version,omitempty"`
	ConflictExistingPath   string `json:"conflict_existing_path,omitempty"`
}

// InstallFromInput 用户输入框 → 解析 → 下载 → 落 store(2026-07-09 增)。
//
// 流程:
//   1) ResolveInstallInput 解析 input → (source_type, remote_id)
//   2) 找 source / 或注册一个新 source(skillhub / skillssh 默认已 seed)
//   3) Orchestrator.DownloadFromSource 拿 canonical
//   4) sskill.Service.Create 落 store(skillstore DB)
//
// 错误:
//   - ErrInvalidInput 解析失败(400)
//   - ErrSourceNotFound 源未注册(404)
//   - ErrPullFailed 下载 / 写盘失败(500)
//
// 不做 apply:本接口只负责把 skill 装到本地 store,apply 到具体工具(scope)
// 由用户在首页 Onboarding / Skill detail 触发,跟现有 Pull / PullV2 行为一致。
func (s *Service) InstallFromInput(ctx context.Context, in *InstallFromInputInput) (*InstallFromInputResult, error) {
	if in == nil {
		return nil, fmt.Errorf("%w: nil input", ErrPullFailed)
	}
	input := strings.TrimSpace(in.Input)
	if input == "" {
		return nil, fmt.Errorf("%w: input 必填", ErrInvalidInput)
	}

	// 1) 解析 input
	resolved, err := ResolveInstallInput(input, in.SourceHint)
	if err != nil {
		return nil, err
	}

	// 2) scope 校验
	scope := strings.ToLower(strings.TrimSpace(in.Scope))
	if scope == "" {
		scope = skilladapter.ScopeGlobal
	}
	if scope != skilladapter.ScopeGlobal && scope != skilladapter.ScopeProject {
		return nil, fmt.Errorf("%w: scope 必须是 global / project", ErrPullFailed)
	}
	if scope == skilladapter.ScopeProject && in.ProjectID == 0 {
		return nil, fmt.Errorf("%w: project scope 需要 project_id", ErrPullFailed)
	}

	// 3) 找 source(skillhub / skillssh 默认已 seed)
	src, err := s.findOrCreateSourceByType(resolved.SourceType)
	if err != nil {
		return nil, err
	}

	// 4) 拿缓存里的 market_skill(若已 refresh 过则有,无也不影响下载)
	conds := append(where.New(mmarketskill.FieldSourceID, "=", src.ID).Conditions(),
		where.New(mmarketskill.FieldRemoteID, "=", resolved.RemoteID).Conditions()...)
	row, _ := s.skillModel().FindOne(conds)

	// 5) 下载
	can, err := s.orchestrator().DownloadFromSource(ctx, src.ID, resolved.RemoteID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPullFailed, err)
	}
	if can == nil {
		return nil, fmt.Errorf("%w: empty canonical for %s", ErrPullFailed, resolved.RemoteID)
	}

	// 6) 写盘:走 sskill.Service.Create
	if s.skillSvcFactory == nil {
		return nil, fmt.Errorf("%w: skill service factory not wired", ErrPullFailed)
	}
	ssvc, ferr := s.skillSvcFactory()
	if ferr != nil {
		return nil, fmt.Errorf("%w: %v", ErrPullFailed, ferr)
	}

	// 7) 2026-07-09 增:同名 skill 冲突检查(用户要求"下载前提示是否覆盖")
	//    - ConflictMode = "" / "prompt":先查 store,存在 → 返 ErrSkillAlreadyExists +
	//      现有 version + path(让前端弹「覆盖/另存为/取消」)
	//    - ConflictMode = "overwrite":跳过检查,Create 内部会覆盖同名 skill
	//    - ConflictMode = "rename":自动给 name 加 -2 / -3 后缀
	can.Manifest.Name = skilladapter.NormalizeName(can.Manifest.Name)
	if can.Manifest.Name == "" {
		return nil, fmt.Errorf("%w: empty skill name after normalize", ErrPullFailed)
	}
	conflictMode := strings.ToLower(strings.TrimSpace(in.ConflictMode))
	if conflictMode == "" || conflictMode == "prompt" {
		if ssvc != nil {
			existing, eerr := ssvc.Get(can.Manifest.Name)
			if eerr == nil && existing != nil {
				// 命中同名,返 409 + 现有信息(让前端弹确认)
				existingPath := existing.Manifest.Name
				if existing.Manifest.GroupPath != "" {
					existingPath = existing.Manifest.GroupPath + "/" + existing.Manifest.Name
				}
				return &InstallFromInputResult{
					SourceType:              resolved.SourceType,
					SourceName:              resolved.SourceName,
					RemoteID:                resolved.RemoteID,
					ResolvedURL:             resolved.ResolvedURL,
					SkillName:               can.Manifest.Name,
					SkillVersion:            can.Manifest.Version,
					Scope:                   scope,
					ConflictExistingVersion: existing.Manifest.Version,
					ConflictExistingPath:   existingPath,
				}, fmt.Errorf("%w: %s", ErrSkillAlreadyExists, can.Manifest.Name)
			}
		}
	} else if conflictMode == "rename" {
		// 2026-07-09 增:另存为 — 自动 -2/-3 后缀或用户指定
		target := skilladapter.NormalizeName(in.RenameTo)
		if target != "" {
			can.Manifest.Name = target
		} else {
			// 自动找空闲名
			base := can.Manifest.Name
			for i := 2; i <= 99; i++ {
				cand := fmt.Sprintf("%s-%d", base, i)
				if ssvc == nil {
					break
				}
				if _, eerr := ssvc.Get(cand); eerr != nil {
					can.Manifest.Name = cand
					break
				}
			}
		}
	}
	// overwrite 模式:不查冲突,直接走 ssvc.Create(底层 store.Create 会覆盖)

	// 8) 补 manifest(以三方元数据为底,canonical 为真)
	if row != nil {
		can.Manifest.Author = firstNonEmpty(can.Manifest.Author, row.Author)
		if can.Manifest.License == "" {
			can.Manifest.License = row.License
		}
	}
	can.Manifest.Source = firstNonEmpty(can.Manifest.Source, "market")
	can.Manifest.SourceRef = firstNonEmpty(can.Manifest.SourceRef, fmt.Sprintf("%s:%s", src.Name, resolved.RemoteID))

	// 9) 2026-07-09 增:按 source_type 自动写 GroupPath(用户要求"按来源放入对应分组")。
	//    - skillhub → 分组 "skillhub"
	//    - skillssh → 分组 "skills-sh"(走 NormalizeGroupName,跟 sskill.CreateGroup 对齐)
	//    - github   → 分组按 owner 划分(anthropics/skills → 分组 "anthropics"),
	//      同一个 owner 的多个 skill 归到同一组下,便于浏览
	//    分组目录不存在时自动 CreateGroup(走 store 建物理目录),
	//    跟 PullV2 一样,前端不需要先建组再装。
	//
	// 2026-07-09 修(关键 bug):早期用 "skills.sh" 字面量,CreateGroup → NormalizeGroupName
	// 把 "." 折叠成 "-",实际建的是 StoreRoot/skills-sh/,但 Manifest.GroupPath 仍是
	// "skills.sh",resolveSkillDir 拼出 StoreRoot/skills.sh/pdf/,lock 路径
	// StoreRoot/skills.sh/pdf.lock 的父目录 StoreRoot/skills.sh/ 不存在 → ENOENT。
	// 修法:groupPath 走 NormalizeGroupName 规范化,CreateGroup 跟 Manifest.GroupPath
	// 用同一个规范化值,保证物理目录跟 GroupPath 字段对齐。
	groupPath := normalizeGroupPathForMarket(deriveGroupPath(resolved.SourceType, resolved.RemoteID))
	can.Manifest.GroupPath = groupPath
	if ssvc != nil && groupPath != "" {
		if gerr := ssvc.CreateGroup(groupPath); gerr != nil {
			return nil, fmt.Errorf("%w: create group %q: %v", ErrPullFailed, groupPath, gerr)
		}
	}

	version := firstNonEmpty(can.Manifest.Version, "0.1.0")
	if row != nil {
		version = firstNonEmpty(version, row.Version, "0.1.0")
	}

	created, cerr := ssvc.Create(&sskill.WriteInput{
		Scope:     scope,
		ProjectID: in.ProjectID,
		Name:      can.Manifest.Name,
		Version:   version,
		Manifest:  can.Manifest,
		Files:     can.Files,
	})
	if cerr != nil {
		return nil, fmt.Errorf("%w: %v", ErrPullFailed, cerr)
	}

	out := &InstallFromInputResult{
		SourceType:   resolved.SourceType,
		SourceName:   resolved.SourceName,
		RemoteID:     resolved.RemoteID,
		ResolvedURL:  resolved.ResolvedURL,
		SkillName:    created.Manifest.Name,
		SkillVersion: created.Manifest.Version,
		Scope:        scope,
		GroupPath:    groupPath, // 2026-07-09 增:返回前端用于跳转 skills tab 后的高亮
	}
	if row != nil {
		out.MarketSkillID = row.ID
	}
	return out, nil
}

// defaultGroupPathFor 2026-07-09 增:按 source_type 派生默认分组路径。
//
// 用户要求"按来源放入对应分组":
//   - skillhub → "skillhub"
//   - skillssh → "skills-sh"(跟源 DisplayName 对齐,经 NormalizeGroupName 规范化)
//   - github   → ""(由 deriveGroupPath 按 owner 动态生成,不在这里写死)
//
// 未知 source 返空字符串(不强制分组,让 skill 落到 store 根)。
//
// 注意:返回值会经过 normalizeGroupPathForMarket 二次规范化,
// 把 "." 折叠成 "-" 等(跟 sskill.CreateGroup 走 NormalizeGroupName 一致),
// 避免物理目录跟 Manifest.GroupPath 不一致导致 lock 路径 ENOENT。
func defaultGroupPathFor(sourceType string) string {
	switch sourceType {
	case skillmarket.SourceSkillhub, "skillhub":
		// 2026-07-10 改:SourceSkillhub 改名 "skillhub-cn",分组目录跟 type 同名;
		// 老 sourceType="skillhub" 走 DB 旧记录,分组仍沿用 "skillhub" 目录(避免已下载的 skill 重新落盘)
		if sourceType == "skillhub" {
			return "skillhub"
		}
		return "skillhub-cn"
	case skillmarket.SourceSkillsSH:
		return "skills.sh"
	default:
		return ""
	}
}

// deriveGroupPath 2026-07-09 增:根据 source_type + remoteID 动态推导分组。
//
//   - skillhub → 返 "skillhub-cn"(2026-07-10 改:对齐 UI / 磁盘目录)
//   - skillssh → 返 "skills.sh"(固定)
//   - github   → 从 remoteID="owner/repo@skill-path" 拆出 owner 当分组,
//     同一个 owner 多个 skill 归到同组(anthropics/skills@pdf + anthropics/skills@pdf
//     都进 "anthropics" 组),便于浏览;owner 不在远程 ID 里时回退到 "github"
func deriveGroupPath(sourceType, remoteID string) string {
	switch sourceType {
	case skillmarket.SourceSkillhub, "skillhub":
		// 2026-07-10 改:同 defaultGroupPathFor,新老 type 用各自分组名
		if sourceType == "skillhub" {
			return "skillhub"
		}
		return "skillhub-cn"
	case skillmarket.SourceSkillsSH:
		return "skills.sh"
	case skillmarket.SourceGitHub:
		owner, _ := splitOwnerFromRemote(remoteID)
		if owner == "" {
			return "github"
		}
		return owner
	default:
		return ""
	}
}

// splitOwnerFromRemote 2026-07-09 增:从各种 remoteID 格式里拆 owner:
//
//   - github:   "owner/repo@skill-path" → "owner"
//   - skillssh: "owner/repo@skill"     → "owner"
//   - skillhub: "slug"                 → ""(固定分组,无 owner)
//
// 拆失败返空字符串,调用方决定 fallback。
func splitOwnerFromRemote(remoteID string) (string, bool) {
	at := strings.LastIndex(remoteID, "@")
	head := remoteID
	if at > 0 {
		head = remoteID[:at]
	}
	slash := strings.Index(head, "/")
	if slash <= 0 || slash >= len(head)-1 {
		return "", false
	}
	return head[:slash], true
}

// normalizeGroupPathForMarket 2026-07-09 增:把 defaultGroupPathFor 返回的字面量
// 走 skilladapter.NormalizeGroupName,跟 sskill.CreateGroup 内部保持一致。
//
// 历史上没规范化时,groupPath="skills.sh" CreateGroup 实际建 "skills-sh" 目录,
// 但 Manifest.GroupPath 仍是 "skills.sh",resolveSkillDir 拼出 StoreRoot/skills.sh/{name}
// lock 路径 StoreRoot/skills.sh/{name}.lock 的父目录 StoreRoot/skills.sh/ 不存在
// → O_CREATE 失败,返 ENOENT。
func normalizeGroupPathForMarket(p string) string {
	if p == "" {
		return ""
	}
	return skilladapter.NormalizeGroupName(p)
}

// findOrCreateSourceByType 按 source_type 找已有 source,没有就 seed 一条(2026-07-09 增)。
//
// skillhub / skillssh 默认由 EnsureDefaultSources 在 ListSources 时 seed;
// 但 InstallFromInput 路径不一定先经过 ListSources,这里兜底自 seed。
func (s *Service) findOrCreateSourceByType(sourceType string) (*entity.MarketSource, error) {
	list, _, err := s.sourceModel().FindList(nil, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: list sources: %v", ErrSourceNotFound, err)
	}
	for _, e := range list {
		if e != nil && e.Type == sourceType && e.Enabled {
			return e, nil
		}
	}
	// 兜底:注册一个临时 enabled source(seed 流程尚未跑过)
	name := sourceType
	switch sourceType {
	case skillmarket.SourceSkillhub:
		name = "skillhub-cn" // 2026-07-10 改:跟 type 同名,作为 market_sources.name 默认值
	case skillmarket.SourceSkillsSH:
		name = "skills.sh"
	}
	created, cerr := s.sourceModel().Create(&entity.MarketSource{
		Name:    name,
		Type:    sourceType,
		Enabled: true,
	})
	if cerr != nil {
		// 并发 race:另一个请求可能同时 seed,FindOneByName 兜底再找一次
		conds := append(where.New(mmarketsource.FieldName, "=", name).Conditions(),
			where.New(mmarketsource.FieldType, "=", sourceType).Conditions()...)
		if again, aerr := s.sourceModel().FindOne(conds); aerr == nil && again != nil {
			return again, nil
		}
		return nil, fmt.Errorf("%w: create source %s: %v", ErrSourceNotFound, sourceType, cerr)
	}
	return created, nil
}

// firstNonEmpty 已在 PullV2 上下文存在(此文件外);这里别名引用避免重复声明。
//
// 本文件其它地方使用 firstNonEmpty 时仍走 package-level 函数。
var _ = errors.Is
var _ = normalizeGroupPathForMarket // 2026-07-09 增:导出前小写函数,测试用占位(避免编译器报 unused)