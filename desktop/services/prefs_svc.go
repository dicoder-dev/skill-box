// Package services 桌面端服务绑定 — PrefsService。
//
// PrefsService 包装 internal/settings.Service,把桌面端偏好 key 暴露给前端。
// 前端通过 window.go.prefs.PrefsService.{Get, Set, GetAll} 读写。
package services

// PrefsStore 抽象 settings.Service 的最小能力,避免 services 反向依赖。
type PrefsStore interface {
	Get(key string) (string, bool, error)
	Set(key, value string) error
	GetAll() (map[string]string, error)
}

// PrefsService 暴露给前端的桌面端偏好服务。
type PrefsService struct {
	store PrefsStore
}

// NewPrefsService 构造 PrefsService。store 可为 nil(此时所有方法返回默认值)。
func NewPrefsService(store PrefsStore) *PrefsService {
	return &PrefsService{store: store}
}

// Get 取单个偏好值;不存在返回 ("", false, nil)。
func (s *PrefsService) Get(key string) (string, bool, error) {
	if s.store == nil {
		return "", false, nil
	}
	return s.store.Get(key)
}

// Set 写入单个偏好。
func (s *PrefsService) Set(key, value string) error {
	if s.store == nil {
		return nil
	}
	return s.store.Set(key, value)
}

// GetAll 返回所有偏好的快照。
func (s *PrefsService) GetAll() (map[string]string, error) {
	if s.store == nil {
		return map[string]string{}, nil
	}
	return s.store.GetAll()
}
