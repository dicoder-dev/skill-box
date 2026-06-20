package start

import (
	"io"
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

func startGinLogger() {
	// 创建日志文件目录
	logDir := "logs/"
	err := os.MkdirAll(logDir, os.ModePerm)
	if err != nil {
		log.Fatalf("Failed to create log directory: %v", err)
	}

	// 设置日志文件名为当前日期
	logFile := logDir + time.Now().Format("2006-01-02") + ".log"

	// 创建日志文件，追加模式写入
	f, _ := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)

	// 设置日志输出到文件
	gin.DefaultWriter = io.MultiWriter(f)

	log.SetOutput(io.MultiWriter(f, os.Stdout))
}
