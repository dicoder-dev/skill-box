package push

// 华为推送类型定义文件
// 此文件用于存放华为推送相关的类型定义和常量

// 华为推送消息类型常量
const (
	// 资讯营销类
	// CategoryMarketing 资讯营销类消息
	// 用于新闻、内容推荐、社交动态、产品促销、财经动态、生活资讯、调研、功能推荐、运营活动
	// 若仅需发送MARKETING消息，则无需申请通知消息自分类权益
	CategoryMarketing = "MARKETING"

	// 服务与通讯类（需要申请通知消息自分类权益）
	// CategoryIM 即时聊天
	// 用于聊天、社交互动等场景
	CategoryIM = "IM"

	// CategoryVoIP 语音通话邀请、视频通话邀请
	// 用于音视频通话场景
	CategoryVoIP = "VOIP"

	// CategoryMissCall 未接通话消息提醒
	// 用于未接通话的消息提醒
	CategoryMissCall = "MISS_CALL"

	// CategorySubscription 订阅
	// 用于订阅相关消息
	CategorySubscription = "SUBSCRIPTION"

	// CategoryTravel 出行
	// 用于出行相关消息
	CategoryTravel = "TRAVEL"

	// CategoryHealth 健康
	// 用于健康相关消息
	CategoryHealth = "HEALTH"

	// CategoryWork 工作事项提醒
	// 用于工作相关提醒消息
	CategoryWork = "WORK"

	// CategoryAccount 账号动态
	// 用于账号相关动态消息
	CategoryAccount = "ACCOUNT"

	// CategoryExpress 订单&物流
	// 用于订单状态、物流信息等服务相关通知
	CategoryExpress = "EXPRESS"

	// CategoryFinance 财务
	// 用于财务相关消息
	CategoryFinance = "FINANCE"

	// CategoryDeviceReminder 设备提醒
	// 用于设备相关提醒消息
	CategoryDeviceReminder = "DEVICE_REMINDER"

	// CategoryMail 邮件
	// 用于邮件相关消息
	CategoryMail = "MAIL"

	// CategoryPlayVoice 语音播报
	// 用于语音播报消息，仅可发送push-type为2的通知扩展消息
	CategoryPlayVoice = "PLAY_VOICE"
)

// 华为推送消息优先级常量
const (
	// PriorityHigh 高优先级
	// 消息会优先展示，适用于重要通知
	PriorityHigh = "HIGH"

	// PriorityNormal 普通优先级（默认）
	// 标准优先级，适用于一般通知
	PriorityNormal = "NORMAL"

	// PriorityLow 低优先级
	// 消息展示优先级较低，适用于非紧急通知
	PriorityLow = "LOW"
)

// 华为推送点击动作类型常量
const (
	// ClickActionTypeOpenApp 点击通知打开应用首页
	ClickActionTypeOpenApp = 0

	// ClickActionTypeOpenURL 点击通知打开指定URL或应用内页面
	ClickActionTypeOpenURL = 1

	// ClickActionTypeOpenIntent 点击通知执行自定义Intent
	ClickActionTypeOpenIntent = 2

	// ClickActionTypeOpenRichResource 点击通知打开富媒体资源
	ClickActionTypeOpenRichResource = 3
)

// 华为推送消息紧急程度常量
const (
	// UrgencyHigh 高紧急程度
	// 消息会立即推送，适用于紧急通知
	UrgencyHigh = "HIGH"

	// UrgencyNormal 普通紧急程度（默认）
	// 标准推送速度
	UrgencyNormal = "NORMAL"
)

// 华为推送消息类型常量
const (
	// MessageTypeNotification 通知消息
	// 由Push Kit直接展示的通知消息
	MessageTypeNotification = "notification"

	// MessageTypeData 数据消息（透传消息）
	// 不会直接展示，需要应用自行处理
	MessageTypeData = "data"
)

// 华为推送通知重要性级别常量
const (
	// ImportanceDefault 默认重要性
	ImportanceDefault = "DEFAULT"

	// ImportanceLow 低重要性
	ImportanceLow = "LOW"

	// ImportanceNormal 普通重要性
	ImportanceNormal = "NORMAL"

	// ImportanceHigh 高重要性
	ImportanceHigh = "HIGH"
)

