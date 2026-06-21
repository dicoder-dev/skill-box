package cos

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"ginp-api/configs"
)

// UploadBytes 服务端直接上传字节内容到 COS。
// 返回 objectKey 与可访问 URL。
func (s *Signer) UploadBytes(objectKey string, content []byte, contentType string) (string, error) {
	if strings.TrimSpace(objectKey) == "" {
		return "", fmt.Errorf("object key is empty")
	}

	opt := &ObjectPutOptions{}
	if contentType != "" {
		opt = &ObjectPutOptions{
			ObjectPutHeaderOptions: &ObjectPutHeaderOptions{
				ContentType: contentType,
			},
		}
	}

	_, err := s.client.Object.Put(context.Background(), objectKey, bytes.NewReader(content), opt)
	if err != nil {
		return "", err
	}

	return objectKey, nil
}

func BuildBucketURL() string {
	bucketName := configs.TencentCosBucketName()
	region := configs.TencentCosRegion()
	if bucketName == "" || region == "" {
		return ""
	}
	return fmt.Sprintf("https://%s.cos.%s.myqcloud.com", bucketName, region)
}

func NewDefaultSigner() (*Signer, error) {
	bucketURL := BuildBucketURL()
	if bucketURL == "" {
		return nil, fmt.Errorf("cos bucket url config is incomplete")
	}
	return NewSigner(bucketURL, configs.TencentCosSecretID(), configs.TencentCosSecretKey())
}

func BuildSystemImageObjectKey(fileExt, customFileName string) string {
	fileExt = strings.TrimPrefix(strings.ToLower(fileExt), ".")
	if fileExt == "" {
		fileExt = "jpg"
	}

	now := time.Now()
	dateFolder := now.Format("2006-01-02")
	return fmt.Sprintf("uploads/canshu/images/%s/%s.%s", dateFolder, customFileName, fileExt)
}

func BuildCanshuImageObjectKey(relativePath string) string {
	relativePath = strings.TrimSpace(relativePath)
	relativePath = strings.TrimPrefix(relativePath, "/")
	if relativePath == "" {
		return ""
	}
	return "uploads/canshu/" + relativePath
}

// BuildPublicURL 生成对外访问地址。
// 现网优先使用项目里已存在的 CDN 域名风格，未命中则回退到 bucket 域名。
func BuildPublicURL(objectKey string) string {
	objectKey = strings.TrimPrefix(objectKey, "/")
	if objectKey == "" {
		return ""
	}

	const defaultCDNBase = "https://ecdn.dicoder.cn"
	if strings.HasPrefix(objectKey, "uploads/system/") || strings.HasPrefix(objectKey, "uploads/user/") || strings.HasPrefix(objectKey, "uploads/canshu/") {
		return defaultCDNBase + "/" + objectKey
	}

	if bucketURL := BuildBucketURL(); bucketURL != "" {
		return strings.TrimRight(bucketURL, "/") + "/" + objectKey
	}

	return objectKey
}

func DetectContentTypeByExt(fileName string, fallback []byte) string {
	ext := strings.ToLower(filepath.Ext(fileName))
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".bmp":
		return "image/bmp"
	default:
		if len(fallback) > 0 {
			return http.DetectContentType(fallback)
		}
		return "application/octet-stream"
	}
}
