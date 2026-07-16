package bootstrap

import (
	"io"
	"log"
	"os"
	"strings"
	"time"

	"ginp-api/pkg/logger"

	"github.com/gin-gonic/gin"
)

// StartGinLogger 把 gin.DefaultWriter 和标准 log 同时输出到日志文件 + stdout。
//
// 文件用于事后排查,stdout 用于开发期在终端直接看请求日志。
// 重复打开当日文件,采用追加模式。
//
// 目录结构(2026-07-16 改):跟 ginp middleware 的请求日志对齐,业务 log 也按月份
// 分目录:
//
//	~/.skill-box/logs/
//	├── 2026-07/                  ← 月份目录(业务 log)
//	│   ├── 2026-07-16.log
//	│   └── ...
//	├── 2026-07/                  ← 月份目录(请求 log)
//	│   ├── 07-16-request.txt
//	│   └── ...
//
// 业务 log 跟请求 log 写在同一个月份目录(命名空间不同),用户和管理员一眼能看出
// 当月有哪些日志。两类日志都会被 main.go 的 cleanupOldLogs 在启动早期清理
// 超过 2 个月的旧月份目录。
func StartGinLogger() {
	// 优先用 logger.SetLogPath 设过的路径(桌面端 ~/.<AppName>/logs/);否则兜底 ./logs/。
	logBaseDir := logger.GetLogPath()
	if logBaseDir == "" {
		logBaseDir = "logs/"
	} else if !strings.HasSuffix(logBaseDir, "/") {
		logBaseDir += "/"
	}
	// 月份子目录:logs/YYYY-MM/
	logDir := logBaseDir + time.Now().Format("2006-01")
	if !strings.HasSuffix(logDir, "/") {
		logDir += "/"
	}
	err := os.MkdirAll(logDir, os.ModePerm)
	if err != nil {
		log.Fatalf("Failed to create log directory: %v", err)
	}

	// 设置日志文件名为当前日期
	logFile := logDir + time.Now().Format("2006-01-02") + ".log"

	// 创建日志文件,追加模式写入
	f, _ := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)

	// gin 自身的 DefaultWriter / DefaultErrorWriter 也一并 tee 到 stdout,
	// 这样 `wails3 task web` 跑开发模式时,终端能直接看到 [GIN] 访问日志。
	gin.DefaultWriter = io.MultiWriter(f, os.Stdout)
	gin.DefaultErrorWriter = io.MultiWriter(f, os.Stderr)

	log.SetOutput(io.MultiWriter(f, os.Stdout))
}
