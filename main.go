package main

import (
	"embed"
	_ "embed"
	"log"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// Wails 使用 Go 的 `embed` 包将前端文件嵌入到二进制文件中。
// frontend/dist 文件夹中的任何文件都会被嵌入到二进制中，
// 并提供给前端访问。
// 更多信息请参见 https://pkg.go.dev/embed。

//go:embed all:frontend/dist
var assets embed.FS

func init() {
	// 注册一个数据类型为 string 的自定义事件。
	// 这一步不是必需的，但绑定生成器会识别已注册的事件，
	// 并为它们提供强类型的 JS/TS API。
	application.RegisterEvent[string]("time")
}

// main 函数作为应用程序的入口点。它会初始化应用程序、创建窗口，
// 并启动一个 goroutine，每秒发送一个基于时间的事件。随后运行应用程序，
// 并在出现错误时进行日志记录。
func main() {

	// 通过提供必要的选项来创建一个新的 Wa7ils 应用程序。
	// 'Name' 和 'Description' 用于设置应用程序的元数据。
	// 'Assets' 通过 'FS' 变量配置资源服务器，指向前端文件。
	// 'Bind' 是 Go 结构体实例的列表，前端可以访问这些实例的方法。
	// 'Mac' 选项用于在 macOS 上运行时定制应用程序。
	app := application.New(application.Options{
		Name:        "skill-box",
		Description: "A demo of using raw HTML & CSS",
		Services: []application.Service{
			application.NewService(&GreetService{}),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	// 使用必要的选项创建一个新窗口。
	// 'Title' 是窗口的标题。
	// 'Mac' 选项用于在 macOS 上运行时定制窗口。
	// 'BackgroundColour' 是窗口的背景颜色。
	// 'URL' 是将在 webview 中加载的地址。
	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title: "Window 1",
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 50,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInset,
		},
		BackgroundColour: application.NewRGB(27, 38, 54),
		URL:              "/",
	})

	// 创建一个 goroutine，每秒发送一个包含当前时间的事件。
	// 前端可以监听此事件并相应地更新 UI。
	go func() {
		for {
			now := time.Now().Format(time.RFC1123)
			app.Event.Emit("time", now)
			time.Sleep(time.Second)
		}
	}()

	// 运行应用程序。此调用会一直阻塞，直到应用程序退出。
	err := app.Run()

	// 如果运行应用程序时发生错误，记录该错误并退出。
	if err != nil {
		log.Fatal(err)
	}
}
