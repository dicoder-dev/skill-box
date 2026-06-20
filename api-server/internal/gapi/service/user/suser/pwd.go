package suser

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// PasswordConfig 密码加密配置
type PasswordConfig struct {
	Time    uint32 // 时间成本参数
	Memory  uint32 // 内存成本参数（KB）
	Threads uint8  // 并行度
	KeyLen  uint32 // 密钥长度
	SaltLen uint32 // 盐长度
}

// 默认密码配置
var defaultConfig = &PasswordConfig{
	Time:    1,
	Memory:  64 * 1024, // 64MB
	Threads: 4,
	KeyLen:  32,
	SaltLen: 16,
}

// HashPassword 使用Argon2id算法对密码进行哈希加密
// 返回格式: $argon2id$v=19$m=65536,t=1,p=4$salt$hash
func HashPassword(password string) (string, error) {
	return HashPasswordWithConfig(password, defaultConfig)
}

// HashPasswordWithConfig 使用自定义配置对密码进行哈希加密
func HashPasswordWithConfig(password string, config *PasswordConfig) (string, error) {
	if password == "" {
		return "", errors.New("password cannot be empty")
	}

	// 生成随机盐
	salt := make([]byte, config.SaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}

	// 使用Argon2id进行哈希
	hash := argon2.IDKey([]byte(password), salt, config.Time, config.Memory, config.Threads, config.KeyLen)

	// 编码为base64
	saltB64 := base64.RawStdEncoding.EncodeToString(salt)
	hashB64 := base64.RawStdEncoding.EncodeToString(hash)

	// 返回格式化的哈希字符串
	return fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		config.Memory, config.Time, config.Threads, saltB64, hashB64), nil
}

// VerifyPassword 验证密码是否匹配
func VerifyPassword(password, hashedPassword string) (bool, error) {
	if password == "" || hashedPassword == "" {
		return false, errors.New("password and hash cannot be empty")
	}

	// 解析哈希字符串
	config, salt, hash, err := parseHash(hashedPassword)
	if err != nil {
		return false, fmt.Errorf("failed to parse hash: %w", err)
	}

	// 使用相同参数重新计算哈希
	computedHash := argon2.IDKey([]byte(password), salt, config.Time, config.Memory, config.Threads, config.KeyLen)

	// 使用constant time比较防止时序攻击
	return subtle.ConstantTimeCompare(hash, computedHash) == 1, nil
}

// parseHash 解析哈希字符串，提取配置参数、盐和哈希值
func parseHash(hashedPassword string) (*PasswordConfig, []byte, []byte, error) {
	parts := strings.Split(hashedPassword, "$")
	if len(parts) != 6 {
		return nil, nil, nil, errors.New("invalid hash format")
	}

	if parts[1] != "argon2id" {
		return nil, nil, nil, errors.New("unsupported algorithm")
	}

	if parts[2] != "v=19" {
		return nil, nil, nil, errors.New("unsupported version")
	}

	// 解析参数
	var memory, time uint32
	var threads uint8
	_, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &time, &threads)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to parse parameters: %w", err)
	}

	// 解码盐
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to decode salt: %w", err)
	}

	// 解码哈希
	hash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to decode hash: %w", err)
	}

	config := &PasswordConfig{
		Time:    time,
		Memory:  memory,
		Threads: threads,
		KeyLen:  uint32(len(hash)),
		SaltLen: uint32(len(salt)),
	}

	return config, salt, hash, nil
}