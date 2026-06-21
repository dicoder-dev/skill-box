package httpclient

import (
	"encoding/json"
	"fmt"
	"ginp-api/pkg/logger"
	"io"
	"net/http"
	"time"
)

type GetParams struct {
	Url           string
	Header        map[string]string
	TimeoutSecond int //超时时间
}

func Get(url string, data interface{}) error {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		logger.Errorf("GET request failed: %v", err)
		return fmt.Errorf("GET request failed: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态码
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		logger.Errorf("unexpected status code: %d, response: %s", resp.StatusCode, string(body))
		return fmt.Errorf("unexpected status code: %d, response: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Errorf("failed to read response body: %v", err)
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// 记录原始响应内容
	// logger.Info(fmt.Sprintf("GET response body: %s", string(body)))

	err = json.Unmarshal(body, data)
	if err != nil {
		logger.Errorf("failed to unmarshal response: %v, body: %s", err, string(body))
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// logger.Info("GET request succeeded")
	return nil
}

func GetWithHeaders(params *GetParams, data interface{}) error {
	// 检查参数是否为空
	if params == nil {
		logger.Errorf("get params cannot be nil")
		return fmt.Errorf("get params cannot be nil")
	}

	// 设置超时时间
	if params.TimeoutSecond <= 0 {
		params.TimeoutSecond = 30
	}

	client := &http.Client{
		Timeout: time.Duration(params.TimeoutSecond) * time.Second,
	}

	// 创建 HTTP GET 请求
	req, err := http.NewRequest("GET", params.Url, nil)
	if err != nil {
		logger.Errorf("failed to create request: %v", err)
		return fmt.Errorf("failed to create request: %w", err)
	}

	// 设置请求头
	for key, value := range params.Header {
		req.Header.Set(key, value)
	}

	// 发起请求
	resp, err := client.Do(req)
	if err != nil {
		logger.Errorf("GET request failed: %v", err)
		return fmt.Errorf("GET request failed: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态码
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		logger.Errorf("unexpected status code: %d, response: %s", resp.StatusCode, string(body))
		return fmt.Errorf("unexpected status code: %d, response: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Errorf("failed to read response body: %v", err)
		return fmt.Errorf("failed to read response body: %w", err)
	}

	err = json.Unmarshal(body, data)
	if err == nil {
		// 记录原始响应内容
		logger.Errorf("failed to unmarshal response: %v, body: %s", err, string(body))
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// logger.Info("GET request succeeded")
	return nil
}
