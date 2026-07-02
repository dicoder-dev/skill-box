// Package toolicon 提供工具自定义图标文件的物理存储管理。
//
// 设计:
//   - 图标文件存到 ~/.<AppName>/tool-icons/<name>.<ext>
//     即 ~/.skill-box/tool-icons/claude.png 这种 basename 形式
//   - 调用方只传 basename(无路径分隔符,无 ../) — 此处做最终兜底校验
//   - 列表接口返回的内容必须经过 validIconFileName — 防路径穿越
//
// 为什么独立成一个包:
//   - 上传 controller / 静态文件服务 controller / seed 数据 copy 三处都要用到
//   - 不跟 stool service 耦合(避免 service 互相 import 死循环)
package toolicon

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"ginp-api/share/func"
)

const (
	// iconsDirName ~/.skill-box 下的子目录名
	iconsDirName = "tool-icons"
)

// allowedExts 后缀白名单。
var allowedExts = []string{".png", ".svg", ".jpg", ".jpeg", ".webp", ".ico", ".gif"}

// ValidIconFileName 校验 name 是合法 basename(无路径分隔符、..、可疑后缀)。
// 对外暴露供其他包复用。
func ValidIconFileName(name string) bool {
	if name == "" {
		return false
	}
	if strings.ContainsAny(name, "/\\") {
		return false
	}
	if strings.Contains(name, "..") {
		return false
	}
	lower := strings.ToLower(name)
	for _, ext := range allowedExts {
		if strings.HasSuffix(lower, ext) {
			return true
		}
	}
	return false
}

// Dir 返回工具图标目录的绝对路径(末尾无分隔符),目录若不存在则 MkdirAll。
func Dir() (string, error) {
	base := sharefunc.DataDir()
	if base == "" {
		return "", errors.New("toolicon: home dir unresolved")
	}
	full := filepath.Join(base, iconsDirName)
	if err := os.MkdirAll(full, 0o755); err != nil {
		return "", err
	}
	return full, nil
}

// ResolveAbsPath 把 basename 解析为绝对路径,做兜底校验;不合规返 error。
func ResolveAbsPath(name string) (string, error) {
	if !ValidIconFileName(name) {
		return "", errors.New("toolicon: invalid icon file name")
	}
	d, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, name), nil
}

// Delete 删除指定 basename 对应的文件;不存在不报错(返回 nil)。
func Delete(name string) error {
	if !ValidIconFileName(name) {
		return nil // 不合规的"文件名"直接当不存在处理,不报错
	}
	p, err := ResolveAbsPath(name)
	if err != nil {
		return nil
	}
	err = os.Remove(p)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// SaveBytes 把字节流写到 iconsDir/<basename>。
// 调用方负责先调 ValidIconFileName 给上层做空指针/后缀校验。
func SaveBytes(name string, data []byte) (string, error) {
	abs, err := ResolveAbsPath(name)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(abs, data, 0o644); err != nil {
		return "", err
	}
	return abs, nil
}
