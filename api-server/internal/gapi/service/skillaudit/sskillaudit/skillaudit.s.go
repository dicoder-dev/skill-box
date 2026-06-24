// Package sskillaudit 提供 Tag / Diff / Rollback 的业务层封装。
//
// 2026-06-24 改造:用 (scope, name) 定位 skill,不再走 mskill 表(已弃用)。
package sskillaudit

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"ginp-api/internal/gapi/entity"
	"ginp-api/internal/gapi/service/audit/saudit"
	"ginp-api/internal/gapi/service/skill/sskill"
	mskillfile "ginp-api/internal/gapi/model/skillbox/mskillfile"
	mskillfilesnapshot "ginp-api/internal/gapi/model/skillbox/mskillfilesnapshot"
	mskilltag "ginp-api/internal/gapi/model/skillbox/mskilltag"
	"ginp-api/internal/skillaudit"
	"ginp-api/internal/skilladapter"
	"ginp-api/internal/skillstore"
	"ginp-api/pkg/where"

	"gorm.io/gorm"
)

// 业务错误(sentinel)。
var (
	ErrSkillNotFound = errors.New("skillaudit: skill not found")
	ErrTagNotFound   = errors.New("skillaudit: tag not found")
	ErrInvalidTag    = errors.New("skillaudit: invalid tag")
	ErrEmptyFiles    = errors.New("skillaudit: no files to tag")
)

// Service 业务服务。
type Service struct {
	dbWrite *gorm.DB
	dbRead  *gorm.DB
	store   *skillstore.Store
}

func New(dbWrite, dbRead *gorm.DB, store *skillstore.Store) *Service {
	return &Service{dbWrite: dbWrite, dbRead: dbRead, store: store}
}

func (s *Service) fileModel() *mskillfile.Model {
	return mskillfile.NewModel(s.dbWrite, s.dbRead)
}
func (s *Service) tagModel() *mskilltag.Model {
	return mskilltag.NewModel(s.dbWrite, s.dbRead)
}
func (s *Service) snapModel() *mskillfilesnapshot.Model {
	return mskillfilesnapshot.NewModel(s.dbWrite, s.dbRead)
}

// audit 内部 helper。
func (s *Service) audit(action string, targetID uint, payload any) {
	if s.dbWrite == nil {
		return
	}
	payloadStr := ""
	if payload != nil {
		if b, err := json.Marshal(payload); err == nil {
			payloadStr = string(b)
		}
	}
	_, _ = saudit.New(s.dbWrite, s.dbRead).Write(saudit.WriteInput{
		Actor:      "system",
		Action:     action,
		TargetType: "tag",
		TargetID:   targetID,
		Payload:    payloadStr,
	})
}

// CreateTagInput 打 tag 入参(2026-06-24:用 scope+name 定位)。
type CreateTagInput struct {
	Scope     string `json:"scope"`
	ProjectID uint   `json:"project_id"`
	Name      string `json:"name"`
	Tag       string `json:"tag"`
	Message   string `json:"message"`
}

// CreateTagOutput 打 tag 出参。
type CreateTagOutput struct {
	TagID     uint      `json:"tag_id"`
	Tag       string    `json:"tag"`
	Message   string    `json:"message"`
	Files     int       `json:"files"`
	CreatedAt time.Time `json:"created_at"`
}

