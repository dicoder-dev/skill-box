package desktop

import (
	"bytes"
	_ "embed"
	"image"
	imgcolor "image/color"
	"image/png"
	"log"
	"runtime"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"golang.org/x/image/draw"
)

// trayAppIconPNG 是 build/appicon.png(1024×1024 源图)的二进制内嵌;
// 用符号链接 desktop/appicon.png 指向 ../build/appicon.png 后,这里用
// //go:embed 把图标带进可执行文件,运行时不再依赖磁盘路径。
//
//go:embed appicon.png
var trayAppIconPNG []byte

// 托盘图标尺寸。
//   - darwin 模板图标 36×36 覆盖 @2x(系统会自己画到 22pt 菜单栏 backing 上,清晰)。
//   - Windows / Linux 走 SetIcon 彩色版,32×32 兼顾小尺寸 DPI。
const (
	trayTemplateSize = 36
	trayColorSize    = 32
)

// TrayManager 包装 Wails 系统托盘,统一通过它创建/管理托盘。
//
// 菜单项:显示主窗口 / 测试通知 / 偏好设置 / 关于 / 退出。
// "测试通知" / "偏好设置" 是 V1 新增;前者调 notifier.Notify,后者跳到
// /settings/desktop 路由(SettingsView 末尾的桌面端 section)。
type TrayManager struct {
	tray     *application.SystemTray
	notifier *Notifier
}

// TrayCallbacks 托盘菜单回调。onShow / onQuit 必须;onOpenSettings 可选。
type TrayCallbacks struct {
	OnShow         func()
	OnQuit         func()
	OnOpenSettings func()
}

// NewTrayManager 构造并创建托盘。
//
// notifier 可为 nil(测试菜单项 no-op);backend 也不强依赖,只是偏好设置菜单
// 跳转需要它。
//
// 跨平台行为:
//   - darwin: 左键单击 = 弹出托盘菜单(SetTemplateIcon 单色图标,系统会随
//     Light/Dark 模式自动反色);SetLabel 留空避免图标+文字并存。
//   - windows / linux: 左键单击 = 显示主窗口,右键 = 弹出菜单(SetIcon 彩色)。
//
// 为什么不绑 OnDoubleClick: Wails v3 alpha.60 的 darwin 路径 processClick
// 根本不 dispatch doubleClickHandler(processClick 只 case left/right
// button),留着 OnDoubleClick = 误导性的死代码;Windows/Linux 下双击
// 体验上与单击同义(OnShow),绑了也看不出区别。
func NewTrayManager(app *application.App, cb TrayCallbacks, notifier *Notifier) *TrayManager {
	t := app.SystemTray.New()

	// 跨平台图标 + label
	tmpl, color, iconErr := generateTrayIcons()
	if iconErr != nil {
		// 图标生成失败时降级到文字 label,不阻塞启动。
		log.Printf("tray: 图标生成失败,降级到文字: %v", iconErr)
		t.SetLabel("Skill Box")
		t.SetTooltip("Skill Box")
	} else {
		switch runtime.GOOS {
		case "darwin":
			// macOS 上 SetTemplateIcon 与 SetIcon 互斥(systemtray_darwin.go
			// 的 setIcon 会读 isTemplateIcon 标记,两个都调会污染),所以只调
			// SetTemplateIcon;label 留空避免图标+文字挤在一起。
			t.SetTemplateIcon(tmpl)
			t.SetLabel("")
		default:
			// windows / linux
			t.SetIcon(color)
			t.SetLabel("Skill Box")
			t.SetTooltip("Skill Box")
		}
	}

	// 左键行为分流
	switch runtime.GOOS {
	case "darwin":
		t.OnClick(func() { t.OpenMenu() })
		// darwin 上右键默认 popUpMenu,无需再设 OnRightClick。
	default:
		t.OnClick(cb.OnShow)
		// Windows 上 OnRightClick 走 openMenu 真实弹菜单;Linux 上是 FIXME
		// no-op(见 systemtray_linux.go:441-444),但不影响左键主流程。
		t.OnRightClick(func() { t.OpenMenu() })
	}

	t.SetMenu(buildTrayMenu(app, cb, notifier))
	t.Show()
	return &TrayManager{tray: t, notifier: notifier}
}

