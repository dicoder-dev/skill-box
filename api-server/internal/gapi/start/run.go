package start

// 程序入口
func Run() {
	startDB()
	startTask()      //启动定时任务
	startGinLogger() //启动日志
	startGinServer() //启动http服务
}
