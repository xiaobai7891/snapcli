package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"snapcli/internal/capture"
	"snapcli/internal/clipboard"
	"snapcli/internal/config"
	"snapcli/internal/hotkey"
	"snapcli/internal/notify"
	"snapcli/internal/storage"
	"snapcli/internal/tray"

	"golang.design/x/hotkey/mainthread"
)

var (
	cfg      *config.Config
	capturer capture.Capturer
	selector capture.Selector
	clip     clipboard.Clipboard
	notifier notify.Notifier
	store    *storage.Storage
	hkMgr    *hotkey.Manager
)

func main() {
	// 命令行参数
	setHotkeyFlag := flag.String("set-hotkey", "", "设置快捷键，格式：ctrl+alt+s")
	showConfig := flag.Bool("config", false, "显示配置文件路径")
	version := flag.Bool("version", false, "显示版本信息")
	flag.Parse()

	if *version {
		fmt.Println("SnapCLI v1.0.0")
		fmt.Println("截图路径复制工具")
		return
	}

	if *showConfig {
		fmt.Println("配置文件路径:", config.GetConfigPath())
		return
	}

	if *setHotkeyFlag != "" {
		if err := updateHotkey(*setHotkeyFlag); err != nil {
			fmt.Println("设置快捷键失败:", err)
			os.Exit(1)
		}
		fmt.Println("快捷键已设置为:", *setHotkeyFlag)
		return
	}

	// 使用 mainthread 确保热键在主线程运行
	mainthread.Init(run)
}

func run() {
	// 加载配置
	var err error
	cfg, err = config.Load()
	if err != nil {
		fmt.Println("加载配置失败:", err)
	}

	// 强制使用 exe 同级目录下的 screenshots 文件夹保存截图
	exePath, err := os.Executable()
	if err == nil {
		exeDir := filepath.Dir(exePath)
		screenshotDir := filepath.Join(exeDir, "screenshots")
		cfg.Storage.Directory = screenshotDir
	}

	// 确保存储目录存在
	cfg.EnsureStorageDir()

	// 初始化模块
	capturer = capture.NewCapturer()
	selector = capture.NewSelector()
	clip = clipboard.NewClipboard()
	notifier = notify.NewNotifier()
	store = storage.NewStorage(cfg.Storage.Directory, cfg.Storage.Format, cfg.Storage.Quality)

	fmt.Println("SnapCLI v1.0.1 已启动")
	fmt.Printf("快捷键: %s\n", cfg.GetHotkeyString())
	fmt.Printf("截图保存到: %s\n", cfg.Storage.Directory)
	fmt.Printf("Storage目录: %s\n", store.GetDirectory())
	fmt.Println("按快捷键截图，路径自动复制到剪贴板")

	// 创建并注册热键
	hkMgr = hotkey.NewManager()
	if err := hkMgr.Register(cfg.Hotkey.Modifiers, cfg.Hotkey.Key, onHotkeyPressed); err != nil {
		fmt.Println("注册热键失败:", err)
		fmt.Println("请检查快捷键是否被其他程序占用")
		fmt.Println("提示: 可以通过 --set-hotkey 参数设置其他快捷键")
		os.Exit(1)
	}
	defer hkMgr.Unregister()

	fmt.Println("热键注册成功!")

	// 异步监听热键
	hkMgr.ListenAsync()

	// 创建系统托盘
	t := tray.NewTray()
	t.SetHotkeyText(cfg.GetHotkeyString())
	t.SetOnScreenshot(onHotkeyPressed)
	t.SetOnOpenDir(openScreenshotDir)
	t.SetOnQuit(func() {
		hkMgr.Unregister()
		os.Exit(0)
	})

	// 运行托盘（阻塞）
	t.Run()
}

func onHotkeyPressed() {
	// 1. 全屏截图
	fullscreen, err := capturer.CaptureFullScreen()
	if err != nil {
		notifier.Show("截图失败", err.Error())
		return
	}

	// 2. 显示选区UI
	region, err := selector.SelectRegion(fullscreen)
	if err != nil {
		notifier.Show("选区失败", err.Error())
		return
	}

	// 用户取消
	if region == nil {
		return
	}

	// 3. 裁剪选区
	cropped := capture.CropImage(fullscreen, *region)

	// 4. 保存图片
	savePath, err := store.Save(cropped)
	if err != nil {
		notifier.Show("保存失败", err.Error())
		return
	}

	// 5. 复制路径到剪贴板
	if err := clip.SetText(savePath); err != nil {
		notifier.Show("复制失败", err.Error())
		return
	}

	// 6. 显示通知
	if cfg.Behavior.ShowNotification {
		notifier.Show("截图完成", savePath)
	}
}

func openScreenshotDir() {
	dir := cfg.Storage.Directory

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		// Windows 下使用 explorer.exe，路径需要处理
		cmd = exec.Command("explorer.exe", dir)
	case "darwin":
		cmd = exec.Command("open", dir)
	default:
		cmd = exec.Command("xdg-open", dir)
	}

	if err := cmd.Start(); err != nil {
		fmt.Println("打开目录失败:", err)
	}
}

func updateHotkey(hotkeyStr string) error {
	// 解析快捷键字符串，如 "ctrl+alt+s"
	modifiers := []string{}
	key := ""

	parts := splitHotkey(hotkeyStr)
	for i, part := range parts {
		if i == len(parts)-1 {
			key = part
		} else {
			modifiers = append(modifiers, part)
		}
	}

	if key == "" {
		return fmt.Errorf("无效的快捷键格式")
	}

	cfg, _ := config.Load()
	return cfg.SetHotkey(modifiers, key)
}

func splitHotkey(s string) []string {
	result := []string{}
	current := ""

	for _, c := range s {
		if c == '+' {
			if current != "" {
				result = append(result, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}

	if current != "" {
		result = append(result, current)
	}

	return result
}
