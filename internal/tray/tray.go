package tray

import (
	"github.com/getlantern/systray"
)

// Tray 系统托盘
type Tray struct {
	onScreenshot func()
	onOpenDir    func()
	onQuit       func()
	hotkeyText   string
}

// NewTray 创建系统托盘
func NewTray() *Tray {
	return &Tray{
		hotkeyText: "Alt+1",
	}
}

// SetHotkeyText 设置快捷键显示文本
func (t *Tray) SetHotkeyText(text string) {
	t.hotkeyText = text
}

// SetOnScreenshot 设置截图回调
func (t *Tray) SetOnScreenshot(fn func()) {
	t.onScreenshot = fn
}

// SetOnOpenDir 设置打开目录回调
func (t *Tray) SetOnOpenDir(fn func()) {
	t.onOpenDir = fn
}


// SetOnQuit 设置退出回调
func (t *Tray) SetOnQuit(fn func()) {
	t.onQuit = fn
}

// Run 运行系统托盘
func (t *Tray) Run() {
	systray.Run(t.onReady, t.onExit)
}

func (t *Tray) onReady() {
	systray.SetIcon(getIcon())
	systray.SetTitle("SnapCLI")
	systray.SetTooltip("SnapCLI - 截图路径复制工具")

	// 截图菜单项
	mScreenshot := systray.AddMenuItem("截图 ("+t.hotkeyText+")", "截取屏幕区域")
	systray.AddSeparator()

	// 打开截图目录
	mOpenDir := systray.AddMenuItem("打开截图目录", "打开截图保存位置")

	systray.AddSeparator()

	// 退出
	mQuit := systray.AddMenuItem("退出", "退出程序")

	go func() {
		for {
			select {
			case <-mScreenshot.ClickedCh:
				if t.onScreenshot != nil {
					t.onScreenshot()
				}
			case <-mOpenDir.ClickedCh:
				if t.onOpenDir != nil {
					t.onOpenDir()
				}
			case <-mQuit.ClickedCh:
				if t.onQuit != nil {
					t.onQuit()
				}
				systray.Quit()
				return
			}
		}
	}()
}

func (t *Tray) onExit() {
	// 清理资源
}