// generateTrayIcons 从嵌入的源 PNG 生成两份托盘图标:
//   - tmpl: macOS 模板图标,36×36 单色 + alpha(RGB 固定为黑,alpha 直接取源图)
//   - color: Windows / Linux 彩色图标,32×32 RGBA
//
// 源图 build/appicon.png 实际是"透明背景 + 深灰 W 字母"(约 35% 全透明,
// 60% 不透明但颜色偏深),不是黑底白字。因此模板版的正确做法是:
//   - 保留源图 alpha(W 笔画不透明、背景透明),系统据此做亮/暗反色;
//   - RGB 全部压成黑色(模板图标准),系统在亮色模式画黑、暗色模式画白;
//   - 抗锯齿的边缘过渡通过源图 alpha 的中间值自然保留。
//
// 用 CatmullRom 高质量缩放 1024×1024 源图到目标尺寸。
func generateTrayIcons() (tmpl []byte, color []byte, err error) {
	src, err := png.Decode(bytes.NewReader(trayAppIconPNG))
	if err != nil {
		return nil, nil, err
	}

	// 模板版:36×36 黑色 + 源图 alpha
	tmplImg := image.NewNRGBA(image.Rect(0, 0, trayTemplateSize, trayTemplateSize))
	draw.CatmullRom.Scale(tmplImg, tmplImg.Bounds(), src, src.Bounds(), draw.Over, nil)
	for y := 0; y < trayTemplateSize; y++ {
		for x := 0; x < trayTemplateSize; x++ {
			_, _, _, a := tmplImg.At(x, y).RGBA()
			// RGB 固定黑色,alpha 直接沿用源图(CatmullRom 缩放后已含抗锯齿过渡)。
			tmplImg.SetNRGBA(x, y, imgcolor.NRGBA{R: 0, G: 0, B: 0, A: uint8(a >> 8)})
		}
	}
	var tmplBuf bytes.Buffer
	if err := png.Encode(&tmplBuf, tmplImg); err != nil {
		return nil, nil, err
	}
	tmpl = tmplBuf.Bytes()

	// 彩色版:32×32 RGBA
	colorImg := image.NewNRGBA(image.Rect(0, 0, trayColorSize, trayColorSize))
	draw.CatmullRom.Scale(colorImg, colorImg.Bounds(), src, src.Bounds(), draw.Over, nil)
	var colorBuf bytes.Buffer
	if err := png.Encode(&colorBuf, colorImg); err != nil {
		return nil, nil, err
	}
	color = colorBuf.Bytes()

	return tmpl, color, nil
}

// buildTrayMenu 构造托盘菜单。
func buildTrayMenu(app *application.App, cb TrayCallbacks, notifier *Notifier) *application.Menu {
	menu := application.NewMenu()

	menu.Add("显示主窗口").
		OnClick(func(_ *application.Context) {
			if cb.OnShow != nil {
				cb.OnShow()
			}
		})

	menu.Add("测试通知").
		OnClick(func(_ *application.Context) {
			if notifier == nil {
				log.Printf("tray: notifier not initialized, skip test notify")
				return
			}
			err := notifier.Notify(
				"",
				"Skill Box",
				"托盘测试通知 — "+time.Now().Format("15:04:05"),
			)
			if err != nil {
				log.Printf("tray: notify failed: %v", err)
			}
		})

	if cb.OnOpenSettings != nil {
		menu.Add("偏好设置").
			OnClick(func(_ *application.Context) {
				if cb.OnOpenSettings != nil {
					cb.OnOpenSettings()
				}
			})
	}

	menu.AddSeparator()
	menu.Add("关于 Skill Box").
		OnClick(func(_ *application.Context) {
			app.Dialog.Info().
				SetTitle("关于").
				SetMessage("Skill Box\n桌面端 + Web 端双部署\n本地后端走 http://127.0.0.1").
				Show()
		})
	menu.AddSeparator()
	menu.Add("退出").
		OnClick(func(_ *application.Context) {
			if cb.OnQuit != nil {
				cb.OnQuit()
			}
			app.Quit()
		})
	return menu
}
