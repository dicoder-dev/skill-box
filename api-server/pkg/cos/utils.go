package cos

import "fmt"

// generateGroupedPath 根据ID值生成分组路径，每1000个ID作为一组
// 例如: id=1 返回 "0_1000", id=1500 返回 "1000_2000"
func generateGroupedPath(id uint) string {
	groupStart := (id / 1000) * 1000
	groupEnd := groupStart + 1000
	return fmt.Sprintf("%d_%d", groupStart, groupEnd)
}

// GetUserDataPath 根据用户ID获取用户数据文件夹路径
// 例如: userId=1 返回 "uploads/user/0_1000/1"
func GetUserDataPath(userId uint) string {
	groupPath := generateGroupedPath(userId)
	return fmt.Sprintf("uploads/user/%s/%d", groupPath, userId)
}

// GetStudioDataPath 根据工作室ID获取工作室数据文件夹路径
// 例如: studioId=1 返回 "uploads/studio/0_1000/1"
func GetStudioDataPath(studioId uint) string {
	groupPath := generateGroupedPath(studioId)
	return fmt.Sprintf("uploads/studio/%s/%d", groupPath, studioId)
}