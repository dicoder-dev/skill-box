// Package settings 提供 Skill Box 通用键值设置存储。
//
// 底层走 entity.Setting 表 + model/skillbox/msetting.Model;设计上
// 屏蔽 dbops 细节,UI / 后端各业务模块统一通过 Get / Set / GetAll / Delete
// 四个方法访问。
//
// 设计见 docs/project/需求规划.md 第 6.11 节。
package settings

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"ginp-api/internal/gapi/entity"
	msetting "ginp-api/internal/gapi/model/skillbox/msetting"
	"ginp-api/pkg/where"

	"gorm.io/gorm"
)

// Service 通用键值设置服务。
type Service struct {
	dbWrite *gorm.DB
	dbRead  *gorm.DB
}

// New 构造 Service。
func New(dbWrite, dbRead *gorm.DB) *Service {
	return &Service{dbWrite: dbWrite, dbRead: dbRead}
}

func (s *Service) model() *msetting.Model {
	return msetting.NewModel(s.dbWrite, s.dbRead)
}

// Get 取单个值;不存在返回 ("", false, nil)。
func (s *Service) Get(key string) (string, bool, error) {
	if key == "" {
		return "", false, errors.New("settings: empty key")
	}
	row, err := s.model().FindOne(where.New(msetting.FieldKey, "=", key).Conditions())
	if err != nil {
		// FindOne 在 model 层已经包了 not-found 错误;这里统一按 not found 处理
		return "", false, nil
	}
	if row == nil || row.ID == 0 {
		return "", false, nil
	}
	return row.Value, true, nil
}

// Set 写入单个键值;存在则覆盖。
func (s *Service) Set(key, value string) error {
	if key == "" {
		return errors.New("settings: empty key")
	}
	row, err := s.model().FindOne(where.New(msetting.FieldKey, "=", key).Conditions())
	if err == nil && row != nil && row.ID > 0 {
		row.Value = value
		return s.model().Update(where.New(msetting.FieldID, "=", row.ID).Conditions(), row)
	}
	_, cerr := s.model().Create(&entity.Setting{Key: key, Value: value})
	return cerr
}

// GetJSON 反序列化 JSON 值到 dst。键不存在返回 (false, nil)。
func (s *Service) GetJSON(key string, dst any) (bool, error) {
	v, ok, err := s.Get(key)
	if err != nil {
		return false, err
	}
	if !ok {
		return false, nil
	}
	if err := json.Unmarshal([]byte(v), dst); err != nil {
		return false, fmt.Errorf("settings: unmarshal %q: %w", key, err)
	}
	return true, nil
}

// SetJSON 把 dst 序列化为 JSON 后写入。
func (s *Service) SetJSON(key string, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("settings: marshal: %w", err)
	}
	return s.Set(key, string(b))
}

// Delete 删除键;不存在时返回 nil(幂等)。
func (s *Service) Delete(key string) error {
	if key == "" {
		return errors.New("settings: empty key")
	}
	row, err := s.model().FindOne(where.New(msetting.FieldKey, "=", key).Conditions())
	if err != nil || row == nil || row.ID == 0 {
		return nil
	}
	return s.model().Delete(where.New(msetting.FieldID, "=", row.ID).Conditions())
}

// Snapshot 全部设置的内存镜像,UI Settings 页可以直接渲染。
type Snapshot struct {
	Items map[string]string
}

// GetAll 返回所有设置项的快照。
func (s *Service) GetAll() (*Snapshot, error) {
	list, _, err := s.model().FindList(nil, nil)
	if err != nil {
		return nil, err
	}
	out := &Snapshot{Items: make(map[string]string, len(list))}
	for _, row := range list {
		out.Items[row.Key] = row.Value
	}
	return out, nil
}

// ApplyMode apply 落盘模式(2026-07-02 增)。
//
//   - copy:    沿用旧行为,把 canonical 文件逐个拷贝到目标目录(占空间,文件副本独立)。
//   - symlink: 把目标目录做成一个软链接指向 skillstore 里的真实 skill 根(零占用,
//              改源文件后目标端即时生效)。
//
// 默认 copy —— 不破坏任何已 apply 的记录,迁移是显式由用户发起。
const (
	ApplyModeCopy    = "copy"
	ApplyModeSymlink = "symlink"

	// KeyApplyMode 通用偏好里 apply_mode 的存储键。
	// 放在 settings 表(走 settings.Service),而不是 SkillApply 行,这样:
	//   - 用户切换一次,后续 apply 都按新模式;
	//   - SkillApply 行只记录"当时用的是什么模式",便于迁移时回查。
	KeyApplyMode = "skillbox.apply_mode"
)

// GetApplyMode 读取当前 apply_mode;不存在或非法值时返回 fallback(默认 copy)。
func (s *Service) GetApplyMode() string {
	v, ok, err := s.Get(KeyApplyMode)
	if err != nil || !ok {
		return ApplyModeCopy
	}
	v = strings.ToLower(strings.TrimSpace(v))
	if v != ApplyModeCopy && v != ApplyModeSymlink {
		return ApplyModeCopy
	}
	return v
}

// SetApplyMode 写入 apply_mode;值不合法返回 error(便于 controller 弹 4xx)。
func (s *Service) SetApplyMode(mode string) error {
	mode = strings.ToLower(strings.TrimSpace(mode))
	if mode != ApplyModeCopy && mode != ApplyModeSymlink {
		return fmt.Errorf("settings: invalid apply_mode %q (allowed: copy/symlink)", mode)
	}
	return s.Set(KeyApplyMode, mode)
}
