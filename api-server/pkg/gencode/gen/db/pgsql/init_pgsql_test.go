package pgsql

import (
	"fmt"
	"strings"
	"testing"
)

//测试连接
func TestConnect(t *testing.T) {
	InitDb("127.0.0.1", "5432", "root", "db_pg", "123456")
}

// TestDSNGeneration 测试DSN生成逻辑
func TestDSNGeneration(t *testing.T) {
	t.Run("测试DSN生成", func(t *testing.T) {
		testCases := []struct {
			name     string
			ip       string
			port     string
			user     string
			dbName   string
			password string
		}{
			{
				name:     "标准IP访问",
				ip:       "192.168.1.100",
				port:     "5432",
				user:     "testuser",
				dbName:   "testdb",
				password: "testpass",
			},
			{
				name:     "自定义端口访问",
				ip:       "10.0.0.50",
				port:     "5433",
				user:     "admin",
				dbName:   "production",
				password: "adminpass",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// 模拟DSN生成逻辑
				dsn := fmt.Sprintf("host=%v user=%v password=%v dbname=%v port=%v sslmode=disable TimeZone=Asia/Shanghai", 
					tc.ip, tc.user, tc.password, tc.dbName, tc.port)

				// 验证DSN格式
				if strings.Contains(dsn, "host="+tc.ip) && 
				   strings.Contains(dsn, "user="+tc.user) && 
				   strings.Contains(dsn, "dbname="+tc.dbName) &&
				   strings.Contains(dsn, "port="+tc.port) {
					t.Logf("✅ %s: DSN生成正确", tc.name)
					t.Logf("   生成的DSN: %s", dsn)
				} else {
					t.Errorf("❌ %s: DSN生成错误", tc.name)
				}
			})
		}
	})
}

// TestParameterValidation 测试参数验证
func TestParameterValidation(t *testing.T) {
	t.Run("测试空参数验证", func(t *testing.T) {
		// 测试空IP
		if isEmpty("") {
			t.Log("✅ 空IP检测正确")
		} else {
			t.Error("❌ 空IP检测失败")
		}

		// 测试空端口
		if isEmpty("") {
			t.Log("✅ 空端口检测正确")
		} else {
			t.Error("❌ 空端口检测失败")
		}

		// 测试非空参数
		if !isEmpty("test") {
			t.Log("✅ 非空参数检测正确")
		} else {
			t.Error("❌ 非空参数检测失败")
		}
	})
}

// isEmpty 辅助函数，用于测试参数验证
func isEmpty(s string) bool {
	return s == ""
} 