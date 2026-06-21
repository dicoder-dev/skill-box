package httpclient

import (
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"ginp-api/pkg/logger"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

type RetryConfig struct {
	MaxRetries    int
	BaseDelay     time.Duration
	MaxDelay      time.Duration
	BackoffFactor float64
}

var DefaultRetryConfig = RetryConfig{
	MaxRetries:    3,
	BaseDelay:     2 * time.Second,
	MaxDelay:      30 * time.Second,
	BackoffFactor: 2,
}

type RetryableError struct {
	StatusCode int
	Message    string
}

func (e *RetryableError) Error() string {
	return fmt.Sprintf("retryable error: status %d, message: %s", e.StatusCode, e.Message)
}

func GetOriginWithRetry(url string, userAgent string, config RetryConfig) (string, error) {
	return GetOriginWithRetryAndReferer(url, userAgent, "", config)
}

func GetOriginWithRetryAndReferer(url string, userAgent string, referer string, config RetryConfig) (string, error) {
	var lastErr error
	for attempt := 1; attempt <= config.MaxRetries; attempt++ {
		if attempt > 1 {
			time.Sleep(calculateDelay(config, attempt))
		}

		result, err := getOriginSingleWithReferer(url, userAgent, referer)
		if err == nil {
			return result, nil
		}

		lastErr = err
		if !isRetryableError(err) || attempt == config.MaxRetries {
			break
		}
	}

	return "", fmt.Errorf("request failed after %d retries: %w", config.MaxRetries, lastErr)
}

func isRetryableError(err error) bool {
	retryErr, ok := err.(*RetryableError)
	if !ok {
		return false
	}
	switch retryErr.StatusCode {
	case http.StatusTooManyRequests, http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}

func calculateDelay(config RetryConfig, attempt int) time.Duration {
	delay := time.Duration(float64(config.BaseDelay) * pow(config.BackoffFactor, float64(attempt-1)))
	if delay > config.MaxDelay {
		return config.MaxDelay
	}
	return delay
}

func pow(base, exp float64) float64 {
	result := 1.0
	for i := 0; i < int(exp); i++ {
		result *= base
	}
	return result
}

func getOriginSingleWithReferer(url string, userAgent string, referer string) (string, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	if referer != "" {
		req.Header.Set("Referer", referer)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("GET request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return "", &RetryableError{StatusCode: resp.StatusCode, Message: string(body)}
	}

	var reader io.Reader = resp.Body
	if resp.Header.Get("Content-Encoding") == "gzip" {
		gzipReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return "", fmt.Errorf("failed to create gzip reader: %w", err)
		}
		defer gzipReader.Close()
		reader = gzipReader
	}

	body, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	contentType := strings.ToLower(resp.Header.Get("Content-Type"))
	bodyStr := string(body)
	if strings.Contains(contentType, "gbk") ||
		strings.Contains(strings.ToLower(bodyStr), "charset=gbk") ||
		strings.Contains(strings.ToLower(bodyStr), "charset=gb2312") {
		decoder := simplifiedchinese.GBK.NewDecoder()
		utf8Body, err := io.ReadAll(transform.NewReader(strings.NewReader(bodyStr), decoder))
		if err != nil {
			logger.Warn("failed to convert GBK to UTF-8: %v", err)
			return bodyStr, nil
		}
		bodyStr = string(utf8Body)
	}

	return bodyStr, nil
}
