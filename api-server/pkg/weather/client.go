package weather

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// 支持的每日天气预报天数常量
// 和风天气 API 支持多种预报时长
const (
	Daily3D  = "3d"  // 3天预报
	Daily7D  = "7d"  // 7天预报
	Daily10D = "10d" // 10天预报
	Daily15D = "15d" // 15天预报
	Daily30D = "30d" // 30天预报
)

// 支持的逐小时天气预报时长常量
// 和风天气 API 支持多种小时预报时长
const (
	Hourly24H = "24h"  // 24小时预报
	Hourly72H = "72h"  // 72小时预报
	Hourly168 = "168h" // 168小时预报（7天）
)

// Client 和风天气 HTTP 客户端
// 用于调用和风天气 API 获取各类天气数据
type Client struct {
	config Config        // 客户端配置
	client *http.Client  // HTTP 客户端实例，用于发送请求
}

// NewClient 创建并初始化一个新的天气客户端
// 会自动设置超时时间并规范化配置
//
// 参数:
//   - cfg: 天气服务配置，包含 API 地址、Token 等信息
//
// 返回值:
//   - *Client: 初始化好的天气客户端指针
func NewClient(cfg Config) *Client {
	// 设置默认超时时间为 10 秒
	timeout := 10 * time.Second
	if cfg.Timeout > 0 {
		timeout = cfg.Timeout
	}

	return &Client{
		config: normalizeConfig(cfg), // 规范化配置（补全协议前缀、去除空格等）
		client: &http.Client{Timeout: timeout},
	}
}

