package upload

import (
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// IsImageFile 判断文件是否为图片文件
// 支持的格式：jpg, jpeg, gif, png, bmp
func IsImageFile(filename string) bool {
	imageExtensions := []string{".jpg", ".jpeg", ".gif", ".png", ".bmp", ".webp"}
	ext := strings.ToLower(filepath.Ext(filename))
	for _, imageExt := range imageExtensions {
		if ext == imageExt {
			return true
		}
	}
	return false
}

// GetImageDimensions 获取图片的宽度和高度
// 返回: width, height, error
func GetImageDimensions(filePath string) (int, int, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, 0, err
	}
	defer file.Close()

	config, _, err := image.DecodeConfig(file)
	if err != nil {
		return 0, 0, err
	}

	return config.Width, config.Height, nil
}

// TruncateFileName 处理文件名长度限制（数据库字段限制100字符）
// 如果文件名超过100字符，截取前97个字符并添加"..."
func TruncateFileName(fileName string) string {
	if len(fileName) > 100 {
		return fileName[:97] + "..."
	}
	return fileName
}

// TruncateMimeType 处理MIME类型长度限制（数据库字段限制100字符）
// 如果MIME类型超过100字符，只保留核心MIME类型部分
// 例如: "text/html; charset=utf-8" -> "text/html"
func TruncateMimeType(mimeType string) string {
	if len(mimeType) <= 100 {
		return mimeType
	}
	// 如果有分号，只保留分号前的部分
	if idx := strings.Index(mimeType, ";"); idx > 0 {
		return strings.TrimSpace(mimeType[:idx])
	}
	// 如果没有分号，直接截取前100字符
	return mimeType[:100]
}

// GetMaterialFileType 根据是否为图片文件确定素材类型
// 返回: "image" 或 "file"
func GetMaterialFileType(isImage bool) string {
	if isImage {
		return "image"
	}
	return "file"
}

// ParseMaterialTypeFromFileKey 从 file_key 解析素材类型和用户ID
// file_key 格式：uploads/system/... 或 uploads/user/[id区间]/[用户ID]/...
// 返回: materialType ("system" 或 "user"), userId
// 注意：对于用户上传，如果无法从路径解析 userId，返回 userId=0，需要调用者从上下文获取
func ParseMaterialTypeFromFileKey(fileKey string) (materialType string, userId uint) {
	if strings.HasPrefix(fileKey, "uploads/user/") {
		// 用户上传：尝试解析路径获取用户ID
		// 格式：uploads/user/[id区间]/[用户ID]/...
		parts := strings.Split(fileKey, "/")
		if len(parts) >= 4 {
			// parts[0] = "uploads"
			// parts[1] = "user"
			// parts[2] = "[id区间]" 如 "0_1000"
			// parts[3] = "[用户ID]" 如 "1"
			if parsedUserId, err := strconv.ParseUint(parts[3], 10, 32); err == nil {
				return "user", uint(parsedUserId)
			}
			return "user", 0 // 解析失败，返回0，需要调用者从上下文获取
		}
		return "user", 0
	} else if strings.HasPrefix(fileKey, "uploads/system/") {
		// 系统上传
		return "system", 0
	}
	// 默认系统上传
	return "system", 0
}

// GenerateAllowPrefix 生成 COS STS 的 AllowPrefix
// userId: 用户ID
// isUserUpload: 是否为用户上传
// appKey: 应用Key，用于在路径中区分不同应用
// 返回: allowPrefix 字符串
func GenerateAllowPrefix(userId uint, isUserUpload bool, appKey string) string {
	if isUserUpload {
		// 用户上传：设置允许访问的用户路径前缀
		// 格式：uploads/user/[id区间]/[用户ID]/（如果appKey不为空，则允许所有文件类型下的appKey目录）
		groupStart := (userId / 1000) * 1000
		groupEnd := groupStart + 1000
		if appKey != "" {
			// 允许所有文件类型下的appKey目录，例如：uploads/user/0_1000/1/images/foodwise/
			return fmt.Sprintf("uploads/user/%d_%d/%d/", groupStart, groupEnd, userId)
		}
		return fmt.Sprintf("uploads/user/%d_%d/%d/", groupStart, groupEnd, userId)
	}
	// 后台上传：设置允许访问的系统路径前缀
	return "uploads/system/"
}
