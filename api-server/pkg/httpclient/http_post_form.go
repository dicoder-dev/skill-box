package httpclient

import (
	"encoding/json"
	"fmt"
	"ginp-api/pkg/logger"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	maxRetriesForm = 3
)

type PostFormParams struct {
	Url           string
	Data          map[string]string
	Header        map[string]string
	TimeoutSecond int //超时时间
}

func PostForm(params *PostFormParams, result any) error {
	// 检查参数是否为空
	if params == nil {
		logger.Errorf("post form params cannot be nil")
		return fmt.Errorf("post form params cannot be nil")
	}

	// 构建表单数据
	formData := url.Values{}
	for key, value := range params.Data {
		formData.Set(key, value)
	}

	for i := 0; i < maxRetriesForm; i++ {
		// 创建 HTTP POST 请求
		req, err := http.NewRequest("POST", params.Url, strings.NewReader(formData.Encode()))
		if err != nil {
			logger.Errorf("failed to create request: %v", err)
			return fmt.Errorf("failed to create request: %w", err)
		}

		// 设置请求头
		for key, value := range params.Header {
			req.Header.Set(key, value)
		}

		if req.Header.Get("Content-Type") == "" {
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}

		// 设置超时时间
		if params.TimeoutSecond <= 0 {
			params.TimeoutSecond = 20
		}

		client := &http.Client{
			Timeout: time.Duration(params.TimeoutSecond) * time.Second,
		}

		// 发起请求
		resp, err := client.Do(req)
		if err != nil {
			logger.Errorf("request attempt %d failed: %v", i+1, err)
			if i < maxRetriesForm-1 {
				time.Sleep(time.Duration(i+1) * time.Second)
				continue
			}
			return fmt.Errorf("failed to send request after %d attempts: %w", maxRetriesForm, err)
		}
		defer resp.Body.Close()

		// 检查响应状态码
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			body, _ := io.ReadAll(resp.Body)
			logger.Errorf("unexpected status code: %d, response: %s", resp.StatusCode, string(body))
			if i < maxRetriesForm-1 {
				time.Sleep(time.Duration(i+1) * time.Second)
				continue
			}
			return fmt.Errorf("unexpected status code: %d, response: %s", resp.StatusCode, string(body))
		}

		// 读取响应体并解析到 result 变量
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			logger.Errorf("failed to read response body: %v", err)
			return fmt.Errorf("failed to read response body: %w", err)
		}

		// 记录原始响应内容
		logger.Info(fmt.Sprintf("POST form response body: %s", string(body)))

		if err := json.Unmarshal(body, result); err != nil {
			logger.Errorf("failed to unmarshal response: %v, body: %s", err, string(body))
			return fmt.Errorf("failed to unmarshal response: %w", err)
		}

		// logger.Info("POST form request succeeded")
		return nil
	}

	return fmt.Errorf("all %d attempts failed", maxRetriesForm)
}
