//go:build windows

package clipboard

import (
	"syscall"
	"unsafe"
)

var (
	user32 = syscall.NewLazyDLL("user32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")

	openClipboard    = user32.NewProc("OpenClipboard")
	closeClipboard   = user32.NewProc("CloseClipboard")
	emptyClipboard   = user32.NewProc("EmptyClipboard")
	setClipboardData = user32.NewProc("SetClipboardData")
	getClipboardData = user32.NewProc("GetClipboardData")

	globalAlloc   = kernel32.NewProc("GlobalAlloc")
	globalFree    = kernel32.NewProc("GlobalFree")
	globalLock    = kernel32.NewProc("GlobalLock")
	globalUnlock  = kernel32.NewProc("GlobalUnlock")
	lstrcpyW      = kernel32.NewProc("lstrcpyW")
)

const (
	CF_UNICODETEXT = 13
	GMEM_MOVEABLE  = 0x0002
)

// WindowsClipboard Windows剪贴板实现
type WindowsClipboard struct{}

// NewClipboard 创建剪贴板实例
func NewClipboard() Clipboard {
	return &WindowsClipboard{}
}

// SetText 设置剪贴板文本
func (c *WindowsClipboard) SetText(text string) error {
	// 转换为UTF-16
	utf16 := syscall.StringToUTF16(text)
	size := len(utf16) * 2

	// 打开剪贴板
	ret, _, _ := openClipboard.Call(0)
	if ret == 0 {
		return syscall.GetLastError()
	}
	defer closeClipboard.Call()

	// 清空剪贴板
	emptyClipboard.Call()

	// 分配全局内存
	hMem, _, _ := globalAlloc.Call(GMEM_MOVEABLE, uintptr(size))
	if hMem == 0 {
		return syscall.GetLastError()
	}

	// 锁定内存并复制数据
	ptr, _, _ := globalLock.Call(hMem)
	if ptr == 0 {
		globalFree.Call(hMem)
		return syscall.GetLastError()
	}

	// 复制字符串
	for i, v := range utf16 {
		*(*uint16)(unsafe.Pointer(ptr + uintptr(i*2))) = v
	}

	globalUnlock.Call(hMem)

	// 设置剪贴板数据
	ret, _, _ = setClipboardData.Call(CF_UNICODETEXT, hMem)
	if ret == 0 {
		globalFree.Call(hMem)
		return syscall.GetLastError()
	}

	return nil
}

// GetText 获取剪贴板文本
func (c *WindowsClipboard) GetText() (string, error) {
	// 打开剪贴板
	ret, _, _ := openClipboard.Call(0)
	if ret == 0 {
		return "", syscall.GetLastError()
	}
	defer closeClipboard.Call()

	// 获取剪贴板数据
	hMem, _, _ := getClipboardData.Call(CF_UNICODETEXT)
	if hMem == 0 {
		return "", nil
	}

	// 锁定内存
	ptr, _, _ := globalLock.Call(hMem)
	if ptr == 0 {
		return "", syscall.GetLastError()
	}
	defer globalUnlock.Call(hMem)

	// 读取字符串
	var text []uint16
	for i := 0; ; i++ {
		c := *(*uint16)(unsafe.Pointer(ptr + uintptr(i*2)))
		if c == 0 {
			break
		}
		text = append(text, c)
	}

	return syscall.UTF16ToString(text), nil
}
