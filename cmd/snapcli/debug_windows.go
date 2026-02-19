//go:build windows

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"
	"unsafe"
)

var debugLogFile *os.File

func initDebugLog() {
	exePath, err := os.Executable()
	if err != nil {
		return
	}
	logPath := filepath.Join(filepath.Dir(exePath), "snapcli_debug.log")
	debugLogFile, _ = os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
}

func debugLog(format string, args ...interface{}) {
	if debugLogFile == nil {
		return
	}
	ts := time.Now().Format("15:04:05.000")
	line := fmt.Sprintf("[%s] %s\n", ts, fmt.Sprintf(format, args...))
	debugLogFile.WriteString(line)
	debugLogFile.Sync()
}

func logDPIInfo() {
	if debugLogFile == nil {
		return
	}

	user32 := syscall.NewLazyDLL("user32.dll")
	shcore := syscall.NewLazyDLL("shcore.dll")
	gsm := user32.NewProc("GetSystemMetrics")

	smCxR, _, _ := gsm.Call(0)  // SM_CXSCREEN
	smCyR, _, _ := gsm.Call(1)  // SM_CYSCREEN
	vsXR, _, _ := gsm.Call(76)  // SM_XVIRTUALSCREEN
	vsYR, _, _ := gsm.Call(77)  // SM_YVIRTUALSCREEN
	vsWR, _, _ := gsm.Call(78)  // SM_CXVIRTUALSCREEN
	vsHR, _, _ := gsm.Call(79)  // SM_CYVIRTUALSCREEN

	// GetSystemMetrics 返回 int32，必须符号扩展
	smCx, smCy := int(int32(smCxR)), int(int32(smCyR))
	vsX, vsY := int(int32(vsXR)), int(int32(vsYR))
	vsW, vsH := int(int32(vsWR)), int(int32(vsHR))

	debugLog("=== DPI/显示器 诊断 ===")
	debugLog("SM_CXSCREEN=%d, SM_CYSCREEN=%d", smCx, smCy)
	debugLog("SM_XVIRTUALSCREEN=%d, SM_YVIRTUALSCREEN=%d (raw: %d, %d)", vsX, vsY, vsXR, vsYR)
	debugLog("SM_CXVIRTUALSCREEN=%d, SM_CYVIRTUALSCREEN=%d", vsW, vsH)

	// 获取系统 DPI
	getDpiForSystem := user32.NewProc("GetDpiForSystem")
	if getDpiForSystem.Find() == nil {
		dpi, _, _ := getDpiForSystem.Call()
		debugLog("GetDpiForSystem=%d (缩放: %d%%)", dpi, dpi*100/96)
	}

	// 获取线程 DPI 感知
	getAwareness := user32.NewProc("GetAwarenessFromDpiAwarenessContext")
	getThreadCtx := user32.NewProc("GetThreadDpiAwarenessContext")
	if getAwareness.Find() == nil && getThreadCtx.Find() == nil {
		ctx, _, _ := getThreadCtx.Call()
		aw, _, _ := getAwareness.Call(ctx)
		debugLog("ThreadDpiAwarenessContext=0x%X, Awareness=%d (0=unaware,1=system,2=permonitor)", ctx, aw)
	}

	// 获取进程 DPI 感知
	gpda := shcore.NewProc("GetProcessDpiAwareness")
	if gpda.Find() == nil {
		var awareness uint32
		gpda.Call(0, uintptr(unsafe.Pointer(&awareness)))
		debugLog("GetProcessDpiAwareness=%d (0=unaware,1=system,2=permonitor)", awareness)
	}
}
