package upload

import (
	"fmt"
	"ginp-api/pkg/utils"
	"strings"
	"time"
)

// getFileType 根据文件后缀判断文件类型
func GetFileType(ext string) string {
	// 处理文件后缀：去除前导点并转为小写
	ext = strings.TrimPrefix(strings.ToLower(ext), ".")

	switch ext {
	case "jpg", "jpeg", "png", "gif", "bmp", "webp":
		return "images"
	case "mp4", "avi", "mov", "wmv", "flv", "mkv":
		return "videos"
	case "mp3", "wav", "flac", "aac":
		return "audios"
	case "pdf", "doc", "docx", "txt", "xls", "xlsx", "ppt", "pptx":
		return "documents"
	default:
		return "others"
	}
}

// generateGroupedPath 根据ID值生成分组路径，每1000个ID作为一组
// 例如: id=1 返回 "0_1000", id=1500 返回 "1000_2000"
func generateGroupedPath(id uint) string {
	groupStart := (id / 1000) * 1000
	groupEnd := groupStart + 1000
	return fmt.Sprintf("%d_%d", groupStart, groupEnd)
}

// GenerateFilePath 生成文件路径
// fileExt: 文件后缀（支持带点或不带点，如 ".png" 或 "png"）
// customFileName: 自定义文件名，如果为空则自动生成
// userId: 用户ID（0表示系统上传）
// isUserUpload: 是否为用户上传，false表示系统上传
// prefix: 路径前缀（如 "static" 或 ""），本地存储需要 "static"，COS 不需要
// appKey: 应用Key，用于在路径中区分不同应用
// 返回: 文件相对路径
func GenerateFilePath(fileExt string, customFileName string, userId uint, isUserUpload bool, prefix string, appKey string) string {
	// 处理文件后缀：去除前导点
	fileExt = strings.TrimPrefix(strings.ToLower(fileExt), ".")
	if fileExt == "" {
		return ""
	}

	// 获取当前时间
	now := time.Now()
	dateFolder := now.Format("2006-01-02") // 日期目录，格式：2025-10-24
	timePrefix := now.Format("15-04-05")   // 时-分-秒，格式：18-29-16

	// 生成文件名：时-分-秒_uuid.后缀
	fileName := customFileName
	if fileName == "" { // 没有自定义文件名，则随机生成
		uuidStr := utils.GetGuidStr()
		fileName = fmt.Sprintf("%s_%s", timePrefix, uuidStr)
	}

	fileType := GetFileType(fileExt)

	var filePath string
	if isUserUpload && userId > 0 {
		// 用户上传路径：uploads/user/[id区间]/[用户ID]/[文件类型]/[appKey]/[日期]/[时-分-秒_uuid].后缀
		// 例如：uploads/user/0_1000/1/images/foodwise/2025-12-19/18-29-16_5bbd5872-209b-4e96-a38d-9d196db4134a.png
		groupPath := generateGroupedPath(userId)
		if appKey != "" {
			filePath = fmt.Sprintf("uploads/user/%s/%d/%s/%s/%s/%s.%s", groupPath, userId, fileType, appKey, dateFolder, fileName, fileExt)
		} else {
			filePath = fmt.Sprintf("uploads/user/%s/%d/%s/%s/%s.%s", groupPath, userId, fileType, dateFolder, fileName, fileExt)
		}
	} else {
		// 系统上传路径：uploads/system/[文件类型]/[日期]/[时-分-秒_uuid].后缀
		// 例如：uploads/system/images/2025-10-24/18-29-16_5bbd5872-209b-4e96-a38d-9d196db4134a.png
		filePath = fmt.Sprintf("uploads/system/%s/%s/%s.%s", fileType, dateFolder, fileName, fileExt)
	}

	// 如果有前缀，添加前缀
	if prefix != "" {
		filePath = fmt.Sprintf("%s/%s", prefix, filePath)
	}

	return filePath
}
