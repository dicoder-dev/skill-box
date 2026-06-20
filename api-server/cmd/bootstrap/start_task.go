package bootstrap

import "ginp-api/pkg/task"

// StartTask 启动定时任务调度器(在 goroutine 内运行,不阻塞)。
func StartTask() {
	go func() {
		taskManager := task.NewTaskManager()
		// mylog.Task("定时任务：等待5秒的倍数后执行...")
		// WaitNewMinStart() //等待下一分钟0秒才开始执行
		// mylog.Task("定时任务开始执行...")

		//每1秒执行一次
		spec1 := task.FormatEverySpace(0, 0, 100)
		taskManager.AddTask("every_10s", spec1, func() {
			//do something
			// mylog.Task("定时任务：每10秒执行一次...")
		})
	}()
}