// 华为推送通知可见性常量
const (
	// VisibilityPublic 公开可见
	VisibilityPublic = "PUBLIC"

	// VisibilityPrivate 私有可见
	VisibilityPrivate = "PRIVATE"

	// VisibilitySecret 秘密可见
	VisibilitySecret = "SECRET"
)

// Android通知渠道重要性常量
const (
	// AndroidImportanceUnspecified 未指定重要性
	AndroidImportanceUnspecified = "UNSPECIFIED"

	// AndroidImportanceNone 无重要性
	AndroidImportanceNone = "NONE"

	// AndroidImportanceMin 最小重要性
	AndroidImportanceMin = "MIN"

	// AndroidImportanceLow 低重要性
	AndroidImportanceLow = "LOW"

	// AndroidImportanceDefault 默认重要性
	AndroidImportanceDefault = "DEFAULT"

	// AndroidImportanceHigh 高重要性
	AndroidImportanceHigh = "HIGH"

	// AndroidImportanceMax 最大重要性
	AndroidImportanceMax = "MAX"
)

// 测试类型常量
const (
	// TestTypeNone 非测试消息
	TestTypeNone = 0

	// TestTypeFormally 正式测试消息
	TestTypeFormally = 1

	// TestTypeTest 测试消息
	TestTypeTest = 2
)

// 收据类型常量
const (
	// ReceiptTypeNone 不返回收据
	ReceiptTypeNone = 0

	// ReceiptTypeOnClick 点击时返回收据
	ReceiptTypeOnClick = 1

	// ReceiptTypeOnArrival 到达时返回收据
	ReceiptTypeOnArrival = 2
)

// 快应用消息类型常量
const (
	// FastAppTargetTypePackageName 包名方式
	FastAppTargetTypePackageName = 1

	// FastAppTargetTypeDeepLink 深度链接方式
	FastAppTargetTypeDeepLink = 2
)

// 消息撤回类型常量
const (
	// RevokeTypeAll 撤回所有消息
	RevokeTypeAll = 0

	// RevokeTypeByMsgID 根据消息ID撤回
	RevokeTypeByMsgID = 1

	// RevokeTypeByCollapseID 根据折叠ID撤回
	RevokeTypeByCollapseID = 2
)

// 消息投递优先级常量
const (
	// DeliveryPriorityHigh 高优先级投递
	DeliveryPriorityHigh = "HIGH"

	// DeliveryPriorityNormal 普通优先级投递
	DeliveryPriorityNormal = "NORMAL"
)

// 多媒体消息类型常量
const (
	// MultiMediaTypeImage 图片类型
	MultiMediaTypeImage = 1

	// MultiMediaTypeAudio 音频类型
	MultiMediaTypeAudio = 2

	// MultiMediaTypeVideo 视频类型
	MultiMediaTypeVideo = 3
)

// 角标操作类型常量
const (
	// BadgeOperationAdd 增加角标
	BadgeOperationAdd = 1

	// BadgeOperationSet 设置角标
	BadgeOperationSet = 2
)

// HuaweiPushConfig 华为推送配置
type HuaweiPushConfig struct {
	ProjectID   string // 项目ID
	AccessToken string // JWT格式的访问令牌
	BaseURL     string // 推送服务基础URL，默认为华为云推送服务地址
}

// NotificationMessage 通知消息结构
type NotificationMessage struct {
	Category       string         `json:"category"`                 // 消息类型分类，使用Category*常量
	Title          string         `json:"title"`                    // 通知标题
	Body           string         `json:"body"`                     // 通知内容
	ClickAction    ClickAction    `json:"clickAction"`              // 点击行为
	ForegroundShow bool           `json:"foregroundShow"`           // 前台是否展示
	NotifyID       int            `json:"notifyId,omitempty"`       // 自定义消息标识
	Image          string         `json:"image,omitempty"`          // 通知图片URL
	Icon           string         `json:"icon,omitempty"`           // 通知图标URL
	Color          string         `json:"color,omitempty"`          // 通知颜色
	Sound          string         `json:"sound,omitempty"`          // 通知声音
	DefaultSound   bool           `json:"defaultSound,omitempty"`   // 是否使用默认声音
	Vibrate        []int          `json:"vibrate,omitempty"`        // 震动模式
	DefaultVibrate bool           `json:"defaultVibrate,omitempty"` // 是否使用默认震动
	When           string         `json:"when,omitempty"`           // 通知时间
	Importance     string         `json:"importance,omitempty"`     // 重要性级别，使用Importance*常量
	Urgency        string         `json:"urgency,omitempty"`        // 紧急程度，使用Urgency*常量
	TTL            string         `json:"ttl,omitempty"`            // 消息存活时间
	BiTag          string         `json:"biTag,omitempty"`          // 消息标签
	Badge          *BadgeInfo     `json:"badge,omitempty"`          // 角标信息
	LightSettings  *LightSettings `json:"lightSettings,omitempty"`  // 呼吸灯设置
	Visibility     string         `json:"visibility,omitempty"`     // 可见性，使用Visibility*常量
	ChannelID      string         `json:"channelId,omitempty"`      // 通知渠道ID
	Group          string         `json:"group,omitempty"`          // 通知分组
	Tag            string         `json:"tag,omitempty"`            // 通知标签
	LocalOnly      bool           `json:"localOnly,omitempty"`      // 是否仅本地显示
	Ticker         string         `json:"ticker,omitempty"`         // 状态栏滚动文本
	AutoCancel     bool           `json:"autoCancel,omitempty"`     // 点击后是否自动取消
	Actions        []ActionButton `json:"actions,omitempty"`        // 操作按钮
}

