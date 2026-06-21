package weather

import "time"

// Config 和风天气客户端配置结构体
// 包含连接和风天气 API 所需的全部配置信息
type Config struct {
	Host    string        // API 接口地址，如 "https://devapi.qweather.com"
	Token   string        // API 认证令牌，从和风天气开发者平台获取
	Lang    string        // 返回数据语言代码，"zh" 表示中文，"en" 表示英文
	Unit    string        // 温度单位，"m" 表示摄氏度，"f" 表示华氏度
	Timeout time.Duration // HTTP 请求超时时间
}

// Query 天气查询请求参数结构体
// 用于指定要查询的地理位置和可选的返回格式
type Query struct {
	Location string // 位置坐标，格式为 "经度,纬度"，如 "120.12,30.57"
	Lang     string // 可选：覆盖默认语言设置
	Unit     string // 可选：覆盖默认温度单位
}

// Refer 数据来源引用信息
// 和风天气 API 的数据使用限制和许可信息
type Refer struct {
	Sources []string `json:"sources"` // 数据来源列表
	License []string `json:"license"` // 许可协议列表
}

// ResponseMeta 和风天气 API 响应元数据
// 所有天气响应都包含的公共信息
type ResponseMeta struct {
	Code       string `json:"code"`        // 响应状态码，"200" 表示成功
	UpdateTime string `json:"updateTime"`  // 数据更新时间，格式如 "2026-04-19T10:00+08:00"
	FxLink     string `json:"fxLink"`      // 和风天气链接
	Refer      Refer  `json:"refer"`      // 数据来源引用
}

// NowWeather 实时天气数据
// 包含当前时刻的温度、湿度、风力等详细信息
type NowWeather struct {
	ObsTime   string `json:"obsTime"`    // 数据观测时间
	Temp      string `json:"temp"`      // 温度（℃）
	FeelsLike string `json:"feelsLike"` // 体感温度（℃）
	Icon      string `json:"icon"`      // 天气图标代码
	Text      string `json:"text"`      // 天气文字描述，如"晴"、"多云"
	Wind360   string `json:"wind360"`    // 风向角度（0-360度）
	WindDir   string `json:"windDir"`   // 风向描述，如"东风"、"西北风"
	WindScale string `json:"windScale"` // 风力等级，如"2"
	WindSpeed string `json:"windSpeed"` // 风速（公里/小时）
	Humidity  string `json:"humidity"`  // 相对湿度（%）
	Precip    string `json:"precip"`    // 降水量（毫米）
	Pressure  string `json:"pressure"`  // 气压（百帕）
	Vis       string `json:"vis"`       // 能见度（公里）
	Cloud     string `json:"cloud"`     // 云量（%）
	Dew       string `json:"dew"`       // 露点温度（℃）
}

// DailyWeather 每日天气预报数据
// 包含日出日落、月相、每日温度范围等信息
type DailyWeather struct {
	FxDate         string `json:"fxDate"`         // 预报日期，格式 "2026-04-19"
	Sunrise        string `json:"sunrise"`        // 日出时间，格式 "05:30"
	Sunset         string `json:"sunset"`         // 日落时间，格式 "18:20"
	Moonrise       string `json:"moonrise"`       // 月升时间
	Moonset        string `json:"moonset"`        // 月落时间
	MoonPhase      string `json:"moonPhase"`      // 月相名称，如"盈"
	MoonPhaseIcon  string `json:"moonPhaseIcon"`  // 月相图标代码
	TempMax        string `json:"tempMax"`        // 最高温度（℃）
	TempMin        string `json:"tempMin"`        // 最低温度（℃）
	IconDay        string `json:"iconDay"`        // 白天天气图标代码
	TextDay        string `json:"textDay"`        // 白天天气文字描述
	IconNight      string `json:"iconNight"`      // 夜间天气图标代码
	TextNight      string `json:"textNight"`      // 夜间天气文字描述
	Wind360Day     string `json:"wind360Day"`     // 白天风向角度
	WindDirDay     string `json:"windDirDay"`     // 白天风向描述
	WindScaleDay   string `json:"windScaleDay"`   // 白天风力等级
	WindSpeedDay   string `json:"windSpeedDay"`   // 白天风速
	Wind360Night   string `json:"wind360Night"`   // 夜间风向角度
	WindDirNight   string `json:"windDirNight"`   // 夜间风向描述
	WindScaleNight string `json:"windScaleNight"` // 夜间风力等级
	WindSpeedNight string `json:"windSpeedNight"` // 夜间风速
	Humidity       string `json:"humidity"`       // 平均相对湿度（%）
	Precip         string `json:"precip"`         // 预报降水量（毫米）
	Pressure       string `json:"pressure"`       // 平均气压（百帕）
	Vis            string `json:"vis"`            // 平均能见度（公里）
	Cloud          string `json:"cloud"`          // 平均云量（%）
	UVIndex        string `json:"uvIndex"`         // 紫外线指数
}

// HourlyWeather 逐小时天气预报数据
// 包含未来每小时的天气信息
type HourlyWeather struct {
	FxTime    string `json:"fxTime"`    // 预报时间，格式 "2026-04-19T11:00+08:00"
	Temp      string `json:"temp"`      // 温度（℃）
	Icon      string `json:"icon"`      // 天气图标代码
	Text      string `json:"text"`      // 天气文字描述
	Wind360   string `json:"wind360"`   // 风向角度
	WindDir   string `json:"windDir"`   // 风向描述
	WindScale string `json:"windScale"` // 风力等级
	WindSpeed string `json:"windSpeed"` // 风速（公里/小时）
	Humidity  string `json:"humidity"` // 相对湿度（%）
	Pop       string `json:"pop"`       // 降水概率（%）
	Precip    string `json:"precip"`    // 降水量（毫米）
	Pressure  string `json:"pressure"`  // 气压（百帕）
	Cloud     string `json:"cloud"`     // 云量（%）
	Dew       string `json:"dew"`       // 露点温度（℃）
}

// NowResponse 实时天气 API 响应结构
// 对应和风天气 v7/weather/now 接口返回数据
type NowResponse struct {
	ResponseMeta                  // 嵌入响应元数据
	Now      NowWeather `json:"now"` // 实时天气数据
}

// DailyResponse 每日天气预报 API 响应结构
// 对应和风天气 v7/weather/{days}d 接口返回数据
type DailyResponse struct {
	ResponseMeta                    // 嵌入响应元数据
	Daily     []DailyWeather `json:"daily"` // 每日天气数据数组
}

// HourlyResponse 逐小时天气预报 API 响应结构
// 对应和风天气 v7/weather/{hours}h 接口返回数据
type HourlyResponse struct {
	ResponseMeta                    // 嵌入响应元数据
	Hourly    []HourlyWeather `json:"hourly"` // 每小时天气数据数组
}

// WeatherBundle 天气数据聚合包
// 将实时天气、三日预报、24小时预报打包返回，减少调用次数
type WeatherBundle struct {
	Location string           `json:"location"`            // 查询位置坐标
	Now      *NowResponse     `json:"now,omitempty"`      // 实时天气数据（可选）
	Today    *DailyWeather    `json:"today,omitempty"`    // 今日天气数据（可选）
	ThreeDay []DailyWeather   `json:"three_day,omitempty"` // 三日天气预报数组（可选）
	Hourly   []HourlyWeather  `json:"hourly,omitempty"`    // 24小时预报数组（可选）
}
