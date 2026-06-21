package skillaudit

import (
	"fmt"
	"time"
)

// TagSnapshot 一次打 tag 的输入。
type TagSnapshot struct {
	SkillID  uint
	Tag      string
	Message  string
	IsImplicit bool
	Files    []FileSnap
}

// TagResult 一次打 tag 的输出(写到 DB 用)。
type TagResult struct {
	TagID    uint      `json:"tag_id"`
	SkillID  uint      `json:"skill_id"`
	Tag      string    `json:"tag"`
	Message  string    `json:"message"`
	IsImplicit bool    `json:"is_implicit"`
	Files    []TaggedFile `json:"files"`
	CreatedAt time.Time `json:"created_at"`
}

// TaggedFile 单文件 tag 记录(给 service 层转 entity.SkillFileSnapshot)。
type TaggedFile struct {
	Path        string `json:"path"`
	Content     string `json:"content"`
	ContentHash string `json:"content_hash"`
}

// BuildTag 校验入参 + 把 FileSnap 转成 TaggedFile(hash 已算好)。
// 不写 DB - 由 service 层负责落库。
func BuildTag(in TagSnapshot) (*TagResult, error) {
	if err := ValidateTag(in.Tag); err != nil {
		return nil, err
	}
	if len(in.Files) == 0 {
		return nil, ErrEmptyFiles
	}
	out := &TagResult{
		SkillID:    in.SkillID,
		Tag:        in.Tag,
		Message:    in.Message,
		IsImplicit: in.IsImplicit,
		CreatedAt:  time.Now(),
		Files:      make([]TaggedFile, 0, len(in.Files)),
	}
	for _, f := range in.Files {
		if f.Path == "" {
			return nil, fmt.Errorf("%w: empty path in files", ErrEmptyFiles)
		}
		out.Files = append(out.Files, TaggedFile{
			Path:        f.Path,
			Content:     f.Content,
			ContentHash: HashContent(f.Content),
		})
	}
	return out, nil
}

// ImplicitPreRollbackTag 生成"预回滚"的隐式 tag 名(避免重复)。
// 格式:_pre_rollback_<RFC3339-ish 紧凑版>;秒级精度足够。
func ImplicitPreRollbackTag(now time.Time) string {
	return "_pre_rollback_" + now.UTC().Format("20060102T150405")
}

// FilesFromTagged 把 TaggedFile 转回 FileSnap(给 rollback 用)。
func FilesFromTagged(tagged []TaggedFile) []FileSnap {
	out := make([]FileSnap, 0, len(tagged))
	for _, t := range tagged {
		out = append(out, FileSnap{Path: t.Path, Content: t.Content})
	}
	return out
}
