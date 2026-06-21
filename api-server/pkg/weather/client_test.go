package weather

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

// TestClientGetBundleByCoordinates 测试通过经纬度坐标批量获取天气数据
// 验证 Client.GetBundleByCoordinates 方法能否正确获取并解析实时、三日、24小时天气数据
func TestClientGetBundleByCoordinates(t *testing.T) {
	var requests []string // 记录发出的请求，用于验证请求参数

	// 创建测试用客户端，配置自定义 Transport 来拦截 HTTP 请求
	client := NewClient(Config{
		Host:    "https://weather.test",
		Token:   "test-token",
		Lang:    "zh",
		Unit:    "m",
		Timeout: 5 * time.Second,
	})

	// 设置自定义 Transport，模拟 HTTP 响应
	client.client.Transport = roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		// 记录请求路径和查询参数
		requests = append(requests, r.URL.Path+"?"+r.URL.RawQuery)

		// 验证 Authorization 请求头
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Fatalf("unexpected authorization header: %s", got)
		}

		var body string
		// 根据不同路径返回模拟数据
		switch r.URL.Path {
		case "/v7/weather/now":
			// 实时天气模拟响应
			body = `{"code":"200","updateTime":"2026-04-19T10:00+08:00","fxLink":"x","now":{"obsTime":"2026-04-19T09:50+08:00","temp":"25","feelsLike":"26","icon":"100","text":"晴","wind360":"90","windDir":"东风","windScale":"2","windSpeed":"8","humidity":"50","precip":"0.0","pressure":"1008","vis":"20","cloud":"5","dew":"12"}}`
		case "/v7/weather/3d":
			// 三日天气预报模拟响应，包含完整的日出日落、月相、温度范围等信息
			body = `{"code":"200","updateTime":"2026-04-19T10:00+08:00","fxLink":"x","daily":[{"fxDate":"2026-04-19","sunrise":"05:30","sunset":"18:20","moonrise":"10:00","moonset":"23:00","moonPhase":"盈","moonPhaseIcon":"801","tempMax":"28","tempMin":"18","iconDay":"100","textDay":"晴","iconNight":"150","textNight":"晴","wind360Day":"90","windDirDay":"东风","windScaleDay":"2","windSpeedDay":"10","wind360Night":"80","windDirNight":"东北风","windScaleNight":"1","windSpeedNight":"6","humidity":"55","precip":"0.0","pressure":"1007","vis":"20","cloud":"10","uvIndex":"6"},{"fxDate":"2026-04-20"},{"fxDate":"2026-04-21"}]}`
		case "/v7/weather/24h":
			// 24小时逐小时预报模拟响应
			body = `{"code":"200","updateTime":"2026-04-19T10:00+08:00","fxLink":"x","hourly":[{"fxTime":"2026-04-19T11:00+08:00","temp":"26","icon":"100","text":"晴","wind360":"90","windDir":"东风","windScale":"2","windSpeed":"10","humidity":"48","pop":"0","precip":"0.0","pressure":"1008","cloud":"8","dew":"11"}]}`
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     make(http.Header),
		}, nil
	})

	// 调用被测试的方法
	resp, err := client.GetBundleByCoordinates(context.Background(), 120.1234, 30.5678)
	if err != nil {
		t.Fatalf("GetBundleByCoordinates failed: %v", err)
	}

	// 验证返回的坐标格式化正确（保留两位小数）
	if resp.Location != "120.12,30.57" {
		t.Fatalf("unexpected location: %s", resp.Location)
	}

	// 验证实时天气数据解析正确
	if resp.Now == nil || resp.Now.Now.Temp != "25" {
		t.Fatalf("unexpected now response: %+v", resp.Now)
	}

	// 验证今日天气数据正确提取
	if resp.Today == nil || resp.Today.FxDate != "2026-04-19" {
		t.Fatalf("unexpected today response: %+v", resp.Today)
	}

	// 验证三日预报返回完整数据
	if len(resp.ThreeDay) != 3 {
		t.Fatalf("unexpected three day count: %d", len(resp.ThreeDay))
	}

	// 验证24小时预报数据
	if len(resp.Hourly) != 1 || resp.Hourly[0].FxTime != "2026-04-19T11:00+08:00" {
		t.Fatalf("unexpected hourly response: %+v", resp.Hourly)
	}

	// 验证共发起了 3 次请求（实时、三日、24小时各一次）
	if len(requests) != 3 {
		t.Fatalf("unexpected request count: %d", len(requests))
	}
}

// TestClientRejectsUnsupportedRange 测试不支持的预报参数会被正确拒绝
// 验证 GetDaily 和 GetHourly 方法对非法参数的处理
func TestClientRejectsUnsupportedRange(t *testing.T) {
	client := NewClient(Config{Host: "https://example.com", Token: "token", Timeout: 5 * time.Second})

	// 测试不支持的天数会被拒绝（"2d" 不在支持列表中）
	if _, err := client.GetDaily(context.Background(), "2d", Query{Location: "120.12,30.57"}); err == nil {
		t.Fatal("expected daily range error")
	}

	// 测试不支持的时长会被拒绝（"12h" 不在支持列表中）
	if _, err := client.GetHourly(context.Background(), "12h", Query{Location: "120.12,30.57"}); err == nil {
		t.Fatal("expected hourly range error")
	}
}

// roundTripperFunc 类型别名，用于将函数转换为 http.RoundTripper 接口
// 这是 Go 语言中常见的适配器模式，用于测试时拦截 HTTP 请求
type roundTripperFunc func(*http.Request) (*http.Response, error)

// RoundTrip 实现 http.RoundTripper 接口
// 将函数调用转发为 HTTP 响应
func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}
