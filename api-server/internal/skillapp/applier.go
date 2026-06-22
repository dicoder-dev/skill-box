package skillapp

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"ginp-api/internal/skilladapter"
)

// Applier 负责把 canonical 落到目标目录 + 返回 pre-snapshot(由 service 写 DB)。
//
// 实现侧约束:
//   - 必须先 snapshot 目标目录状态,再 apply;失败立刻用 snapshot 回滚
//   - 写 SkillApply 行由 service 层负责(本包只负责落盘 + 拍照)
//   - v1 文件落盘用 os.WriteFile(文本),二进制 P1 补
type Applier struct {
	registry *skilladapter.Registry
	now      func() time.Time // 测试用
}

// NewApplier 构造 Applier;registry=nil 时用默认全局。
func NewApplier(registry *skilladapter.Registry) *Applier {
	return &Applier{registry: registry, now: time.Now}
}

// NewApplierWithClock 测试用 - 注入 clock。
func NewApplierWithClock(registry *skilladapter.Registry, now func() time.Time) *Applier {
	a := NewApplier(registry)
	if now != nil {
		a.now = now
	}
	return a
}

// ApplyResult 单 tool 的 apply 结果(含 pre-snapshot,服务层据此落 DB)。
type ApplyResult struct {
	Tool        string       `json:"tool"`
	TargetPath  string       `json:"target_path"`
	Status      string       `json:"status"` // applied / failed
	ApplyID     uint         `json:"apply_id,omitempty"` // service 写完 DB 后回填
	PreSnapshot *PreSnapshot `json:"-"`      // 不进 JSON,只走 service 内部
	Error       string       `json:"error,omitempty"`
	StartedAt   time.Time    `json:"started_at"`
	FinishedAt  time.Time    `json:"finished_at"`
}

// ApplyOne 把 canonical 落到 in.Tools[0](单 tool);批量由 caller 循环调。
//
// 失败语义:即使 apply 失败,PreSnapshot 也会带回(部分文件可能已落),
// service 写 DB 时 status=failed + 仍存 pre_snapshot,方便排查。
func (a *Applier) ApplyOne(in ApplyInput) (*ApplyResult, error) {
	if in.Canonical == nil {
		return nil, fmt.Errorf("%w: canonical nil", ErrEmptySkill)
	}
	if len(in.Canonical.Files) == 0 {
		return nil, fmt.Errorf("%w: name=%s", ErrEmptyFiles, in.Canonical.Manifest.Name)
	}
	if len(in.Tools) == 0 {
		return nil, ErrEmptyTools
	}
	scope := strings.ToLower(strings.TrimSpace(in.Scope))
	if scope == "" {
		scope = skilladapter.ScopeGlobal
	}
	if scope != skilladapter.ScopeGlobal && scope != skilladapter.ScopeProject {
		return nil, fmt.Errorf("skillapp: invalid scope %q", in.Scope)
	}
	toolID := in.Tools[0]
	ad, ok := a.registry.Get(toolID)
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrToolNotFound, toolID)
	}
	targetDir, err := resolveTargetDir(ad, in.Canonical, scope, in.ProjectID)
	if err != nil {
		return nil, err
	}
	pre, snapErr := snapshotDir(targetDir)
	if snapErr != nil {
		return nil, fmt.Errorf("skillapp: snapshot %s: %w", targetDir, snapErr)
	}
	started := a.now()
	if err := ad.Apply(*in.Canonical, targetDir); err != nil {
		_ = restoreFromSnapshot(targetDir, pre)
		finished := a.now()
		return &ApplyResult{
			Tool:        toolID,
			TargetPath:  targetDir,
			Status:      StatusFailed,
			PreSnapshot: pre,
			Error:       err.Error(),
			StartedAt:   started,
			FinishedAt:  finished,
		}, fmt.Errorf("skillapp: apply %s to %s: %w", in.Canonical.Manifest.Name, toolID, err)
	}
	post := make([]string, 0, len(in.Canonical.Files))
	for _, f := range in.Canonical.Files {
		if f.Path == "" {
			continue
		}
		post = append(post, f.Path)
	}
	pre.PostFiles = post
	finished := a.now()
	return &ApplyResult{
		Tool:        toolID,
		TargetPath:  targetDir,
		Status:      StatusApplied,
		PreSnapshot: pre,
		StartedAt:   started,
		FinishedAt:  finished,
	}, nil
}

