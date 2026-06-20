package entity

import (
	"ginp-api/internal/gapi/typ"
	"ginp-api/internal/gen"
	"time"
)

const tableNameUser = "users"

type User struct {
	ID              uint      `json:"id,omitempty"`
	AvatarUrl       string    `gorm:"type:varchar(100);" json:"avatar_url,omitempty"`
	WechatOpenId    string    `gorm:"column:wechat_open_id;comment:微信公众号openid" json:"wechat_open_id,omitempty"`
	MiniAppOpenId   string    `gorm:"column:mini_app_open_id;comment:小程序openid" json:"mini_app_open_id,omitempty"`
	Username        string    `gorm:"column:username;comment:用户名;" json:"username,omitempty"`
	Email           string    `gorm:"column:email;comment:邮箱" json:"email,omitempty"`
	Password        string    `gorm:"column:password;comment:密码" json:"password,omitempty"`
	Status          uint8     `gorm:"default:1;column:status;comment:状态" json:"status,omitempty"`
	Phone           string    `gorm:"column:phone;comment:手机号" json:"phone,omitempty"`
	Points          uint      `gorm:"default:0;column:points;comment:积分" json:"points,omitempty"` //积分
	HuaweiPushToken string    `gorm:"column:huawei_push_token;comment:华为推送token" json:"huawei_push_token,omitempty"`
	VipEndAt        time.Time `gorm:"column:vip_end_at;comment:vip到期时间;default:'1970-01-01 00:00:00'" json:"vip_end_at,omitempty"`
	VipPermanent    bool      `gorm:"column:vip_permanent;comment:是否为永久VIP" json:"vip_permanent,omitempty"`
	IsCancel        bool      `gorm:"column:is_cancel;comment:是否已注销" json:"is_cancel,omitempty"` //软注销
	CreatedAt       time.Time `json:"created_at,omitempty"`
	UpdatedAt       time.Time `json:"updated_at,omitempty"`
}

var _ typ.IEntity = (*User)(nil) // U实体必须实现接口GenConfig

func (User) GenConfig() *gen.EntityConfig {
	return &gen.EntityConfig{
		TableName: tableNameUser,
	}
}

func (User) TableName() string {
	return tableNameUser
}
