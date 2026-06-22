// Package sskillaudit 提供 Tag / Diff / Rollback 的业务层封装。
//
// 设计要点(见 docs/project/需求规划.md 第 4.1.9 节):
//   - 打 tag = 把 skillstore 里的当前文件读出来,固化到 skill_tags + skill_file_snapshots
//   - 列 tag = 按 skill_id 查 skill_tags
//   - diff = 拿两个 tag / 或一个 tag + "current" 调 skillaudit.Diff
//   - rollback = 先打一个 _pre_rollback_<ts> 隐式 tag(覆盖当前状态),再把目标 tag 的
//     files 写回 skillstore + 更新 skill_files
package sskillaudit

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"ginp-api/internal/gapi/entity"
	"ginp-api/internal/gapi/service/audit/saudit"
	"ginp-api/internal/gapi/service/skill/sskill"
	mskill "ginp-api/internal/gapi/model/skillbox/mskill"
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

func (s *Service) skillModel() *mskill.Model {
	return mskill.NewModel(s.dbWrite, s.dbRead)
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

// CreateTagInput 打 tag 入参。
type CreateTagInput struct {
	SkillID uint   `json:"skill_id"`
	Tag     string `json:"tag"`
	Message string `json:"message"`
}

// CreateTagOutput 打 tag 出参。
type CreateTagOutput struct {
	TagID    uint      `json:"tag_id"`
	Tag      string    `json:"tag"`
	Message  string    `json:"message"`
	Files    int       `json:"files"`
	CreatedAt time.Time `json:"created_at"`
}

// CreateTag 给一个 skill 打 tag:从 skillstore 读当前所有文件 → skillaudit.BuildTag → 落 DB。
// 失败时(同 skill_id + tag 已存在)返 sentinel。
func (s *Service) CreateTag(in *CreateTagInput) (*CreateTagOutput, error) {
	if in == nil || in.SkillID == 0 {
		return nil, fmt.Errorf("%w: skill_id required", ErrSkillNotFound)
	}
	// 校验 tag
	if err := skillaudit.ValidateTag(in.Tag); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidTag, err)
	}
	// 查 skill
	row, err := s.skillModel().FindOneById(in.SkillID)
	if err != nil {
		return nil, fmt.Errorf("%w: id=%d", ErrSkillNotFound, in.SkillID)
	}
	// 读当前 canonical
	c, err := s.store.Load(row.Scope, row.Name, row.Version, row.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("skillaudit: load skill: %w", err)
	}
	// 转 FileSnap
	files := make([]skillaudit.FileSnap, 0, len(c.Files))
	for _, f := range c.Files {
		files = append(files, skillaudit.FileSnap{Path: f.Path, Content: f.Content})
	}
	// 用 skillaudit.BuildTag 算 hash
	built, err := skillaudit.BuildTag(skillaudit.TagSnapshot{
		SkillID: in.SkillID,
		Tag:     in.Tag,
		Message: in.Message,
		Files:   files,
	})
	if err != nil {
		return nil, err
	}
	// 写 tag 行
	tagRow := &entity.SkillTag{
		SkillID:    in.SkillID,
		Tag:        in.Tag,
		Message:    in.Message,
		IsImplicit: false,
	}
	if _, err := s.tagModel().Create(tagRow); err != nil {
		return nil, fmt.Errorf("skillaudit: create tag: %w", err)
	}
	// 写文件快照
	for _, f := range built.Files {
		if _, err := s.snapModel().Create(&entity.SkillFileSnapshot{
			SkillTagID:  tagRow.ID,
			Path:        f.Path,
			Content:     f.Content,
			ContentHash: f.ContentHash,
		}); err != nil {
			return nil, fmt.Errorf("skillaudit: create file snapshot: %w", err)
		}
	}
	return &CreateTagOutput{
		TagID:     tagRow.ID,
		Tag:       in.Tag,
		Message:   in.Message,
		Files:     len(built.Files),
		CreatedAt: tagRow.CreatedAt,
	}, nil
}

// ListTags 列出某 skill 的所有 tag(默认按 created_at desc)。
func (s *Service) ListTags(skillID uint) ([]*entity.SkillTag, error) {
	if skillID == 0 {
		return nil, fmt.Errorf("%w: skill_id required", ErrSkillNotFound)
	}
	conds := where.New(mskilltag.FieldSkillID, "=", skillID).Conditions()
	extra := &where.Extra{PageNum: 1, PageSize: 1000, OrderByColumn: mskilltag.FieldCreatedAt, OrderByDesc: true}
	items, _, err := s.tagModel().FindList(conds, extra)
	if err != nil {
		return nil, err
	}
	return items, nil
}

// DeleteTag 删除一个 tag(包括它的 file_snapshots)。
// 隐式 tag 也允许删(规格:可手动删除)。
func (s *Service) DeleteTag(tagID uint) error {
	if tagID == 0 {
		return ErrTagNotFound
	}
	// 先确认行存在
	if _, err := s.tagModel().FindOneById(tagID); err != nil {
		return fmt.Errorf("%w: id=%d", ErrTagNotFound, tagID)
	}
	// 先删 file_snapshots
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
	// 再删 tag 行
	if err := s.tagModel().DeleteById(tagID); err != nil {
		return fmt.Errorf("skillaudit: delete tag: %w", err)
	}
	return nil
}