// CreateTag 给一个 skill 打 tag:从 store 读当前所有文件 → skillaudit.BuildTag → 落 DB。
func (s *Service) CreateTag(in *CreateTagInput) (*CreateTagOutput, error) {
	if in == nil || strings.TrimSpace(in.Name) == "" {
		return nil, fmt.Errorf("%w: name required", ErrSkillNotFound)
	}
	if err := skillaudit.ValidateTag(in.Tag); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidTag, err)
	}
	// 从 store 读 canonical
	c, err := s.store.Load(in.Name)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrSkillNotFound, err)
	}
	// 转 FileSnap
	files := make([]skillaudit.FileSnap, 0, len(c.Files))
	for _, f := range c.Files {
		files = append(files, skillaudit.FileSnap{Path: f.Path, Content: f.Content})
	}
	// 用 skillaudit.BuildTag 算 hash(以 name 当 skillID 用,只为生成稳定 hash)
	built, err := skillaudit.BuildTag(skillaudit.TagSnapshot{
		SkillID: hashNameToID(in.Name),
		Tag:     in.Tag,
		Message: in.Message,
		Files:   files,
	})
	if err != nil {
		return nil, err
	}
	// 写 tag 行
	tagRow := &entity.SkillTag{
		Scope:     in.Scope,
		ProjectID: in.ProjectID,
		Name:      in.Name,
		Tag:       in.Tag,
		Message:   in.Message,
		IsImplicit: false,
	}
	if _, err := s.tagModel().Create(tagRow); err != nil {
		s.audit("tag_create_failed", 0, map[string]any{
			"name":    in.Name,
			"tag":     in.Tag,
			"message": in.Message,
			"error":   err.Error(),
		})
		return nil, fmt.Errorf("skillaudit: create tag: %w", err)
	}
	// 写文件快照
	for _, f := range built.Files {
		if _, err := s.snapModel().Create(&entity.SkillFileSnapshot{
			SkillTagID:   tagRow.ID,
			Scope:        in.Scope,
			ProjectID:    in.ProjectID,
			Name:         in.Name,
			Path:         f.Path,
			Content:      f.Content,
			ContentHash:  f.ContentHash,
		}); err != nil {
			s.audit("tag_create_failed", 0, map[string]any{
				"name":    in.Name,
				"tag":     in.Tag,
				"message": in.Message,
				"error":   err.Error(),
			})
			return nil, fmt.Errorf("skillaudit: create file snapshot: %w", err)
		}
	}
	s.audit("tag_create", 0, map[string]any{
		"name":    in.Name,
		"tag_id":  tagRow.ID,
		"tag":     in.Tag,
		"message": in.Message,
		"files":   len(built.Files),
	})
	return &CreateTagOutput{
		TagID:     tagRow.ID,
		Tag:       in.Tag,
		Message:   in.Message,
		Files:     len(built.Files),
		CreatedAt: tagRow.CreatedAt,
	}, nil
}

// ListTags 列出某 skill 的所有 tag(2026-06-24:用 scope+name 定位)。
func (s *Service) ListTags(scope, name string) ([]*entity.SkillTag, error) {
	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("%w: name required", ErrSkillNotFound)
	}
	conds := where.New(mskilltag.FieldScope, "=", scope).Conditions()
	conds = append(conds, where.New(mskilltag.FieldName, "=", name).Conditions()...)
	extra := &where.Extra{PageNum: 1, PageSize: 1000, OrderByColumn: mskilltag.FieldCreatedAt, OrderByDesc: true}
	items, _, err := s.tagModel().FindList(conds, extra)
	if err != nil {
		return nil, err
	}
	return items, nil
}

// DeleteTag 删除一个 tag(包括它的 file_snapshots)。
func (s *Service) DeleteTag(tagID uint) error {
	if tagID == 0 {
		return ErrTagNotFound
	}
	tagRow, err := s.tagModel().FindOneById(tagID)
	if err != nil {
		return fmt.Errorf("%w: id=%d", ErrTagNotFound, tagID)
	}
	snapConds := where.New(mskillfilesnapshot.FieldSkillTagID, "=", tagID).Conditions()
	snaps, _, err := s.snapModel().FindList(snapConds, &where.Extra{PageNum: 1, PageSize: 10000})
	if err != nil {
		return fmt.Errorf("skillaudit: list snaps: %w", err)
	}
	for _, sn := range snaps {
		if err := s.snapModel().DeleteById(sn.ID); err != nil {
			return fmt.Errorf("skillaudit: delete snap: %w", err)
		}
	}
	if err := s.tagModel().DeleteById(tagID); err != nil {
		return fmt.Errorf("skillaudit: delete tag: %w", err)
	}
	s.audit("tag_delete", 0, map[string]any{
		"name":       tagRow.Name,
		"tag_id":     tagID,
		"tag":        tagRow.Tag,
		"is_implicit": tagRow.IsImplicit,
	})
	return nil
}

