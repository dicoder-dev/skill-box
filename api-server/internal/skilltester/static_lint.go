package skilltester

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"ginp-api/internal/skilladapter"
)

// 静态 lint 规则常量(命名 / 版本 / 长度等)。
const (
	minNameLen     = 2
	maxNameLen     = 64
	minDescLen     = 10
	maxDescLen     = 500
	minTriggers    = 1
	maxTriggers    = 10
	maxBodyChars   = 50000 // body 上限,防止误粘贴巨型内容
	maxFileCount   = 50
	maxFileBytes   = 1 * 1024 * 1024
)

var (
	nameRE    = regexp.MustCompile(`^[a-z][a-z0-9-]{1,63}$`)
	versionRE = regexp.MustCompile(`^v?\d+\.\d+\.\d+([-+].+)?$`)
	// secret 嗅探: sk- / sk_live_ / api_key= / token= / secret=
	secretRE = regexp.MustCompile(`(?i)(sk-[A-Za-z0-9_-]{6,}|sk_live_[A-Za-z0-9_-]{6,}|api[_-]?key\s*=\s*["']?[A-Za-z0-9_-]{6,}|token\s*=\s*["']?[A-Za-z0-9_-]{6,}|secret\s*=\s*["']?[A-Za-z0-9_-]{6,})`)
	// 路径合法性:不允许 .. / 绝对 / 含 \0
	badPathRE = regexp.MustCompile(`(\.\.|^\/|\x00)`)
)

// StaticFinding 单条 lint 命中,汇总到 CheckResult.Detail。
type StaticFinding struct {
	Rule    string `json:"rule"`
	OK      bool   `json:"ok"`
	Message string `json:"message,omitempty"`
}

// StaticLintSummary 静态 lint 的详情(便于前端展示)。
type StaticLintSummary struct {
	Findings []StaticFinding `json:"findings"`
	// BodyChars / FileCount 计数
	BodyChars int `json:"body_chars"`
	FileCount int `json:"file_count"`
}

// Lint 对 canonical 做静态检查,产出单个 CheckResult(永远非 nil,失败也降级为 reported)。
func Lint(c skilladapter.Canonical) CheckResult {
	summary := StaticLintSummary{}
	add := func(rule string, ok bool, msg string) {
		summary.Findings = append(summary.Findings, StaticFinding{Rule: rule, OK: ok, Message: msg})
	}

	// manifest 基础
	m := c.Manifest
	if m.Name == "" {
		add("name_present", false, "manifest.name is empty")
	} else {
		add("name_present", true, "")
		if !nameRE.MatchString(m.Name) {
			add("name_format", false, fmt.Sprintf("name %q does not match %s", m.Name, nameRE.String()))
		} else {
			add("name_format", true, "")
		}
	}
	if m.Version == "" {
		add("version_present", false, "manifest.version is empty")
	} else if !versionRE.MatchString(m.Version) {
		add("version_format", false, fmt.Sprintf("version %q does not match semver", m.Version))
	} else {
		add("version_format", true, "")
	}
	dl := len([]rune(m.Description))
	switch {
	case dl < minDescLen:
		add("description_length", false, fmt.Sprintf("description has %d chars, need >=%d", dl, minDescLen))
	case dl > maxDescLen:
		add("description_length", false, fmt.Sprintf("description has %d chars, need <=%d", dl, maxDescLen))
	default:
		add("description_length", true, "")
	}
	// triggers 去重 + 长度
	seen := map[string]struct{}{}
	dup := false
	for _, t := range m.Triggers {
		t = strings.ToLower(strings.TrimSpace(t))
		if t == "" {
			continue
		}
		if _, ok := seen[t]; ok {
			dup = true
		}
		seen[t] = struct{}{}
	}
	switch {
	case len(seen) < minTriggers:
		add("triggers_count", false, fmt.Sprintf("triggers (deduped) = %d, need >=%d", len(seen), minTriggers))
	case len(seen) > maxTriggers:
		add("triggers_count", false, fmt.Sprintf("triggers (deduped) = %d, need <=%d", len(seen), maxTriggers))
	case dup:
		add("triggers_unique", false, "triggers have duplicates")
	default:
		add("triggers_count", true, "")
		add("triggers_unique", true, "")
	}

	// files
	if len(c.Files) == 0 {
		add("files_present", false, "canonical has no files")
	} else {
		add("files_present", true, "")
	}
	if len(c.Files) > maxFileCount {
		add("files_count", false, fmt.Sprintf("file count %d exceeds %d", len(c.Files), maxFileCount))
	}
	hasSkillMD := false
	var body string
	for _, f := range c.Files {
		if badPathRE.MatchString(f.Path) {
			add("file_path", false, fmt.Sprintf("file %q has invalid path", f.Path))
		}
		if len(f.Content) > maxFileBytes {
			add("file_size", false, fmt.Sprintf("file %q is %d bytes, exceeds %d", f.Path, len(f.Content), maxFileBytes))
		}
		if f.Path == "SKILL.md" {
			hasSkillMD = true
			body = f.Content
		}
	}
	if !hasSkillMD {
		add("skill_md_present", false, "missing SKILL.md")
	} else {
		add("skill_md_present", true, "")
	}
	if body == "" {
		add("body_present", false, "SKILL.md body is empty")
	} else {
		summary.BodyChars = len([]rune(body))
		if summary.BodyChars > maxBodyChars {
			add("body_size", false, fmt.Sprintf("body has %d chars, exceeds %d", summary.BodyChars, maxBodyChars))
		} else {
			add("body_size", true, "")
		}
	}

	// 全文 secret 扫描(对所有 file content 拼起来)
	combined := strings.Builder{}
	for _, f := range c.Files {
		combined.WriteString(f.Content)
		combined.WriteString("\n")
	}
	if m := secretRE.FindString(combined.String()); m != "" {
		add("no_secrets", false, fmt.Sprintf("found potential secret pattern: %s", truncate(m, 40)))
	} else {
		add("no_secrets", true, "")
	}

	// 汇总
	passed := true
	failed := []string{}
	for _, f := range summary.Findings {
		if !f.OK {
			passed = false
			failed = append(failed, f.Rule)
		}
	}
	summary.FileCount = len(c.Files)

	detail, _ := json.Marshal(summary)
	res := CheckResult{
		Check:   CheckStatic,
		Detail:  string(detail),
		Message: fmt.Sprintf("%d checks", len(summary.Findings)),
	}
	if passed {
		res.Status = StatusPassed
	} else {
		res.Status = StatusFailed
		res.Message = "failed: " + strings.Join(failed, ",")
	}
	return res
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