// DiffInput diff 入参(支持 tag vs tag / tag vs current / current vs current)。
type DiffInput struct {
	SkillID uint `json:"skill_id"`
	// LeftTagID 0 = "current" 状态;>0 = 该 tag 的文件
	LeftTagID uint `json:"left_tag_id"`
	// RightTagID 同上
	RightTagID uint `json:"right_tag_id"`
}

// DiffOutput diff 出参。
type DiffOutput struct {
	Files []skillaudit.FileDiff `json:"files"`
	// 统计
	Added     int `json:"added"`
	Removed   int `json:"removed"`
	Modified  int `json:"modified"`
	Unchanged int `json:"unchanged"`
}

// Diff 拿两个视图的文件做 diff。
func (s *Service) Diff(in *DiffInput) (*DiffOutput, error) {
	if in == nil || in.SkillID == 0 {
		return nil, fmt.Errorf("%w: skill_id required", ErrSkillNotFound)
	}
	left, err := s.loadView(in.SkillID, in.LeftTagID)
	if err != nil {
		return nil, err
	}
	right, err := s.loadView(in.SkillID, in.RightTagID)
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

// loadView 0 = "current"(走 skillstore.Load),>0 = 该 tag 的 file_snapshots。
func (s *Service) loadView(skillID uint, tagID uint) ([]skillaudit.FileSnap, error) {
	if tagID == 0 {
		row, err := s.skillModel().FindOneById(skillID)
		if err != nil {
			return nil, fmt.Errorf("%w: id=%d", ErrSkillNotFound, skillID)
		}
		c, err := s.store.Load(row.Scope, row.Name, row.Version, row.ProjectID)
		if err != nil {
			return nil, fmt.Errorf("skillaudit: load current: %w", err)
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
	if tag.SkillID != skillID {
		return nil, fmt.Errorf("skillaudit: tag %d belongs to skill %d, not %d", tagID, tag.SkillID, skillID)
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
// 流程:先打 _pre_rollback_<ts> 隐式 tag(覆盖当前状态)→ 把目标 tag 的 files 写回 skillstore
// → 重建 entity.SkillFile 列表。
func (s *Service) Rollback(in *RollbackInput) (*RollbackOutput, error) {
	if in == nil || in.TagID == 0 {
		return nil, ErrTagNotFound
	}
	tag, err := s.tagModel().FindOneById(in.TagID)
	if err != nil {
		return nil, fmt.Errorf("%w: id=%d", ErrTagNotFound, in.TagID)
	}
	row, err := s.skillModel().FindOneById(tag.SkillID)
	if err != nil {
		return nil, fmt.Errorf("%w: id=%d", ErrSkillNotFound, tag.SkillID)
	}
	// 1) 隐式预回滚 tag
	preName := skillaudit.ImplicitPreRollbackTag(time.Now())
	preOut, err := s.CreateTag(&CreateTagInput{
		SkillID: tag.SkillID,
		Tag:     preName,
		Message: fmt.Sprintf("auto pre-rollback to tag %s", tag.Tag),
	})
	if err != nil {
		return nil, fmt.Errorf("skillaudit: create pre-rollback tag: %w", err)
	}
	// 标记 is_implicit
	if err := s.tagModel().Update(
		where.New(mskilltag.FieldID, "=", preOut.TagID).Conditions(),
		&entity.SkillTag{IsImplicit: true, Message: fmt.Sprintf("auto pre-rollback to tag %s", tag.Tag)},
		mskilltag.FieldIsImplicit, mskilltag.FieldMessage,
	); err != nil {
		return nil, fmt.Errorf("skillaudit: mark implicit: %w", err)
	}
	// 2) 读目标 tag 的文件
	target, err := s.loadView(tag.SkillID, in.TagID)
	if err != nil {
		return nil, err
	}
	// 3) 重建 manifest(用当前 skill 的 source/source_ref/version,files 从 target tag 取)
	cur, err := s.store.Load(row.Scope, row.Name, row.Version, row.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("skillaudit: load current: %w", err)
	}
	// 用 target 的 file 列表 + 当前 manifest 的元数据 构造 WriteInput
	mfst := cur.Manifest
	// 重新构造 files
	files := make([]skilladapter.File, 0, len(target))
	for _, f := range target {
		files = append(files, skilladapter.File{Path: f.Path, Content: f.Content})
	}
	if _, err := sskill.New(s.dbWrite, s.dbRead, s.store).Update(row.Scope, row.Name, row.Version, row.ProjectID, &sskill.WriteInput{
		Scope:     row.Scope,
		ProjectID: row.ProjectID,
		Manifest:  mfst,
		Files:     files,
		Source:    row.Source,
		SourceRef: row.SourceRef,
	}); err != nil {
		return nil, fmt.Errorf("skillaudit: replace files: %w", err)
	}
	return &RollbackOutput{
		PreRollbackTagID: preOut.TagID,
		PreRollbackTag:   preName,
		FilesRestored:    len(target),
	}, nil
}
