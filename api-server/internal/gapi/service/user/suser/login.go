package suser

import (
	"errors"
	"fmt"
	"ginp-api/internal/gapi/entity"
	"ginp-api/internal/gapi/model/user/muser"

	"ginp-api/pkg/where"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

type MyClaims struct {
	Id                   any `json:"id"`
	Username             any `json:"username"`
	jwt.RegisteredClaims     //官方自带字段
}

const secretKey = "https://github.com/dicoder-cn/ginpapi"

// JwtGenerateToken 生成token
// uid 用户id
// username 用户名
// timeoutDay 过期时间,单位天
// return token, error
func GenerateLoginToken(uid uint, username string, timeoutDay uint) (string, error) {
	//保存元数据信息
	myClaims := MyClaims{
		Id:       uid,
		Username: username,
	}
	//系统自带的payload
	sysClaims := jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute * time.Duration(60*24*timeoutDay))), // 过期时间
		IssuedAt:  jwt.NewNumericDate(time.Now()),                                                    // 签发时间
		NotBefore: jwt.NewNumericDate(time.Now()),                                                    // 生效时间
	}
	// 设置Payload数据
	myClaims.RegisteredClaims = sysClaims

	// 使用HS256加密算法创建Token对象
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, myClaims)

	// 生成签名字符串
	signToken, err := token.SignedString([]byte(secretKey))
	if err != nil {
		return "", err
	}
	return signToken, nil
}

// 解析api接口携带的token
func ParseToken(tokenString string) (*MyClaims, error) {
	// 使用指定的解析函数解析Token信息
	token, err := jwt.ParseWithClaims(tokenString, &MyClaims{}, secretFunc())
	// 当解析出错时，根据错误类型进行不同的处理
	if err != nil {
		if ve, ok := err.(*jwt.ValidationError); ok {
			if ve.Errors&jwt.ValidationErrorMalformed != 0 {
				return nil, errors.New("that's not even a token")
			} else if ve.Errors&jwt.ValidationErrorExpired != 0 {
				return nil, errors.New("token is expired")
			} else if ve.Errors&jwt.ValidationErrorNotValidYet != 0 {
				return nil, errors.New("token not active yet")
			} else {
				return nil, errors.New("couldn't handle this token")
			}
		}
	}
	// 解析出了有效的Token数据时，返回解析结果
	if claims, ok := token.Claims.(*MyClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, errors.New("couldn't handle this token")
}

// secretFunc 定义获取密钥的函数
func secretFunc() jwt.Keyfunc {
	// 返回使用getSecretKey()获取到的密钥
	return func(token *jwt.Token) (interface{}, error) {
		return []byte(secretKey), nil
	}
}

// 注册账户
func Register(user *entity.User) (*entity.User, string, error) {
	model := Model()
	// 验证用户名是否已存在
	wheres := where.New(muser.FieldUsername, "=", user.Username)
	res, _ := model.FindOne(wheres.Conditions())
	if res != nil && res.ID > 0 {
		return nil, "", errors.New("username already exists " + res.Username)
	}

	// 验证邮箱是否已存在
	wheres = where.New(muser.FieldEmail, "=", user.Email)
	resInfoEmail, _ := model.FindOne(wheres.Conditions())
	if resInfoEmail != nil && resInfoEmail.ID > 0 {
		return nil, "", errors.New("email already exists " + resInfoEmail.Email)
	}
	// 创建账户 - 使用新的安全密码哈希算法
	hashedPassword, err := HashPassword(user.Password)
	if err != nil {
		return nil, "", fmt.Errorf("failed to hash password: %w", err)
	}
	user.Password = hashedPassword
	userInfo, err := model.Create(user)
	if err != nil {
		return nil, "", err
	}
	//创建token
	token, err := GenerateLoginToken(userInfo.ID, userInfo.Username, 1)
	if err != nil {
		return nil, "", err
	}
	return userInfo, token, nil
}

func LoginByEmail(email string, password string) (*entity.User, string, error) {
	// 验证用户名和密码
	model := Model()
	wheres := where.New(muser.FieldEmail, "=", email)
	userInfo, err := model.FindOne(wheres.Conditions())
	if err != nil {
		return nil, "", err
	}

	// 验证密码
	passwordValid, err := VerifyPassword(password, userInfo.Password)
	if err != nil {
		return nil, "", fmt.Errorf("password verification error: %w", err)
	}

	if !passwordValid {
		return nil, "", errors.New("email or password is incorrect")
	}

	// 生成Token
	token, err := GenerateLoginToken(userInfo.ID, userInfo.Username, 1)
	if err != nil {
		return nil, "", err
	}

	return userInfo, token, nil
}

// 通过用户名登录
func LoginByUsername(username string, password string) (*entity.User, string, error) {
	// 验证用户名和密码
	model := Model()
	wheres := where.New(muser.FieldUsername, "=", username)
	userInfo, err := model.FindOne(wheres.Conditions())
	if err != nil {
		return nil, "", err
	}

	// 验证密码
	passwordValid, err := VerifyPassword(password, userInfo.Password)
	if err != nil {
		return nil, "", fmt.Errorf("password verification error: %w", err)
	}

	if !passwordValid {
		return nil, "", errors.New("username or password is incorrect")
	}

	// 生成Token
	token, err := GenerateLoginToken(userInfo.ID, userInfo.Username, 1)
	if err != nil {
		return nil, "", err
	}

	return userInfo, token, nil
}
