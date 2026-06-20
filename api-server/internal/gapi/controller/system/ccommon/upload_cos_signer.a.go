package ccommon

import (
	"ginp-api/configs"
	"ginp-api/pkg/cos"

	"ginp-api/pkg/ginp"
)

const ApiUploadCosSigner = "/api/common/upload_cos_signer" //API Path

// RequestUploadCosSigner 请求结构体，可按需添加字段
type RequestUploadCosSigner struct {
	FileSuffix     string `json:"file_suffix"`                // 文件后缀
	StudioId       uint   `json:"studio_id,omitempty"`        // 工作室ID 非必须，默认0
	CustomFileName string `json:"custom_file_name,omitempty"` // 自定义文件名 非必须，默认“”
}

// RespondUploadCosSigner 响应结构体，包含 COS 直传所需参数
type RespondUploadCosSigner struct {
	PresignedURL string `json:"presigned_url"`  // COS 预签名上传 URL
	FileKey      string `json:"file_key"`       // 文件在 COS 中的存储路径
	TmpSecretID  string `json:"tmp_secret_id"`  // 临时密钥 ID
	TmpSecretKey string `json:"tmp_secret_key"` // 临时密钥 Key
	SessionToken string `json:"session_token"`  // 会话令牌
}

func UploadCosSigner(c *ginp.ContextPlus) {
	var req RequestUploadCosSigner
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Fail("request param error:" + err.Error())
		return
	}
	userId := uint(1)
	if req.FileSuffix == "" {
		c.Fail("file suffix is empty")
		return
	}

	config := &cos.STSConfig{
		SecretID:  configs.TencentCosSecretID(),
		SecretKey: configs.TencentCosSecretKey(),
		Bucket:    configs.TencentCosBucketName(),
		Region:    configs.TencentCosRegion(),
		AppID:     configs.TencentCosBucketAppId(),
		//自定义文件名
		CustomFileName: req.CustomFileName,
		Duration:       int64(configs.TencentCosDuration()), //单位秒
		UserId:         userId,                              //用户ID
		StudioId:       req.StudioId,                        //工作室ID，如果为0则表示为用户上传的数据
	}

	stsSigner, err := cos.NewSTSSigner(config)
	if err != nil {
		c.Fail("签名器初始化失败:" + err.Error())
		return
	}

	presignedURL, fileKey, err := stsSigner.GeneratePresignedURL(req.FileSuffix)
	if err != nil {
		c.Fail("生成预签名URL失败:" + err.Error())
		return
	}

	credential := stsSigner.GetCredential()

	resp := RespondUploadCosSigner{
		PresignedURL: presignedURL,
		FileKey:      fileKey,
		TmpSecretID:  credential.Credentials.TmpSecretID,
		TmpSecretKey: credential.Credentials.TmpSecretKey,
		SessionToken: credential.Credentials.SessionToken,
	}

	c.SuccessData(resp)
}

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           ApiUploadCosSigner,                    //api路径
		Handlers:       ginp.RegisterHandler(UploadCosSigner), //对应控制器
		HttpType:       ginp.HttpPost,                         //http请求类型
		NeedLogin:      false,                                 //是否需要登录
		NeedPermission: false,                                 //是否需要鉴权
		PermissionName: "common.upload_cos_pre",               //完整的权限名称,会跟权限表匹配
		Swagger: &ginp.SwaggerInfo{
			Title:       "upload_cos_pre",
			Description: "获取 COS 直传所需参数",
			RequestDto:  RequestUploadCosSigner{},
		},
	})
}
