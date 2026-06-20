package utils

import (
	"encoding/base64"
	"fmt"

	"github.com/google/uuid"
)

func GetGuidStr() string {
	// 生成一个新的 UUIDv4
	id := uuid.New()

	// 将 UUID 转换为字符串
	idStr := id.String()
	return idStr
}

func GetGuidBase64() (string, error) {
	uuidStr := GetGuidStr()
	// 将 UUID 转换为字节切片
	uuidBytes, err := uuid.Parse(uuidStr)
	if err != nil {
		return uuidStr, fmt.Errorf("failed to parse UUID: %w", err)
	}

	// 对字节切片进行 Base64 编码
	base64Encoded := base64.StdEncoding.EncodeToString(uuidBytes[:])

	return base64Encoded, nil

}
