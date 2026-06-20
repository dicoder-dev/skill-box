package services

import (
	desktoppkg "skill-box/desktop"
)

// WindowService 暴露给前端的窗口控制能力。
// 仅做"窗口层"的事,业务逻辑不允许塞到这里。
type WindowService struct {
	mgr *desktoppkg.WindowManager
}

// NewWindowService 构造 WindowService。
func NewWindowService(mgr *desktoppkg.WindowManager) *WindowService {
	return &WindowService{mgr: mgr}
}

// ToggleAlwaysOnTop 切换窗口置顶,返回切换后的状态。
func (s *WindowService) ToggleAlwaysOnTop() bool {
	if s.mgr == nil {
		return false
	}
	return s.mgr.ToggleAlwaysOnTop()
}

// Show 主窗口。
func (s *WindowService) Show() {
	if s.mgr == nil {
		return
	}
	s.mgr.ShowPrimary()
}

// ToggleMaximise 切换窗口最大化。
func (s *WindowService) ToggleMaximise() {
	if s.mgr == nil {
		return
	}
	primary := s.mgr.Primary()
	if primary != nil {
		primary.ToggleMaximise()
	}
}