// DiffInput diff 入参(2026-06-24:用 scope+name 定位)。
type DiffInput struct {
	Scope      string `json:"scope"`
	Name       string `json:"name"`
	LeftTagID  uint   `json:"left_tag_id"`
	RightTagID uint   `json:"right_tag_id"`
}

// DiffOutput diff 出参。
type DiffOutput struct {
	Files     []skillaudit.FileDiff `json:"files"`
	Added     int                   `json:"added"`
	Removed   int                   `json:"removed"`
	Modified  int                   `json:"modified"`
	Unchanged int                   `json:"unchanged"`
}

// Diff 拿两个视图的文件做 diff。
func (s *Service) Diff(in *DiffInput) (*DiffOutput, error) {
	if in == nil || strings.TrimSpace(in.Name) == "" {
		return nil, fmt.Errorf("%w: name required", ErrSkillNotFound)
	}
	left, err := s.loadView(in.Name, in.LeftTagID)
	if err != nil {
		return nil, err
	}
	right, err := s.loadView(in.Name, in.RightTagID)
	if err != nil {
		return nil, err
	}
	files := skillaudit.Diff(left, right)
	out := &DiffOutput{Files: files}
	for _, f := range files {
		switch f.Kind {
		case "added":
			out.Added++
		case "removed":
			out.Removed++
		case "modified":
			out.Modified++
		case "unchanged":
			out.Unchanged++
		}
	}
	return out, nil
}

// loadView 0 = "current"(走 store.Load),>0 = 该 tag 的 file_snapshots。
func (s *Service) loadView(name string, tagID uint) ([]skillaudit.FileSnap, error) {
	if tagID == 0 {
		c, err := s.store.Load(name)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrSkillNotFound, err)
		}
		out := make([]skillaudit.FileSnap, 0, len(c.Files))
		for _, f := range c.Files {
			out = append(out, skillaudit.FileSnap{Path: f.Path, Content: f.Content})
		}
		return out, nil
	}
	tag, err := s.tagModel().FindOneById(tagID)
	if err != nil {
		return nil, fmt.Errorf("%w: id=%d", ErrTagNotFound, tagID)
	}
	if tag.Name != name {
		return nil, fmt.Errorf("skillaudit: tag %d belongs to skill %q, not %q", tagID, tag.Name, name)
	}
	conds := where.New(mskillfilesnapshot.FieldSkillTagID, "=", tagID).Conditions()
	snaps, _, err := s.snapModel().FindList(conds, &where.Extra{PageNum: 1, PageSize: 10000})
	if err != nil {
		return nil, fmt.Errorf("skillaudit: list snaps: %w", err)
	}
	out := make([]skillaudit.FileSnap, 0, len(snaps))
	for _, sn := range snaps {
		out = append(out, skillaudit.FileSnap{Path: sn.Path, Content: sn.Content})
	}
	return out, nil
}

// RollbackInput 回滚入参。
type RollbackInput struct {
	TagID uint `json:"tag_id"`
}

// RollbackOutput 回滚出参。
type RollbackOutput struct {
	PreRollbackTagID uint   `json:"pre_rollback_tag_id"`
	PreRollbackTag   string `json:"pre_rollback_tag"`
	FilesRestored    int    `json:"files_restored"`
}

