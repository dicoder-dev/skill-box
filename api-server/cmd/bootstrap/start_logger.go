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
func StartGinLogger() {
	// 优先用 logger.SetLogPath 设过的路径(桌面端 ~/.<AppName>/logs/);否则兜底 ./logs/。
	logDir := logger.GetLogPath()
	if logDir == "" {
		logDir = "logs/"
	} else if !strings.HasSuffix(logDir, "/") {
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
