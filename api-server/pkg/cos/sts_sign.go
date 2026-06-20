package cos

import (
	"context"
	"errors"
	"fmt"
	"ginp-api/pkg/utils"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/tencentyun/cos-go-sdk-v5"
	sts "github.com/tencentyun/qcloud-cos-sts-sdk/go"
)

// STSSigner 用于生成基于临时密钥的 COS 签名
type STSSigner struct {
	client *cos.Client
	sts    *sts.CredentialResult
	config *STSConfig
}

// STSConfig 临时密钥配置
type STSConfig struct {
	SecretID       string
	SecretKey      string
	Bucket         string
	Region         string
	CustomFileName string // 自定义文件名,如果不为空则使用该文件名，现在已经有的话则会覆盖，适用于头像等场景
	Duration       int64  // 临时密钥有效期（秒）
	AllowPrefix    string // 允许操作的路径前缀
	AppID          string // APPID
	UserId         uint   // 用户ID
	StudioId       uint   // 工作室ID
}

// NewSTSSigner 创建一个新的 STSSigner 实例
func NewSTSSigner(config *STSConfig) (*STSSigner, error) {
	if config == nil {
		return nil, errors.New("config is nil")
	}

	// 获取临时密钥
	stsClient := sts.NewClient(config.SecretID, config.SecretKey, nil)

	// 策略配置
	opt := &sts.CredentialOptions{
		DurationSeconds: config.Duration, // 临时密钥有效期（秒）
		Policy: &sts.CredentialPolicy{
			Version: "2.0",
			Statement: []sts.CredentialPolicyStatement{
				{
					Action: []string{
						"name/cos:PostObject",
						"name/cos:PutObject",
					},
					Effect: "allow",
					Resource: []string{
						// 这里改成允许的路径前缀，可以根据自己网站的用户登录态判断允许上传的具体路径，例子： a.jpg 或者 a/* 或者 * (使用通配符*存在重大安全风险, 请谨慎评估使用)
						// 存储桶的命名格式为 BucketName-APPID，此处填写的 bucket 必须为此格式
						// "qcs::cos:ap-guangzhou:uid/" + config.appId + ":" + config.Bucket + "/" + key,
						fmt.Sprintf("qcs::cos:%s:uid/%s:%s/%s*", config.Region, config.AppID, config.Bucket, config.AllowPrefix),
					},
				},
			},
		},
	}

	// 请求临时密钥
	res, err := stsClient.GetCredential(opt)
	if err != nil {
		return nil, err
	}

	// 创建 COS 客户端
	bucketURL := fmt.Sprintf("https://%s.cos.%s.myqcloud.com", config.Bucket, config.Region)
	u, err := url.Parse(bucketURL)
	if err != nil {
		return nil, err
	}

	b := &cos.BaseURL{BucketURL: u}
	client := cos.NewClient(b, &http.Client{
		Transport: &cos.AuthorizationTransport{
			SecretID:     res.Credentials.TmpSecretID,
			SecretKey:    res.Credentials.TmpSecretKey,
			SessionToken: res.Credentials.SessionToken,
		},
	})

	return &STSSigner{
		config: config,
		client: client,
		sts:    res,
	}, nil
}

// getFileType 根据文件后缀判断文件类型
func getFileType(ext string) string {
	ext = strings.ToLower(ext)

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

// GeneratePresignedURL 生成基于临时密钥的预签名 URL
func (s *STSSigner) GeneratePresignedURL(fileExt string) (string, string, error) {
	if s.config == nil {
		return "", "", fmt.Errorf("STSSigner config is nil")
	}
	// 自定义文件名
	fileName := s.config.CustomFileName
	if s.config.CustomFileName == "" { //没有自定义文件名，则随机生成
		formattedTime := time.Now().Format("20060102150405")
		fileName = fmt.Sprintf("%s_%d_%s", formattedTime, time.Now().Unix(), utils.GenerateRandomString(8))
	}
	fileType := getFileType(fileExt)

	// 如果工作室ID为0，则表示为用户上传的数据 uploads/user/0_1000/1/images/1.png
	var fileKey string
	if s.config.StudioId > 0 {
		// 工作室上传的数据
		groupPath := GetStudioDataPath(s.config.StudioId)
		fileKey = fmt.Sprintf("%s/%s/%s.%s", groupPath, fileType, fileName, fileExt)
	} else {
		// 用户上传的数据
		groupPath := GetUserDataPath(s.config.UserId)
		fileKey = fmt.Sprintf("%s/%s/%s.%s", groupPath, fileType, fileName, fileExt)
	}

	presignedURL, err := s.client.Object.GetPresignedURL2(context.Background(), "PUT", fileKey, time.Duration(s.config.Duration)*time.Second, nil)
	if err != nil {
		return "", "", err
	}
	return presignedURL.String(), fileKey, nil
}

// GetCredential 获取临时密钥信息
func (s *STSSigner) GetCredential() *sts.CredentialResult {
	return s.sts
}

// RefreshCredential 刷新临时密钥
func (s *STSSigner) RefreshCredential(config *STSConfig) error {
	// 获取新的临时密钥
	stsClient := sts.NewClient(
		config.SecretID,
		config.SecretKey,
		nil,
	)

	opt := &sts.CredentialOptions{
		DurationSeconds: config.Duration,
		Policy: &sts.CredentialPolicy{
			Version: "2.0",
			Statement: []sts.CredentialPolicyStatement{
				{
					Action: []string{
						"name/cos:PostObject",
						"name/cos:PutObject",
					},
					Effect: "allow",
					Resource: []string{
						fmt.Sprintf("qcs::cos:%s:uid/%s:%s/%s*", config.Region, config.AppID, config.Bucket, config.AllowPrefix),
					},
				},
			},
		},
	}

	res, err := stsClient.GetCredential(opt)
	if err != nil {
		return err
	}

	// 更新客户端凭证
	// 重新创建客户端以更新凭证
	bucketURL := fmt.Sprintf("https://%s.cos.%s.myqcloud.com", config.Bucket, config.Region)
	u, err := url.Parse(bucketURL)
	if err != nil {
		return err
	}

	b := &cos.BaseURL{BucketURL: u}
	s.client = cos.NewClient(b, &http.Client{
		Transport: &cos.AuthorizationTransport{
			SecretID:     res.Credentials.TmpSecretID,
			SecretKey:    res.Credentials.TmpSecretKey,
			SessionToken: res.Credentials.SessionToken,
		},
	})

	s.sts = res
	return nil
}
