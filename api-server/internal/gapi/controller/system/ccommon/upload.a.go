package ccommon

import (
	"ginp-api/pkg/utils"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"ginp-api/pkg/ginp"
)

const ApiCommonUpload = "/api/common/upload" //这是一个测试接口连通性的接口

func init() {
	ginp.RouterAppend(ginp.RouterItem{
		Path:           ApiCommonUpload,              //api路径
		Handlers:       ginp.RegisterHandler(Upload), //对应控制器
		HttpType:       ginp.HttpPost,                //http请求类型
		NeedLogin:      true,                         //是否需要登录
		NeedPermission: true,                         //是否需要鉴权
		PermissionName: "Common.upload",              //完整的权限名称,会跟权限表匹配
	})
}

const baseUploadDir = "./static/upload"

// 文件类型映射目录
var fileTypeDirMap = map[string]string{
	".jpg":  "images",
	".jpeg": "images",
	".gif":  "images",
	".png":  "images",
	".bmp":  "images",
	".mp4":  "videos",
	".txt":  "texts",
	".doc":  "documents",
	".docx": "documents",
	".pdf":  "documents",
}

func Upload(c *ginp.ContextPlus) {
	// 获取上传的保存路径参数，默认为空字符串（相对路径）
	relativeSavePath := c.DefaultPostForm("save_path", "")
	// 获取上传的文件
	file, err := c.FormFile("file")
	if err != nil {
		c.Fail(err.Error())
		return
	}

	// 构建目标文件的路径
	currentTime := time.Now()
	dateFolder := currentTime.Format("2006-01-02")
	guid := utils.GetGuidStr()
	timeFilename := currentTime.Format("15-04-05") + "_" + guid + filepath.Ext(file.Filename)

	var uploadDir string
	if relativeSavePath != "" {
		uploadDir = filepath.Join(baseUploadDir, relativeSavePath)
	} else {
		fileExt := strings.ToLower(filepath.Ext(file.Filename))
		fileTypeDir, exists := fileTypeDirMap[fileExt]
		if !exists {
			fileTypeDir = "others"
		}
		uploadDir = filepath.Join(baseUploadDir, fileTypeDir, dateFolder)
	}

	dstPath := filepath.Join(uploadDir, timeFilename)
	err = os.MkdirAll(uploadDir, os.ModePerm)
	if err != nil {
		c.Fail(err.Error())
		return
	}

	dst, err := os.Create(dstPath)
	if err != nil {
		c.Fail(err.Error())
		return
	}
	defer dst.Close()
	// 将上传的文件写入目标文件
	src, err := file.Open()
	if err != nil {
		c.Fail(err.Error())
		return
	}
	defer src.Close()

	_, err = io.Copy(dst, src)
	if err != nil {
		c.Fail(err.Error())
		return
	}

	c.SuccessData(strings.ReplaceAll(dstPath, "./", "/"))

}

func isImageFile(filename string) bool {
	imageExtensions := []string{".jpg", ".jpeg", ".gif", ".png", ".bmp"} // 可扩展其他图片格式
	extension := strings.ToLower(filename[strings.LastIndex(filename, "."):])
	for _, ext := range imageExtensions {
		if ext == extension {
			return true
		}
	}

	return false
}