// Rollback 把 skill 当前状态回滚到指定 tag 的内容。
// 流程:先打 _pre_rollback_<ts> 隐式 tag(覆盖当前状态)→ 把目标 tag 的 files 写回 store。
func (s *Service) Rollback(in *RollbackInput) (*RollbackOutput, error) {
	if in == nil || in.TagID == 0 {
		return nil, ErrTagNotFound
	}
	tag, err := s.tagModel().FindOneById(in.TagID)
	if err != nil {
		return nil, fmt.Errorf("%w: id=%d", ErrTagNotFound, in.TagID)
	}
	// 1) 隐式预回滚 tag
	preName := skillaudit.ImplicitPreRollbackTag(time.Now())
	preOut, err := s.CreateTag(&CreateTagInput{
		Scope:     tag.Scope,
		ProjectID: tag.ProjectID,
		Name:      tag.Name,
		Tag:       preName,
		Message:   fmt.Sprintf("auto pre-rollback to tag %s", tag.Tag),
	})
	if err != nil {
		s.audit("rollback_failed", 0, map[string]any{
			"name":   tag.Name,
			"tag_id": in.TagID,
			"tag":    tag.Tag,
			"stage":  "pre_tag",
			"error":  err.Error(),
		})
		return nil, fmt.Errorf("skillaudit: create pre-rollback tag: %w", err)
	}
	// 标记 is_implicit
	if err := s.tagModel().Update(
		where.New(mskilltag.FieldID, "=", preOut.TagID).Conditions(),
		&entity.SkillTag{IsImplicit: true, Message: fmt.Sprintf("auto pre-rollback to tag %s", tag.Tag)},
		mskilltag.FieldIsImplicit, mskilltag.FieldMessage,
	); err != nil {
		s.audit("rollback_failed", 0, map[string]any{
			"name":   tag.Name,
			"tag_id": in.TagID,
			"tag":    tag.Tag,
			"stage":  "mark_implicit",
			"error":  err.Error(),
		})
		return nil, fmt.Errorf("skillaudit: mark implicit: %w", err)
	}
	// 2) 读目标 tag 的文件
	target, err := s.loadView(tag.Name, in.TagID)
	if err != nil {
		return nil, err
	}
	// 3) 重建 manifest:用当前 skill 的 manifest + target 的 file 列表
	cur, err := s.store.Load(tag.Name)
	if err != nil {
		s.audit("rollback_failed", 0, map[string]any{
			"name":   tag.Name,
			"tag_id": in.TagID,
			"tag":    tag.Tag,
			"stage":  "load_current",
			"error":  err.Error(),
		})
		return nil, fmt.Errorf("skillaudit: load current: %w", err)
	}
	mfst := cur.Manifest
	files := make([]skilladapter.File, 0, len(target))
	for _, f := range target {
		files = append(files, skilladapter.File{Path: f.Path, Content: f.Content})
	}
	if _, err := sskill.New(s.store).Update(tag.Name, &sskill.WriteInput{
		Manifest: mfst,
		Files:    files,
	}); err != nil {
		s.audit("rollback_failed", 0, map[string]any{
			"name":   tag.Name,
			"tag_id": in.TagID,
			"tag":    tag.Tag,
			"stage":  "replace_files",
			"error":  err.Error(),
		})
		return nil, fmt.Errorf("skillaudit: replace files: %w", err)
	}
	s.audit("rollback", 0, map[string]any{
		"name":              tag.Name,
		"tag_id":            in.TagID,
		"tag":               tag.Tag,
		"pre_rollback_tag":  preName,
		"pre_rollback_tag_id": preOut.TagID,
		"files_restored":    len(target),
	})
	return &RollbackOutput{
		PreRollbackTagID: preOut.TagID,
		PreRollbackTag:   preName,
		FilesRestored:    len(target),
	}, nil
}

// hashNameToID 给 name 算一个稳定 uint 当临时 SkillID(只为 skillaudit.BuildTag 内部用)。
// 注意:仅是 hash,不是 DB 里的真 id;CreateTag 内部不依赖这个值。
func hashNameToID(name string) uint {
	var h uint = 2166136261
	for i := 0; i < len(name); i++ {
		h ^= uint(name[i])
		h *= 16777619
	}
	// 避开 0
	if h == 0 {
		h = 1
	}
	return h
}
