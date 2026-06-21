package maps

// =============================================================================
// 地图服务接口定义
// 本文件定义了所有地图服务提供商需要实现的统一接口和数据结构
// 支持的功能包括：地点搜索、线路规划、地理编码、逆地理编码、行政区划查询
// =============================================================================

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// =============================================================================
// 基础Provider结构体
// 所有地图服务提供商的基类，包含通用的配置和HTTP客户端
// =============================================================================

// baseProvider 基础地图服务提供商结构体
// 封装了所有提供商共有的配置和HTTP客户端功能
type baseProvider struct {
	name    ProviderName  // 提供商名称（如baidu、amap、google等）
	config  ProviderConfig // 提供商配置（API密钥、密钥、基础URL等）
	client  *http.Client  // HTTP客户端，用于发送请求
	baseURL string      // API基础URL
}

// newBaseProvider 创建基础Provider实例
// 参数说明：
//   - name: 提供商名称
//   - cfg: 提供商配置信息
//   - timeout: HTTP请求超时时间
//   - defaultBaseURL: 默认的API基础URL（当配置中未指定时使用）
func newBaseProvider(name ProviderName, cfg ProviderConfig, timeout time.Duration, defaultBaseURL string) baseProvider {
	baseURL := strings.TrimRight(cfg.BaseURL, "/")
	if baseURL == "" {
		baseURL = strings.TrimRight(defaultBaseURL, "/")
	}
	return baseProvider{
		name:    name,
		config:  cfg,
		baseURL: baseURL,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

// Name 获取提供商名称
func (b *baseProvider) Name() ProviderName { return b.name }

// IsAvailable 检查提供商是否可用（需要配置了有效的API密钥）
func (b *baseProvider) IsAvailable() bool  { return strings.TrimSpace(b.config.Key) != "" }

// =============================================================================
// HTTP请求辅助方法
// 提供GET/POST请求的JSON解析和字节数组获取功能
// =============================================================================

// getJSON 发送GET请求并将响应解析为JSON
// 参数说明：
//   - ctx: 上下文，用于控制请求取消
//   - path: 请求路径
//   - params: URL查询参数
//   - out: 解析结果的目标结构体
func (b *baseProvider) getJSON(ctx context.Context, path string, params url.Values, out interface{}) error {
	body, err := b.getBytes(ctx, path, params)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("解析响应失败: %w, body=%s", err, string(body))
	}
	return nil
}

// postJSON 发送POST请求并携带JSON载荷，然后将响应解析为JSON
// 参数说明：
//   - ctx: 上下文
//   - path: 请求路径
//   - params: URL查询参数
//   - payload: 请求载荷（将被序列化为JSON）
//   - out: 解析结果的目标结构体
func (b *baseProvider) postJSON(ctx context.Context, path string, params url.Values, payload interface{}, out interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("序列化请求失败: %w", err)
	}

	respBody, err := b.doBytes(ctx, http.MethodPost, path, params, body)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(respBody, out); err != nil {
		return fmt.Errorf("解析响应失败: %w, body=%s", err, string(respBody))
	}
	return nil
}

// getBytes 发送GET请求并获取响应字节数组
func (b *baseProvider) getBytes(ctx context.Context, path string, params url.Values) ([]byte, error) {
	return b.doBytes(ctx, http.MethodGet, path, params, nil)
}

// doBytes 发送HTTP请求并获取响应字节数组
// 参数说明：
//   - ctx: 上下文
//   - method: HTTP方法（GET/POST等）
//   - path: 请求路径
//   - params: URL查询参数
//   - body: 请求体字节数组（nil表示无请求体）
func (b *baseProvider) doBytes(ctx context.Context, method string, path string, params url.Values, body []byte) ([]byte, error) {
	fullURL := b.baseURL + path
	if len(params) > 0 {
		fullURL += "?" + params.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
	}

	resp, err := b.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("请求失败: status=%d body=%s", resp.StatusCode, string(respBody))
	}
	return respBody, nil
}

// =============================================================================
// 工具函数
// 提供坐标解析、默认值设置等辅助功能
// =============================================================================

// parseCoordinatePair 解析坐标字符串（格式：经度,纬度）
// 参数说明：
//   - value: 坐标字符串，格式如 "116.403988,39.914266"
// 返回值：
//   - lng: 经度
//   - lat: 纬度
//   - err: 解析错误
func parseCoordinatePair(value string) (lng, lat float64, err error) {
	parts := strings.Split(strings.TrimSpace(value), ",")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("坐标格式错误: %s", value)
	}
	lng, err = strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	if err != nil {
		return 0, 0, fmt.Errorf("解析经度失败: %w", err)
	}
	lat, err = strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	if err != nil {
		return 0, 0, fmt.Errorf("解析纬度失败: %w", err)
	}
	return lng, lat, nil
}

// mustMode 获取旅行模式，如果未指定则默认为驾车
func mustMode(req *RoutePlanningRequest) TravelMode {
	if req.Mode == "" {
		return TravelModeDriving
	}
	return req.Mode
}

// defaultPageSize 获取默认的分页大小
func defaultPageSize(size int) int {
	if size <= 0 {
		return 20
	}
	return size
}

// defaultPageIndex 获取默认的页码索引（从1开始）
func defaultPageIndex(index int) int {
	if index <= 0 {
		return 1
	}
	return index
}

// defaultRadius 获取默认的搜索半径（单位：米）
func defaultRadius(radius int) int {
	if radius <= 0 {
		return 5000
	}
	return radius
}

// newUnsupportedResponse 创建不支持操作的错误信息
func newUnsupportedResponse(provider ProviderName, capability string) error {
	return fmt.Errorf("%s 暂不支持 %s", provider, capability)
}

// parseFloat 解析浮点数字符串
func parseFloat(value string) float64 {
	num, _ := strconv.ParseFloat(strings.TrimSpace(value), 64)
	return num
}

// parseInt64 解析64位整数字符串
func parseInt64(value string) int64 {
	num, _ := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	return num
}

// anyToString ���任��类型转换为字符串
func anyToString(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case []interface{}:
		if len(v) == 0 {
			return ""
		}
		return fmt.Sprint(v[0])
	default:
		return fmt.Sprint(v)
	}
}