package httpclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"ginp-api/pkg/logger"
	"io"
	"net/http"
	"time"
)

const (
	maxRetries = 3
)

type PostParams struct {
	Url           string
	Data          map[string]any
	Header        map[string]string
	TimeoutSecond int //超时时间
}

func Post(params *PostParams, result any) error {

	// 检查参数是否为空
	if params == nil {
		logger.Errorf("post params cannot be nil")
		return fmt.Errorf("post params cannot be nil")
	}

	// 将数据编码为 JSON
	jsonData, err := json.Marshal(params.Data)
	if err != nil {
		logger.Errorf("failed to marshal data: %v", err)
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	for i := 0; i < maxRetries; i++ {
		// 创建 HTTP POST 请求
		req, err := http.NewRequest("POST", params.Url, bytes.NewBuffer(jsonData))
		if err != nil {
			logger.Errorf("failed to create request: %v", err)
			return fmt.Errorf("failed to create request: %w", err)
		}

		// 设置请求头
		for key, value := range params.Header {
			req.Header.Set(key, value)
		}

		if req.Header.Get("Content-Type") == "" {
			req.Header.Set("Content-Type", "application/json")
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
			if i < maxRetries-1 {
				time.Sleep(time.Duration(i+1) * time.Second)
				continue
			}
			return fmt.Errorf("failed to send request after %d attempts: %w", maxRetries, err)
		}
		defer resp.Body.Close()

		// 检查响应状态码
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			body, _ := io.ReadAll(resp.Body)
			logger.Errorf("unexpected status code: %d, response: %s", resp.StatusCode, string(body))
			if i < maxRetries-1 {
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
		// utils.
		if err := json.Unmarshal(body, result); err != nil {
			logger.Errorf("failed to unmarshal response: %v", err)
			return fmt.Errorf("failed to unmarshal response: %w", err)
		}

		logger.Info("POST request succeeded")
		return nil
	}

	return fmt.Errorf("all %d attempts failed", maxRetries)
}