// ClickAction 点击行为
type ClickAction struct {
	ActionType   int    `json:"actionType"`             // 点击动作类型，使用ClickActionType*常量
	Intent       string `json:"intent,omitempty"`       // Intent URI（当actionType为2时使用）
	URL          string `json:"url,omitempty"`          // 跳转URL（当actionType为1时使用）
	RichResource string `json:"richResource,omitempty"` // 富媒体资源（当actionType为3时使用）
	Action       string `json:"action,omitempty"`       // 自定义动作
}

// BadgeInfo 角标信息
type BadgeInfo struct {
	AddNum int    `json:"addNum,omitempty"` // 角标增加数量
	SetNum int    `json:"setNum,omitempty"` // 角标设置数量
	Class  string `json:"class,omitempty"`  // 应用入口Activity类全路径
}

// LightSettings 呼吸灯设置
type LightSettings struct {
	Color            string `json:"color,omitempty"`            // 呼吸灯颜色
	LightOnDuration  string `json:"lightOnDuration,omitempty"`  // 亮灯持续时间
	LightOffDuration string `json:"lightOffDuration,omitempty"` // 熄灯持续时间
}

// ActionButton 操作按钮
type ActionButton struct {
	Name       string `json:"name"`                 // 按钮名称
	ActionType int    `json:"actionType"`           // 按钮动作类型
	IntentType int    `json:"intentType,omitempty"` // Intent类型
	Intent     string `json:"intent,omitempty"`     // Intent URI
	Action     string `json:"action,omitempty"`     // 自定义动作
}

// DataMessage 数据消息结构
type DataMessage struct {
	Data        map[string]string `json:"data,omitempty"`        // 自定义数据
	CollapseKey string            `json:"collapseKey,omitempty"` // 消息折叠键
	Urgency     string            `json:"urgency,omitempty"`     // 紧急程度，使用Urgency*常量
	TTL         string            `json:"ttl,omitempty"`         // 消息存活时间
	BiTag       string            `json:"biTag,omitempty"`       // 消息标签
	ReceiptID   string            `json:"receiptId,omitempty"`   // 回执ID
}

// AndroidConfig Android平台配置
type AndroidConfig struct {
	CollapseKey   string               `json:"collapseKey,omitempty"`   // 消息折叠键
	Urgency       string               `json:"urgency,omitempty"`       // 紧急程度
	TTL           string               `json:"ttl,omitempty"`           // 消息存活时间
	BiTag         string               `json:"biTag,omitempty"`         // 消息标签
	FastAppTarget int                  `json:"fastAppTarget,omitempty"` // 快应用目标类型
	Data          string               `json:"data,omitempty"`          // 透传消息内容
	Notification  *AndroidNotification `json:"notification,omitempty"`  // Android通知消息
}

