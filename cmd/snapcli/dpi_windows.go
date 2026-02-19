//go:build windows

package main

import "syscall"

func init() {
	// DPI 感知必须在任何 Win32 调用之前设置，放在 init() 确保最早执行
	user32 := syscall.NewLazyDLL("user32.dll")

	// 优先使用 Windows 10 1703+ API (最可靠)
	ctx := user32.NewProc("SetProcessDpiAwarenessContext")
	if err := ctx.Find(); err == nil {
		// DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2 = -4 (as uintptr)
		r, _, _ := ctx.Call(^uintptr(3))
		if r != 0 {
			return // 成功
		}
		// 如果 V2 失败，尝试 V1 (DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE = -3)
		r, _, _ = ctx.Call(^uintptr(2))
		if r != 0 {
			return
		}
		// 继续尝试旧 API
	}

	// 其次使用 Windows 8.1+ API
	shcore := syscall.NewLazyDLL("shcore.dll")
	awareness := shcore.NewProc("SetProcessDpiAwareness")
	if err := awareness.Find(); err == nil {
		r, _, _ := awareness.Call(2) // PROCESS_PER_MONITOR_DPI_AWARE
		if r == 0 {                  // S_OK = 0
			return
		}
		// E_ACCESSDENIED = 已设置过，尝试 SYSTEM_DPI_AWARE
		awareness.Call(1)
		return
	}

	// 最后回退到 Windows Vista+ API
	user32.NewProc("SetProcessDPIAware").Call()
}
