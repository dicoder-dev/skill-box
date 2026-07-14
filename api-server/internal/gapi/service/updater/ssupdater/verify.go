// Package ssupdater 包级 verify.go 是 SHA256 校验的独立入口。
//
// 设计意图:把"读文件 + sha256"提到顶层,避免在 download.go 内部掺杂过深的 IO 逻辑;
// 未来切换到 minisign / cosign 等签名校验只需替换本文件的入口,不影响 download。
package ssupdater

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"
)

// VerifyFileSHA256 计算 path 的 sha256,与 expect(忽略大小写)比对;
// expect 空字符串直接通过(部分 manifest 可能没填 sha256,MVP 容忍)。
//
// 失败时把期望值与实算值都返回,便于 controller 在响应里给用户看。
func VerifyFileSHA256(path, expect string) error {
	if expect == "" {
		return nil
	}
	got, err := sha256OfFile(path)
	if err != nil {
		return fmt.Errorf("sha256 read failed: %w", err)
	}
	if !strings.EqualFold(got, expect) {
		return fmt.Errorf("sha256 mismatch: got=%s expect=%s", got, expect)
	}
	return nil
}

func sha256OfFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