// AndroidNotification Android通知消息
type AndroidNotification struct {
	Title             string            `json:"title,omitempty"`             // 通知标题
	Body              string            `json:"body,omitempty"`              // 通知内容
	Icon              string            `json:"icon,omitempty"`              // 通知图标
	Color             string            `json:"color,omitempty"`             // 通知颜色
	Sound             string            `json:"sound,omitempty"`             // 通知声音
	DefaultSound      bool              `json:"defaultSound,omitempty"`      // 是否使用默认声音
	Tag               string            `json:"tag,omitempty"`               // 通知标签
	ClickAction       *ClickAction      `json:"clickAction,omitempty"`       // 点击行为
	BodyLocKey        string            `json:"bodyLocKey,omitempty"`        // 消息内容本地化键
	BodyLocArgs       []string          `json:"bodyLocArgs,omitempty"`       // 消息内容本地化参数
	TitleLocKey       string            `json:"titleLocKey,omitempty"`       // 标题本地化键
	TitleLocArgs      []string          `json:"titleLocArgs,omitempty"`      // 标题本地化参数
	MultiLangKey      map[string]string `json:"multiLangKey,omitempty"`      // 多语言键值对
	ChannelID         string            `json:"channelId,omitempty"`         // 通知渠道ID
	NotifySummary     string            `json:"notifySummary,omitempty"`     // 通知摘要
	Image             string            `json:"image,omitempty"`             // 通知图片
	Style             int               `json:"style,omitempty"`             // 通知样式
	BigTitle          string            `json:"bigTitle,omitempty"`          // 大标题
	BigBody           string            `json:"bigBody,omitempty"`           // 大内容
	AutoCancel        bool              `json:"autoCancel,omitempty"`        // 点击后是否自动取消
	NotifyID          int               `json:"notifyId,omitempty"`          // 通知ID
	Group             string            `json:"group,omitempty"`             // 通知分组
	Badge             *BadgeInfo        `json:"badge,omitempty"`             // 角标信息
	Ticker            string            `json:"ticker,omitempty"`            // 状态栏滚动文本
	When              string            `json:"when,omitempty"`              // 通知时间
	Importance        string            `json:"importance,omitempty"`        // 重要性级别
	UseDefaultVibrate bool              `json:"useDefaultVibrate,omitempty"` // 是否使用默认震动
	UseDefaultLight   bool              `json:"useDefaultLight,omitempty"`   // 是否使用默认呼吸灯
	VibrateConfig     []int             `json:"vibrateConfig,omitempty"`     // 震动配置
	Visibility        string            `json:"visibility,omitempty"`        // 可见性
	LightSettings     *LightSettings    `json:"lightSettings,omitempty"`     // 呼吸灯设置
	ForegroundShow    bool              `json:"foregroundShow,omitempty"`    // 前台是否展示
	InboxContent      []string          `json:"inboxContent,omitempty"`      // 收件箱内容
	Buttons           []ActionButton    `json:"buttons,omitempty"`           // 操作按钮
}

// PushTarget 推送目标
type PushTarget struct {
	Token []string `json:"token"` // Push Token列表
}

// PushOptions 推送选项
type PushOptions struct {
	TestMessage      bool   `json:"testMessage"`                // 是否为测试消息
	TTL              int    `json:"ttl"`                        // 消息缓存时间(秒)
	Priority         string `json:"priority,omitempty"`         // 消息优先级，使用Priority*常量
	Urgency          string `json:"urgency,omitempty"`          // 紧急程度，使用Urgency*常量
	CollapseKey      string `json:"collapseKey,omitempty"`      // 消息折叠键
	ReceiptID        string `json:"receiptId,omitempty"`        // 回执ID
	TestType         int    `json:"testType,omitempty"`         // 测试类型，使用TestType*常量
	ReceiptType      int    `json:"receiptType,omitempty"`      // 收据类型，使用ReceiptType*常量
	DeliveryPriority string `json:"deliveryPriority,omitempty"` // 投递优先级，使用DeliveryPriority*常量
	BiTag            string `json:"biTag,omitempty"`            // 消息标签
	FastAppTarget    int    `json:"fastAppTarget,omitempty"`    // 快应用目标类型，使用FastAppTargetType*常量
}

// PushPayload 推送载荷
type PushPayload struct {
	Notification NotificationMessage `json:"notification"`
}

// HuaweiPushRequest 华为推送请求
type HuaweiPushRequest struct {
	Payload     PushPayload `json:"payload"`
	Target      PushTarget  `json:"target"`
	PushOptions PushOptions `json:"pushOptions"`
}

// HuaweiPushResponse 华为推送响应
type HuaweiPushResponse struct {
	Code      string `json:"code"`
	Msg       string `json:"msg"`
	RequestID string `json:"requestId"`
}