// GetNow 获取实时天气数据
// 调用和风天气 /v7/weather/now 接口
//
// 参数:
//   - ctx: 上下文，用于控制请求的取消和超时
//   - query: 查询参数，必须包含 Location（经纬度坐标）
//
// 返回值:
//   - *NowResponse: 实时天气响应数据
//   - error: 请求失败时返回错误
func (c *Client) GetNow(ctx context.Context, query Query) (*NowResponse, error) {
	var resp NowResponse
	if err := c.get(ctx, "/v7/weather/now", query, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetDaily 获取每日天气预报
// 调用和风天气 /v7/weather/{days}d 接口，支持 3/7/10/15/30 天预报
//
// 参数:
//   - ctx: 上下文，用于控制请求的取消和超时
//   - days: 预报天数，支持 Daily3D、Daily7D、Daily10D、Daily15D、Daily30D
//   - query: 查询参数，必须包含 Location（经纬度坐标）
//
// 返回值:
//   - *DailyResponse: 每日天气预报响应数据
//   - error: 请求失败或不支持的天数时返回错误
func (c *Client) GetDaily(ctx context.Context, days string, query Query) (*DailyResponse, error) {
	// 检查是否支持请求的天数
	if !isSupportedDaily(days) {
		return nil, fmt.Errorf("不支持的每日天气天数: %s", days)
	}

	var resp DailyResponse
	if err := c.get(ctx, "/v7/weather/"+days, query, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetHourly 获取逐小时天气预报
// 调用和风天气 /v7/weather/{hours}h 接口，支持 24/72/168 小时预报
//
// 参数:
//   - ctx: 上下文，用于控制请求的取消和超时
//   - hours: 预报时长，支持 Hourly24H、Hourly72H、Hourly168
//   - query: 查询参数，必须包含 Location（经纬度坐标）
//
// 返回值:
//   - *HourlyResponse: 逐小时天气预报响应数据
//   - error: 请求失败或不支持的时长时返回错误
func (c *Client) GetHourly(ctx context.Context, hours string, query Query) (*HourlyResponse, error) {
	// 检查是否支持请求的时长
	if !isSupportedHourly(hours) {
		return nil, fmt.Errorf("不支持的逐小时天气时长: %s", hours)
	}

	var resp HourlyResponse
	if err := c.get(ctx, "/v7/weather/"+hours, query, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetBundleByCoordinates 通过经纬度坐标批量获取天气数据
// 一次性获取实时天气、三日预报、24小时预报，减少 API 调用次数
//
// 参数:
//   - ctx: 上下文，用于控制请求的取消和超时
//   - longitude: 经度，如 120.1234
//   - latitude: 纬度，如 30.5678
//
// 返回值:
//   - *WeatherBundle: 聚合的天气数据包
//   - error: 任一请求失败时返回错误
func (c *Client) GetBundleByCoordinates(ctx context.Context, longitude, latitude float64) (*WeatherBundle, error) {
	location := FormatCoordinateLocation(longitude, latitude)
	query := Query{Location: location}

	// 并行调用三个接口获取完整天气数据
	nowResp, err := c.GetNow(ctx, query)
	if err != nil {
		return nil, err
	}

	dailyResp, err := c.GetDaily(ctx, Daily3D, query)
	if err != nil {
		return nil, err
	}

	hourlyResp, err := c.GetHourly(ctx, Hourly24H, query)
	if err != nil {
		return nil, err
	}

	// 组装聚合天气包
	bundle := &WeatherBundle{
		Location: location,
		Now:      nowResp,
		ThreeDay: dailyResp.Daily,
		Hourly:   hourlyResp.Hourly,
	}
	// 单独保存今日数据，便于前端直接使用
	if len(dailyResp.Daily) > 0 {
		today := dailyResp.Daily[0]
		bundle.Today = &today
	}

	return bundle, nil
}

// FormatCoordinateLocation 格式化经纬度坐标为和风天气 API 要求的字符串格式
// 将浮点数转换为 "经度,纬度" 格式，保留两位小数
//
// 参数:
//   - longitude: 经度
//   - latitude: 纬度
//
// 返回值:
//   - string: 格式化后的坐标字符串，如 "120.12,30.57"
func FormatCoordinateLocation(longitude, latitude float64) string {
	return fmt.Sprintf("%.2f,%.2f", longitude, latitude)
}

// get 发送 GET 请求到和风天气 API 的通用方法
// 处理参数构建、认证、响应解析和错误处理
//
// 参数:
//   - ctx: 上下文，用于控制请求的取消和超时
//   - path: API 路径，如 "/v7/weather/now"
//   - query: 查询参数
//   - out: 响应数据解析目标结构体指针
//
// 返回值:
//   - error: 请求失败、配置错误或 API 返回错误时返回错误
func (c *Client) get(ctx context.Context, path string, query Query, out interface{}) error {
	// 参数校验：检查必要配置是否齐全
	if strings.TrimSpace(c.config.Host) == "" {
		return fmt.Errorf("未配置和风天气 API Host")
	}
	if strings.TrimSpace(c.config.Token) == "" {
		return fmt.Errorf("未配置和风天气 API Token")
	}
	if strings.TrimSpace(query.Location) == "" {
		return fmt.Errorf("location 不能为空")
	}

	// 构建查询参数
	params := url.Values{}
	params.Set("location", strings.TrimSpace(query.Location))

	// 处理语言设置：优先使用查询参数中的设置，否则使用全局配置
	lang := strings.TrimSpace(query.Lang)
	if lang == "" {
		lang = c.config.Lang
	}
	if lang != "" {
		params.Set("lang", lang)
	}

	// 处理温度单位设置：优先使用查询参数中的设置，否则使用全局配置
	unit := strings.TrimSpace(query.Unit)
	if unit == "" {
		unit = c.config.Unit
	}
	if unit != "" {
		params.Set("unit", unit)
	}

	// 拼接完整 URL
	fullURL := c.config.Host + path + "?" + params.Encode()

	// 创建 HTTP 请求，使用上下文控制超时和取消
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return err
	}

	// 设置请求头
	req.Header.Set("Authorization", "Bearer "+c.config.Token) // Bearer Token 认证
	req.Header.Set("Accept", "application/json")                // 期望返回 JSON 格式

	// 发送请求
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// 检查 HTTP 状态码
	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("和风天气请求失败: status=%d body=%s", resp.StatusCode, string(body))
	}

	// 解析 JSON 响应到目标结构体
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("解析和风天气响应失败: %w, body=%s", err, string(body))
	}

	// 检查业务错误码（和风天气使用 code 字段表示业务状态）
	if meta, ok := out.(interface{ getCode() string }); ok {
		if code := meta.getCode(); code != "" && code != "200" {
			return fmt.Errorf("和风天气返回错误: code=%s", code)
		}
	}

	return nil
}

// normalizeConfig 规范化配置参数
// 确保 Host 有协议前缀、去除多余空格等
//
// 参数:
//   - cfg: 原始配置
//
// 返回值:
//   - Config: 规范化后的配置
func normalizeConfig(cfg Config) Config {
	cfg.Host = strings.TrimSpace(cfg.Host)
	// 自动补全协议前缀
	if cfg.Host != "" && !strings.HasPrefix(cfg.Host, "http://") && !strings.HasPrefix(cfg.Host, "https://") {
		cfg.Host = "https://" + cfg.Host
	}
	// 去除末尾斜杠
	cfg.Host = strings.TrimRight(cfg.Host, "/")
	cfg.Lang = strings.TrimSpace(cfg.Lang)
	cfg.Unit = strings.TrimSpace(cfg.Unit)
	cfg.Token = strings.TrimSpace(cfg.Token)
	return cfg
}

// isSupportedDaily 检查请求的天数是否被支持
//
// 参数:
//   - days: 预报天数常量
//
// 返回值:
//   - bool: 是否支持
func isSupportedDaily(days string) bool {
	switch days {
	case Daily3D, Daily7D, Daily10D, Daily15D, Daily30D:
		return true
	default:
		return false
	}
}

// isSupportedHourly 检查请求的时长是否被支持
//
// 参数:
//   - hours: 预报时长常量
//
// 返回值:
//   - bool: 是否支持
func isSupportedHourly(hours string) bool {
	switch hours {
	case Hourly24H, Hourly72H, Hourly168:
		return true
	default:
		return false
	}
}

// getCode 实现 ResponseMeta 接口，用于获取业务状态码
// 让 get 方法能够统一检查各种响应的业务状态
func (r *NowResponse) getCode() string    { return r.Code }
func (r *DailyResponse) getCode() string  { return r.Code }
func (r *HourlyResponse) getCode() string { return r.Code }
