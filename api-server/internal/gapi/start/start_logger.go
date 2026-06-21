package start

import (
	"log"
	"os"
)

// startGinLogger 创建日志文件目录
func startGinLogger() {
	logDir := "logs/"
	err := os.MkdirAll(logDir, os.ModePerm)
	if err != nil {
		log.Fatalf("Failed to create log directory: %v", err)
	}
}
