package services

import (
	"runtime"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// PlatformService 暴露平台/系统能力给前端。
type PlatformService struct {
	app *application.App
}

// NewPlatformService 构造 PlatformService。
func NewPlatformService(app *application.App) *PlatformService {
	return &PlatformService{app: app}
}

// OS 返回 "darwin" / "windows" / "linux" 等。
func (s *PlatformService) OS() string {
	return runtime.GOOS
}

// Arch 返回架构,如 "arm64" / "amd64"。
func (s *PlatformService) Arch() string {
	return runtime.GOARCH
}

// ClipboardText 读取剪贴板文本。
func (s *PlatformService) ClipboardText() string {
	if s.app == nil {
		return ""
	}
	if text, ok := s.app.Clipboard.Text(); ok {
		return text
	}
	return ""
}

// SetClipboardText 写入剪贴板文本。
func (s *PlatformService) SetClipboardText(text string) bool {
	if s.app == nil {
		return false
	}
	return s.app.Clipboard.SetText(text)
}

// OpenExternal 用系统默认浏览器打开 URL。
func (s *PlatformService) OpenExternal(url string) error {
	if s.app == nil {
		return nil
	}
	return s.app.Browser.OpenURL(url)
}
