//go:build windows

package capture

import (
	"fmt"
	"image"
	"syscall"
	"unsafe"
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	gdi32    = syscall.NewLazyDLL("gdi32.dll")
	shcore   = syscall.NewLazyDLL("shcore.dll")

	getDC             = user32.NewProc("GetDC")
	releaseDC         = user32.NewProc("ReleaseDC")
	getSystemMetrics  = user32.NewProc("GetSystemMetrics")
	enumDisplayMonitors = user32.NewProc("EnumDisplayMonitors")
	getMonitorInfoW   = user32.NewProc("GetMonitorInfoW")

	createCompatibleDC     = gdi32.NewProc("CreateCompatibleDC")
	createCompatibleBitmap = gdi32.NewProc("CreateCompatibleBitmap")
	createDIBSection       = gdi32.NewProc("CreateDIBSection")
	selectObject           = gdi32.NewProc("SelectObject")
	bitBlt                 = gdi32.NewProc("BitBlt")
	deleteDC               = gdi32.NewProc("DeleteDC")
	deleteObject           = gdi32.NewProc("DeleteObject")
	getDIBits              = gdi32.NewProc("GetDIBits")

	getDpiForMonitor = shcore.NewProc("GetDpiForMonitor")
)

const (
	SM_XVIRTUALSCREEN  = 76
	SM_YVIRTUALSCREEN  = 77
	SM_CXVIRTUALSCREEN = 78
	SM_CYVIRTUALSCREEN = 79
	SRCCOPY            = 0x00CC0020
	BI_RGB             = 0
	DIB_RGB_COLORS     = 0
)

type BITMAPINFOHEADER struct {
	BiSize          uint32
	BiWidth         int32
	BiHeight        int32
	BiPlanes        uint16
	BiBitCount      uint16
	BiCompression   uint32
	BiSizeImage     uint32
	BiXPelsPerMeter int32
	BiYPelsPerMeter int32
	BiClrUsed       uint32
	BiClrImportant  uint32
}

type BITMAPINFO struct {
	BmiHeader BITMAPINFOHEADER
	BmiColors [1]uint32
}

type RECT struct {
	Left   int32
	Top    int32
	Right  int32
	Bottom int32
}

type MONITORINFO struct {
	CbSize    uint32
	RcMonitor RECT
	RcWork    RECT
	DwFlags   uint32
}

// WindowsCapturer Windows截图实现
type WindowsCapturer struct {
	displays []Display
}

// NewCapturer 创建截图器
func NewCapturer() Capturer {
	return &WindowsCapturer{}
}

// GetDisplays 获取所有显示器信息
func (c *WindowsCapturer) GetDisplays() ([]Display, error) {
	c.displays = []Display{}

	callback := syscall.NewCallback(func(hMonitor, hdcMonitor, lprcMonitor, dwData uintptr) uintptr {
		var mi MONITORINFO
		mi.CbSize = uint32(unsafe.Sizeof(mi))

		ret, _, _ := getMonitorInfoW.Call(hMonitor, uintptr(unsafe.Pointer(&mi)))
		if ret != 0 {
			display := Display{
				Index:       len(c.displays),
				X:           int(mi.RcMonitor.Left),
				Y:           int(mi.RcMonitor.Top),
				Width:       int(mi.RcMonitor.Right - mi.RcMonitor.Left),
				Height:      int(mi.RcMonitor.Bottom - mi.RcMonitor.Top),
				ScaleFactor: 1.0,
			}

			// 尝试获取DPI
			var dpiX, dpiY uint32
			if getDpiForMonitor.Find() == nil {
				ret, _, _ := getDpiForMonitor.Call(
					hMonitor,
					0, // MDT_EFFECTIVE_DPI
					uintptr(unsafe.Pointer(&dpiX)),
					uintptr(unsafe.Pointer(&dpiY)),
				)
				if ret == 0 {
					display.ScaleFactor = float64(dpiX) / 96.0
				}
			}

			c.displays = append(c.displays, display)
		}
		return 1 // 继续枚举
	})

	enumDisplayMonitors.Call(0, 0, callback, 0)

	return c.displays, nil
}

// GetFullBounds 获取所有显示器的总边界
func (c *WindowsCapturer) GetFullBounds() Region {
	x, _, _ := getSystemMetrics.Call(SM_XVIRTUALSCREEN)
	y, _, _ := getSystemMetrics.Call(SM_YVIRTUALSCREEN)
	width, _, _ := getSystemMetrics.Call(SM_CXVIRTUALSCREEN)
	height, _, _ := getSystemMetrics.Call(SM_CYVIRTUALSCREEN)

	return Region{
		X:      int(x),
		Y:      int(y),
		Width:  int(width),
		Height: int(height),
	}
}

// CaptureFullScreen 全屏截图
func (c *WindowsCapturer) CaptureFullScreen() (*image.RGBA, error) {
	bounds := c.GetFullBounds()
	return c.CaptureRegion(bounds)
}

// CaptureRegion 截取指定区域
func (c *WindowsCapturer) CaptureRegion(region Region) (*image.RGBA, error) {
	// 获取屏幕DC
	hdcScreen, _, err := getDC.Call(0)
	if hdcScreen == 0 {
		return nil, fmt.Errorf("无法获取屏幕DC: %v", err)
	}
	defer releaseDC.Call(0, hdcScreen)

	// 创建兼容DC
	hdcMem, _, err := createCompatibleDC.Call(hdcScreen)
	if hdcMem == 0 {
		return nil, fmt.Errorf("无法创建兼容DC: %v", err)
	}
	defer deleteDC.Call(hdcMem)

	// 使用 CreateDIBSection 创建可直接访问的位图
	var bi BITMAPINFO
	bi.BmiHeader.BiSize = uint32(unsafe.Sizeof(bi.BmiHeader))
	bi.BmiHeader.BiWidth = int32(region.Width)
	bi.BmiHeader.BiHeight = -int32(region.Height) // 负值 = 自顶向下
	bi.BmiHeader.BiPlanes = 1
	bi.BmiHeader.BiBitCount = 32
	bi.BmiHeader.BiCompression = BI_RGB

	var pBits uintptr
	hBitmap, _, _ := createDIBSection.Call(
		hdcMem,
		uintptr(unsafe.Pointer(&bi)),
		DIB_RGB_COLORS,
		uintptr(unsafe.Pointer(&pBits)),
		0, 0,
	)
	if hBitmap == 0 || pBits == 0 {
		return nil, fmt.Errorf("无法创建DIB位图")
	}
	defer deleteObject.Call(hBitmap)

	// 选择位图到DC
	hOldBitmap, _, _ := selectObject.Call(hdcMem, hBitmap)
	defer selectObject.Call(hdcMem, hOldBitmap)

	// BitBlt复制屏幕
	ret, _, _ := bitBlt.Call(
		hdcMem, 0, 0, uintptr(region.Width), uintptr(region.Height),
		hdcScreen, uintptr(region.X), uintptr(region.Y),
		SRCCOPY,
	)
	if ret == 0 {
		return nil, fmt.Errorf("BitBlt失败")
	}

	// 直接从 pBits 读取像素数据
	pixels := unsafe.Slice((*byte)(unsafe.Pointer(pBits)), region.Width*region.Height*4)

	// 转换为image.RGBA (Windows是BGRA格式)
	img := image.NewRGBA(image.Rect(0, 0, region.Width, region.Height))
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i+0] = pixels[i+2] // R <- B
		img.Pix[i+1] = pixels[i+1] // G <- G
		img.Pix[i+2] = pixels[i+0] // B <- R
		img.Pix[i+3] = 255         // A
	}

	return img, nil
}