// resolveTargetDir 把 (tool + scope + project_id + name) 拼到具体目录。
// v1 简化:global scope 直接用 adapter 的第一个 DiscoverPaths;project scope 把相对路径
// 拼到 home/.skillbox/projects/<id>/(占位实现,Step 9 配合 sproject 用真项目根)。
func resolveTargetDir(ad skilladapter.Adapter, c *skilladapter.Canonical, scope string, projectID uint) (string, error) {
	paths, err := ad.DiscoverPaths(scope)
	if err != nil {
		return "", err
	}
	if len(paths) == 0 {
		return "", fmt.Errorf("skillapp: tool %s has no paths for scope %s", ad.ToolID(), scope)
	}
	parent := paths[0]
	if !filepath.IsAbs(parent) {
		if projectID == 0 {
			return "", fmt.Errorf("skillapp: scope=project 需要 project_id")
		}
		homedir, _ := os.UserHomeDir()
		if homedir == "" {
			return "", fmt.Errorf("skillapp: cannot resolve home for relative project path")
		}
		parent = filepath.Join(homedir, ".skillbox", "projects", fmt.Sprintf("%d", projectID), parent)
	}
	localName := ad.LocalName(*c)
	return filepath.Join(parent, localName), nil
}

// snapshotDir 拍目录快照:列出所有文本文件 + 读内容。v1 假设都是文本。
func snapshotDir(dir string) (*PreSnapshot, error) {
	snap := &PreSnapshot{PostFiles: nil}
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return snap, nil
		}
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("skillapp: target %s is not a dir", dir)
	}
	snap.TargetExisted = true
	walkErr := filepath.Walk(dir, func(path string, fi os.FileInfo, werr error) error {
		if werr != nil {
			return werr
		}
		rel, _ := filepath.Rel(dir, path)
		if rel != "." {
			base := filepath.Base(rel)
			// 跳过隐藏目录(.git / .claude 等)
			if strings.HasPrefix(base, ".") && fi.IsDir() && rel != base {
				return filepath.SkipDir
			}
		}
		if fi.IsDir() {
			return nil
		}
		if fi.Size() > 4*1024*1024 {
			// 太大当 binary 跳过(PreSnapshot 不存 content)
			snap.Files = append(snap.Files, FileSnapshot{Path: rel, Existed: true})
			return nil
		}
		b, rerr := os.ReadFile(path)
		if rerr != nil {
			return rerr
		}
		snap.Files = append(snap.Files, FileSnapshot{
			Path:    rel,
			Existed: true,
			Content: string(b),
		})
		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}
	return snap, nil
}

// restoreFromSnapshot 从快照恢复目录。
// - pre 里的 file:写回原 content(覆盖 apply 写的)
// - post_files 不在 pre 里的:删除(apply 加进去的)
func restoreFromSnapshot(dir string, pre *PreSnapshot) error {
	if pre == nil {
		return nil
	}
	preSet := map[string]bool{}
	for _, f := range pre.Files {
		preSet[f.Path] = true
	}
	// 1) 删 post_files 里不在 pre 的
	for _, p := range pre.PostFiles {
		if preSet[p] {
			continue
		}
		full := filepath.Join(dir, p)
		if err := os.Remove(full); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("restore: remove %s: %w", full, err)
		}
		_ = removeEmptyParents(filepath.Dir(full), filepath.Dir(dir))
	}
	// 2) 写回 pre 里的
	for _, f := range pre.Files {
		full := filepath.Join(dir, f.Path)
		if f.Content == "" {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			return fmt.Errorf("restore: mkdir %s: %w", filepath.Dir(full), err)
		}
		if err := os.WriteFile(full, []byte(f.Content), 0o644); err != nil {
			return fmt.Errorf("restore: write %s: %w", full, err)
		}
	}
	return nil
}

// removeEmptyParents 从 leaf 往 root 删空目录,直到 stopAt 为止。
func removeEmptyParents(dir, stopAt string) error {
	for dir != stopAt && dir != filepath.Dir(dir) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return nil
		}
		if len(entries) > 0 {
			return nil
		}
		if err := os.Remove(dir); err != nil {
			return nil
		}
		dir = filepath.Dir(dir)
	}
	return nil
}

var _ = errors.New
