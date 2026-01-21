//go:build windows

package capture

import (
	"image"
	"image/draw"
	"sync"
	"syscall"
	"unsafe"
)

var (
	kernel32 = syscall.NewLazyDLL("kernel32.dll")

	getModuleHandle   = kernel32.NewProc("GetModuleHandleW")
	peekMessageW      = user32.NewProc("PeekMessageW")
	registerClassExW  = user32.NewProc("RegisterClassExW")
	createWindowExW   = user32.NewProc("CreateWindowExW")
	showWindow          = user32.NewProc("ShowWindow")
	updateWindow        = user32.NewProc("UpdateWindow")
	destroyWindow       = user32.NewProc("DestroyWindow")
	setForegroundWindow = user32.NewProc("SetForegroundWindow")
	setFocus            = user32.NewProc("SetFocus")
	defWindowProcW    = user32.NewProc("DefWindowProcW")
	postQuitMessage   = user32.NewProc("PostQuitMessage")
	getMessageW       = user32.NewProc("GetMessageW")
	translateMessage  = user32.NewProc("TranslateMessage")
	dispatchMessageW  = user32.NewProc("DispatchMessageW")
	setCapture        = user32.NewProc("SetCapture")
	releaseCapture    = user32.NewProc("ReleaseCapture")
	getCursorPos      = user32.NewProc("GetCursorPos")
	setCursor         = user32.NewProc("SetCursor")
	loadCursorW       = user32.NewProc("LoadCursorW")
	setClassLongPtrW  = user32.NewProc("SetClassLongPtrW")
	invalidateRect    = user32.NewProc("InvalidateRect")
	beginPaint        = user32.NewProc("BeginPaint")
	endPaint          = user32.NewProc("EndPaint")
	windowFromPoint   = user32.NewProc("WindowFromPoint")
	getWindowRect     = user32.NewProc("GetWindowRect")
	getAncestor       = user32.NewProc("GetAncestor")
	isWindowVisible   = user32.NewProc("IsWindowVisible")
	setDIBitsToDevice = gdi32.NewProc("SetDIBitsToDevice")
	stretchDIBits     = gdi32.NewProc("StretchDIBits")
)

const (
	GA_ROOT = 2
)

const (
	WS_POPUP         = 0x80000000
	WS_VISIBLE       = 0x10000000
	WS_EX_TOPMOST    = 0x00000008
	WS_EX_TOOLWINDOW = 0x00000080

	SW_SHOW = 5

	WM_DESTROY     = 0x0002
	WM_PAINT       = 0x000F
	WM_QUIT        = 0x0012
	WM_KEYDOWN     = 0x0100
	WM_LBUTTONDOWN = 0x0201
	WM_LBUTTONUP   = 0x0202
	WM_SETCURSOR   = 0x0020
	WM_MOUSEMOVE   = 0x0200
	WM_RBUTTONDOWN = 0x0204

	VK_ESCAPE = 0x1B
	VK_RETURN = 0x0D

	IDC_CROSS = 32515
	IDC_ARROW = 32512

	PM_REMOVE = 0x0001

	// 光标相关常量
	CrosshairArmLength = 15 // 十字光标臂长
	CrosshairCenterGap = 3  // 十字光标中心空隙
	CrosshairPadding   = 5  // 光标区域额外边距
)

// getCursorRect 获取光标所在区域的矩形
func getCursorRect(x, y, maxW, maxH int) RECT {
	size := CrosshairArmLength + CrosshairCenterGap + CrosshairPadding
	return RECT{
		Left:   int32(max(0, x-size)),
		Top:    int32(max(0, y-size)),
		Right:  int32(min(maxW, x+size)),
		Bottom: int32(min(maxH, y+size)),
	}
}

// getSelectionRect 获取选区的矩形（带边框扩展）
func getSelectionRect(x1, y1, x2, y2, maxW, maxH int) RECT {
	if x1 > x2 {
		x1, x2 = x2, x1
	}
	if y1 > y2 {
		y1, y2 = y2, y1
	}
	border := 5 // 边框宽度 + 余量
	return RECT{
		Left:   int32(max(0, x1-border)),
		Top:    int32(max(0, y1-border)),
		Right:  int32(min(maxW, x2+border)),
		Bottom: int32(min(maxH, y2+border)),
	}
}

// getHoverRectBounds 获取悬停窗口区域的矩形（转换为屏幕坐标）
func getHoverRectBounds(hr *Region, screenX, screenY, maxW, maxH int) RECT {
	if hr == nil {
		return RECT{}
	}
	border := 5
	x := hr.X - screenX
	y := hr.Y - screenY
	return RECT{
		Left:   int32(max(0, x-border)),
		Top:    int32(max(0, y-border)),
		Right:  int32(min(maxW, x+hr.Width+border)),
		Bottom: int32(min(maxH, y+hr.Height+border)),
	}
}

// unionRect 合并两个矩形
func unionRect(r1, r2 RECT) RECT {
	if r1.Left == 0 && r1.Top == 0 && r1.Right == 0 && r1.Bottom == 0 {
		return r2
	}
	if r2.Left == 0 && r2.Top == 0 && r2.Right == 0 && r2.Bottom == 0 {
		return r1
	}
	return RECT{
		Left:   min(r1.Left, r2.Left),
		Top:    min(r1.Top, r2.Top),
		Right:  max(r1.Right, r2.Right),
		Bottom: max(r1.Bottom, r2.Bottom),
	}
}

// isRectEmpty 检查矩形是否为空
func isRectEmpty(r RECT) bool {
	return r.Right <= r.Left || r.Bottom <= r.Top
}

// invalidateDirtyRegions 使脏区域无效，触发重绘
func (s *WindowsSelector) invalidateDirtyRegions() {
	if len(s.dirtyRects) == 0 {
		return
	}

	// 合并所有脏区域为一个大矩形（简化处理）
	var combined RECT
	for _, r := range s.dirtyRects {
		combined = unionRect(combined, r)
	}

	if !isRectEmpty(combined) {
		invalidateRect.Call(s.hwnd, uintptr(unsafe.Pointer(&combined)), 0)
	}

	// 清空脏区域列表
	s.dirtyRects = s.dirtyRects[:0]
}

// addDirtyRect 添加脏区域
func (s *WindowsSelector) addDirtyRect(r RECT) {
	if !isRectEmpty(r) {
		s.dirtyRects = append(s.dirtyRects, r)
	}
}

// drainQuitMessages 清理消息队列中残留的 WM_QUIT 消息
func drainQuitMessages() {
	var msg MSG
	for {
		ret, _, _ := peekMessageW.Call(
			uintptr(unsafe.Pointer(&msg)),
			0,
			WM_QUIT,
			WM_QUIT,
			PM_REMOVE,
		)
		if ret == 0 {
			break
		}
	}
}

type WNDCLASSEXW struct {
	CbSize        uint32
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     uintptr
	HIcon         uintptr
	HCursor       uintptr
	HbrBackground uintptr
	LpszMenuName  *uint16
	LpszClassName *uint16
	HIconSm       uintptr
}

type MSG struct {
	HWnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      POINT
}

type POINT struct {
	X int32
	Y int32
}

type PAINTSTRUCT struct {
	Hdc         uintptr
	FErase      int32
	RcPaint     RECT
	FRestore    int32
	FIncUpdate  int32
	RgbReserved [32]byte
}

// WindowsSelector Windows选区实现
type WindowsSelector struct {
	hwnd       uintptr
	background *image.RGBA
	startX     int
	startY     int
	endX       int
	endY       int
	selecting  bool
	done       bool
	cancelled  bool
	result     *Region
	// 窗口检测
	hoverRect  *Region // 鼠标下方的窗口区域
	screenX    int     // 屏幕起始X坐标
	screenY    int     // 屏幕起始Y坐标
	// 鼠标位置（用于绘制自定义光标）
	mouseX int
	mouseY int
	// 脏区域跟踪（用于优化重绘）
	lastMouseX    int     // 上一帧鼠标X
	lastMouseY    int     // 上一帧鼠标Y
	lastHoverRect *Region // 上一帧的窗口高亮区域
	lastEndX      int     // 上一帧选区终点X
	lastEndY      int     // 上一帧选区终点Y
	dirtyRects    []RECT  // 需要重绘的区域列表
}

var (
	selectorMutex           sync.Mutex
	selectorInstance        *WindowsSelector
	selectorClassRegistered bool // 标记窗口类是否已注册
)

var (
	getDesktopWindow = user32.NewProc("GetDesktopWindow")
	getShellWindow   = user32.NewProc("GetShellWindow")
	getParent        = user32.NewProc("GetParent")
	getClassName     = user32.NewProc("GetClassNameW")
)

// getWindowAtPoint 获取指定屏幕坐标处的窗口区域
// excludeHwnd 是要排除的窗口句柄（通常是选区窗口本身）
// 返回 nil 表示在桌面上
func getWindowAtPoint(screenX, screenY, fullWidth, fullHeight int, excludeHwnd uintptr) *Region {
	// WindowFromPoint 需要传递一个 POINT 结构
	point := uintptr(uint32(screenX)) | (uintptr(uint32(screenY)) << 32)

	hwnd, _, _ := windowFromPoint.Call(point)
	if hwnd == 0 {
		return nil
	}

	// 排除选区窗口本身
	if hwnd == excludeHwnd {
		return nil
	}

	// 获取桌面窗口句柄
	desktopHwnd, _, _ := getDesktopWindow.Call()
	shellHwnd, _, _ := getShellWindow.Call()

	// 获取顶级窗口
	rootHwnd, _, _ := getAncestor.Call(hwnd, GA_ROOT)
	if rootHwnd != 0 {
		hwnd = rootHwnd
	}

	// 排除选区窗口本身（检查顶级窗口）
	if hwnd == excludeHwnd {
		return nil
	}

	// 检查是否是桌面或 shell 窗口
	if hwnd == desktopHwnd || hwnd == shellHwnd {
		return nil // 桌面，返回 nil
	}

	// 检查窗口类名，排除桌面相关窗口
	className := make([]uint16, 256)
	getClassName.Call(hwnd, uintptr(unsafe.Pointer(&className[0])), 256)
	classStr := syscall.UTF16ToString(className)
	if classStr == "Progman" || classStr == "WorkerW" || classStr == "Shell_TrayWnd" {
		return nil // 桌面相关窗口
	}

	// 检查窗口是否可见
	visible, _, _ := isWindowVisible.Call(hwnd)
	if visible == 0 {
		return nil
	}

	// 获取窗口矩形
	var rect RECT
	ret, _, _ := getWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&rect)))
	if ret == 0 {
		return nil
	}

	return &Region{
		X:      int(rect.Left),
		Y:      int(rect.Top),
		Width:  int(rect.Right - rect.Left),
		Height: int(rect.Bottom - rect.Top),
	}
}

// NewSelector 创建选区器
func NewSelector() Selector {
	return &WindowsSelector{}
}

// SelectRegion 显示选区UI
func (s *WindowsSelector) SelectRegion(background *image.RGBA) (*Region, error) {
	// 防止并发调用
	selectorMutex.Lock()
	defer selectorMutex.Unlock()

	s.background = background
	s.selecting = false
	s.done = false
	s.cancelled = false
	s.result = nil
	s.lastMouseX = -1 // 初始化脏区域跟踪状态
	s.lastMouseY = -1
	s.lastHoverRect = nil
	s.lastEndX = 0
	s.lastEndY = 0
	s.dirtyRects = nil

	// 清理消息队列中可能残留的 WM_QUIT 消息
	drainQuitMessages()

	// 重置缓存
	if cachedBitmap != 0 {
		deleteObject.Call(cachedBitmap)
		cachedBitmap = 0
	}
	if cachedMemDC != 0 {
		deleteDC.Call(cachedMemDC)
		cachedMemDC = 0
	}
	cachedWidth = 0
	cachedHeight = 0

	selectorInstance = s

	// 获取模块句柄
	hInstance, _, _ := getModuleHandle.Call(0)

	// 注册窗口类（只注册一次）
	className := syscall.StringToUTF16Ptr("SnapCLISelectorClass")
	if !selectorClassRegistered {
		var wc WNDCLASSEXW
		wc.CbSize = uint32(unsafe.Sizeof(wc))
		wc.LpfnWndProc = syscall.NewCallback(selectorWndProc)
		wc.HInstance = hInstance
		wc.HCursor, _, _ = loadCursorW.Call(0, uintptr(IDC_CROSS))
		wc.LpszClassName = className

		registerClassExW.Call(uintptr(unsafe.Pointer(&wc)))
		selectorClassRegistered = true
	}

	// 获取屏幕尺寸
	bounds := NewCapturer().GetFullBounds()
	s.screenX = bounds.X
	s.screenY = bounds.Y

	// 创建全屏窗口
	hwnd, _, _ := createWindowExW.Call(
		WS_EX_TOPMOST|WS_EX_TOOLWINDOW,
		uintptr(unsafe.Pointer(className)),
		0,
		WS_POPUP|WS_VISIBLE,
		uintptr(bounds.X), uintptr(bounds.Y),
		uintptr(bounds.Width), uintptr(bounds.Height),
		0, 0, hInstance, 0,
	)

	if hwnd == 0 {
		// 清理缓存资源，防止泄漏
		cleanupSelectorCache()
		return nil, nil
	}

	s.hwnd = hwnd

	showWindow.Call(hwnd, SW_SHOW)
	updateWindow.Call(hwnd)
	setForegroundWindow.Call(hwnd)
	setFocus.Call(hwnd)
	// 强制重绘
	invalidateRect.Call(hwnd, 0, 1)

	// 消息循环
	var msg MSG
	for !s.done {
		ret, _, _ := getMessageW.Call(
			uintptr(unsafe.Pointer(&msg)),
			0, 0, 0,
		)
		// ret == 0 表示 WM_QUIT，ret == -1 表示错误
		if ret == 0 || ret == uintptr(^uintptr(0)) {
			break
		}
		translateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		dispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
	}

	destroyWindow.Call(hwnd)

	// 清理缓存资源
	cleanupSelectorCache()

	// 重置状态
	s.hwnd = 0
	s.hoverRect = nil
	s.background = nil
	selectorInstance = nil

	// 恢复默认光标
	defaultCursor, _, _ := loadCursorW.Call(0, IDC_ARROW)
	setCursor.Call(defaultCursor)

	if s.cancelled {
		return nil, nil
	}

	return s.result, nil
}

func selectorWndProc(hwnd, msg, wParam, lParam uintptr) uintptr {
	s := selectorInstance
	if s == nil {
		ret, _, _ := defWindowProcW.Call(hwnd, msg, wParam, lParam)
		return ret
	}

	switch msg {
	case WM_SETCURSOR:
		// 隐藏系统光标，我们自己绘制
		setCursor.Call(0)
		return 1 // 返回 TRUE 表示已处理

	case WM_PAINT:
		var ps PAINTSTRUCT
		hdc, _, _ := beginPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))

		// 绘制背景图片（带遮罩）
		if s.background != nil {
			drawBackgroundWithOverlay(hdc, s.background, s)
		}

		endPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
		return 0

	case WM_KEYDOWN:
		if wParam == VK_ESCAPE {
			s.cancelled = true
			s.done = true
			postQuitMessage.Call(0)
		} else if wParam == VK_RETURN {
			// 全屏截图
			s.result = &Region{
				X:      0,
				Y:      0,
				Width:  s.background.Bounds().Dx(),
				Height: s.background.Bounds().Dy(),
			}
			s.done = true
			postQuitMessage.Call(0)
		}
		return 0

	case WM_LBUTTONDOWN:
		// 如果有检测到的窗口，直接使用该窗口区域
		if s.hoverRect != nil && !s.selecting {
			// 转换为相对于截图的坐标
			s.result = &Region{
				X:      s.hoverRect.X - s.screenX,
				Y:      s.hoverRect.Y - s.screenY,
				Width:  s.hoverRect.Width,
				Height: s.hoverRect.Height,
			}
			// 确保区域在屏幕范围内
			if s.result.X < 0 {
				s.result.Width += s.result.X
				s.result.X = 0
			}
			if s.result.Y < 0 {
				s.result.Height += s.result.Y
				s.result.Y = 0
			}
			bgWidth := s.background.Bounds().Dx()
			bgHeight := s.background.Bounds().Dy()
			if s.result.X+s.result.Width > bgWidth {
				s.result.Width = bgWidth - s.result.X
			}
			if s.result.Y+s.result.Height > bgHeight {
				s.result.Height = bgHeight - s.result.Y
			}

			s.done = true
			postQuitMessage.Call(0)
			return 0
		}

		// 否则开始手动选择
		s.startX = int(int16(lParam & 0xFFFF))
		s.startY = int(int16((lParam >> 16) & 0xFFFF))
		s.endX = s.startX
		s.endY = s.startY
		s.lastEndX = s.startX // 初始化脏区域跟踪
		s.lastEndY = s.startY
		s.selecting = true
		s.hoverRect = nil // 清除窗口高亮
		setCapture.Call(hwnd)
		return 0

	case WM_MOUSEMOVE:
		mouseX := int(int16(lParam & 0xFFFF))
		mouseY := int(int16((lParam >> 16) & 0xFFFF))

		bgWidth := s.background.Bounds().Dx()
		bgHeight := s.background.Bounds().Dy()

		// 清空脏区域列表
		s.dirtyRects = s.dirtyRects[:0]

		// 1. 添加旧光标位置为脏区域（-1 表示首次，跳过旧位置清理）
		if s.lastMouseX >= 0 && s.lastMouseY >= 0 {
			s.addDirtyRect(getCursorRect(s.lastMouseX, s.lastMouseY, bgWidth, bgHeight))
		}
		// 2. 添加新光标位置为脏区域
		s.addDirtyRect(getCursorRect(mouseX, mouseY, bgWidth, bgHeight))

		if s.selecting {
			// 3. 选区模式：添加旧选区和新选区的脏区域
			if s.lastEndX != s.startX || s.lastEndY != s.startY {
				s.addDirtyRect(getSelectionRect(s.startX, s.startY, s.lastEndX, s.lastEndY, bgWidth, bgHeight))
			}
			s.addDirtyRect(getSelectionRect(s.startX, s.startY, mouseX, mouseY, bgWidth, bgHeight))
			s.endX = mouseX
			s.endY = mouseY
			s.lastEndX = mouseX
			s.lastEndY = mouseY
		} else {
			// 4. 非选区模式：检测窗口变化
			screenX := s.screenX + mouseX
			screenY := s.screenY + mouseY
			newHoverRect := getWindowAtPoint(screenX, screenY, bgWidth, bgHeight, s.hwnd)

			// 如果悬停窗口发生变化，添加旧和新的窗口区域为脏区域
			hoverChanged := false
			if s.lastHoverRect == nil && newHoverRect != nil {
				hoverChanged = true
			} else if s.lastHoverRect != nil && newHoverRect == nil {
				hoverChanged = true
			} else if s.lastHoverRect != nil && newHoverRect != nil {
				if s.lastHoverRect.X != newHoverRect.X || s.lastHoverRect.Y != newHoverRect.Y ||
					s.lastHoverRect.Width != newHoverRect.Width || s.lastHoverRect.Height != newHoverRect.Height {
					hoverChanged = true
				}
			}

			if hoverChanged {
				if s.lastHoverRect != nil {
					s.addDirtyRect(getHoverRectBounds(s.lastHoverRect, s.screenX, s.screenY, bgWidth, bgHeight))
				}
				if newHoverRect != nil {
					s.addDirtyRect(getHoverRectBounds(newHoverRect, s.screenX, s.screenY, bgWidth, bgHeight))
				}
			}

			s.hoverRect = newHoverRect
			// 复制当前 hoverRect 到 lastHoverRect
			if newHoverRect != nil {
				s.lastHoverRect = &Region{
					X: newHoverRect.X, Y: newHoverRect.Y,
					Width: newHoverRect.Width, Height: newHoverRect.Height,
				}
			} else {
				s.lastHoverRect = nil
			}
		}

		// 更新鼠标位置
		s.mouseX = mouseX
		s.mouseY = mouseY
		s.lastMouseX = mouseX
		s.lastMouseY = mouseY

		// 仅使脏区域无效
		s.invalidateDirtyRegions()
		return 0

	case WM_LBUTTONUP:
		if s.selecting {
			releaseCapture.Call()
			s.selecting = false

			s.endX = int(int16(lParam & 0xFFFF))
			s.endY = int(int16((lParam >> 16) & 0xFFFF))

			// 计算选区
			x1, x2 := s.startX, s.endX
			y1, y2 := s.startY, s.endY
			if x1 > x2 {
				x1, x2 = x2, x1
			}
			if y1 > y2 {
				y1, y2 = y2, y1
			}

			width := x2 - x1
			height := y2 - y1

			// 忽略太小的选区
			if width > 5 && height > 5 {
				s.result = &Region{
					X:      x1,
					Y:      y1,
					Width:  width,
					Height: height,
				}
				s.done = true
				postQuitMessage.Call(0)
			}
		}
		return 0

	case WM_RBUTTONDOWN:
		s.cancelled = true
		s.done = true
		postQuitMessage.Call(0)
		return 0

	case WM_DESTROY:
		// 不要在这里调用 postQuitMessage，因为消息循环已经在 s.done 时退出了
		// 如果在这里调用，会在消息队列中留下 WM_QUIT，导致下次截图时立即退出
		return 0
	}

	ret, _, _ := defWindowProcW.Call(hwnd, msg, wParam, lParam)
	return ret
}

// 缓存
var (
	cachedMemDC    uintptr
	cachedBitmap   uintptr
	cachedBits     uintptr
	cachedWidth    int
	cachedHeight   int
	darkPixels     []byte // 预计算的暗色背景
	brightPixels   []byte // 原始亮色背景
)

// cleanupSelectorCache 清理选区器缓存资源
func cleanupSelectorCache() {
	if cachedBitmap != 0 {
		deleteObject.Call(cachedBitmap)
		cachedBitmap = 0
	}
	if cachedMemDC != 0 {
		deleteDC.Call(cachedMemDC)
		cachedMemDC = 0
	}
	cachedBits = 0
	cachedWidth = 0
	cachedHeight = 0
	darkPixels = nil
	brightPixels = nil
}

func drawBackgroundWithOverlay(hdc uintptr, bg *image.RGBA, s *WindowsSelector) {
	bounds := bg.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// 创建或重用内存 DC 和 DIB
	if cachedMemDC == 0 || cachedWidth != width || cachedHeight != height {
		// 清理旧资源
		if cachedBitmap != 0 {
			deleteObject.Call(cachedBitmap)
		}
		if cachedMemDC != 0 {
			deleteDC.Call(cachedMemDC)
		}

		// 创建内存 DC
		cachedMemDC, _, _ = createCompatibleDC.Call(hdc)

		// 创建 DIB Section
		var bi BITMAPINFO
		bi.BmiHeader.BiSize = uint32(unsafe.Sizeof(bi.BmiHeader))
		bi.BmiHeader.BiWidth = int32(width)
		bi.BmiHeader.BiHeight = -int32(height)
		bi.BmiHeader.BiPlanes = 1
		bi.BmiHeader.BiBitCount = 32
		bi.BmiHeader.BiCompression = BI_RGB

		cachedBitmap, _, _ = createDIBSection.Call(
			cachedMemDC,
			uintptr(unsafe.Pointer(&bi)),
			DIB_RGB_COLORS,
			uintptr(unsafe.Pointer(&cachedBits)),
			0, 0,
		)

		if cachedBitmap == 0 {
			return
		}

		selectObject.Call(cachedMemDC, cachedBitmap)
		cachedWidth = width
		cachedHeight = height

		// 预计算暗色和亮色背景
		size := width * height * 4
		darkPixels = make([]byte, size)
		brightPixels = make([]byte, size)
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				i := (y*width + x) * 4
				c := bg.RGBAAt(x, y)
				// 暗色 (50%亮度)
				darkPixels[i+0] = c.B / 2
				darkPixels[i+1] = c.G / 2
				darkPixels[i+2] = c.R / 2
				darkPixels[i+3] = 255
				// 亮色 (原始)
				brightPixels[i+0] = c.B
				brightPixels[i+1] = c.G
				brightPixels[i+2] = c.R
				brightPixels[i+3] = 255
			}
		}
	}

	pixels := unsafe.Slice((*byte)(unsafe.Pointer(cachedBits)), width*height*4)

	// 确定要高亮的区域
	var highlightRegion *Region

	if s.selecting {
		// 手动选择模式
		x1, x2 := s.startX, s.endX
		y1, y2 := s.startY, s.endY
		if x1 > x2 {
			x1, x2 = x2, x1
		}
		if y1 > y2 {
			y1, y2 = y2, y1
		}
		highlightRegion = &Region{X: x1, Y: y1, Width: x2 - x1, Height: y2 - y1}
	} else if s.hoverRect != nil {
		// 窗口检测模式 - 转换为相对坐标
		highlightRegion = &Region{
			X:      s.hoverRect.X - s.screenX,
			Y:      s.hoverRect.Y - s.screenY,
			Width:  s.hoverRect.Width,
			Height: s.hoverRect.Height,
		}
	}
	// 当 hoverRect 为 nil 时，不设置 highlightRegion，显示全屏暗色遮罩

	// 始终先填充暗色背景
	copy(pixels, darkPixels)

	// 如果有高亮区域，用亮色覆盖
	if highlightRegion != nil {
		x1 := max(0, highlightRegion.X)
		y1 := max(0, highlightRegion.Y)
		x2 := min(highlightRegion.X+highlightRegion.Width, width)
		y2 := min(highlightRegion.Y+highlightRegion.Height, height)

		for y := y1; y < y2; y++ {
			srcStart := (y*width + x1) * 4
			srcEnd := (y*width + x2) * 4
			copy(pixels[srcStart:srcEnd], brightPixels[srcStart:srcEnd])
		}
	}

	// 绘制绿色边框（3像素宽）
	if highlightRegion != nil {
		x1 := highlightRegion.X
		y1 := highlightRegion.Y
		x2 := x1 + highlightRegion.Width
		y2 := y1 + highlightRegion.Height

		borderB, borderG, borderR := byte(0), byte(200), byte(0)
		borderWidth := 3

		for t := 0; t < borderWidth; t++ {
			// 上边框
			if y1+t >= 0 && y1+t < height {
				for x := max(0, x1); x < min(x2, width); x++ {
					i := ((y1+t)*width + x) * 4
					pixels[i+0], pixels[i+1], pixels[i+2] = borderB, borderG, borderR
				}
			}
			// 下边框
			if y2-1-t >= 0 && y2-1-t < height {
				for x := max(0, x1); x < min(x2, width); x++ {
					i := ((y2-1-t)*width + x) * 4
					pixels[i+0], pixels[i+1], pixels[i+2] = borderB, borderG, borderR
				}
			}
			// 左边框
			if x1+t >= 0 && x1+t < width {
				for y := max(0, y1); y < min(y2, height); y++ {
					i := (y*width + x1 + t) * 4
					pixels[i+0], pixels[i+1], pixels[i+2] = borderB, borderG, borderR
				}
			}
			// 右边框
			if x2-1-t >= 0 && x2-1-t < width {
				for y := max(0, y1); y < min(y2, height); y++ {
					i := (y*width + x2 - 1 - t) * 4
					pixels[i+0], pixels[i+1], pixels[i+2] = borderB, borderG, borderR
				}
			}
		}
	}

	// 绘制自定义十字光标（白色轮廓 + 红色中心，在任何背景下都可见）
	drawCrosshair(pixels, width, height, s.mouseX, s.mouseY)

	// BitBlt 到窗口
	bitBlt.Call(hdc, 0, 0, uintptr(width), uintptr(height), cachedMemDC, 0, 0, SRCCOPY)
}

// drawCrosshair 绘制带轮廓的十字光标
func drawCrosshair(pixels []byte, width, height, mx, my int) {
	if mx < 0 || mx >= width || my < 0 || my >= height {
		return
	}

	// 使用定义的常量
	armLength := CrosshairArmLength
	centerGap := CrosshairCenterGap

	// 颜色定义 (BGR格式)
	outlineB, outlineG, outlineR := byte(255), byte(255), byte(255) // 白色轮廓
	centerB, centerG, centerR := byte(0), byte(0), byte(255)        // 红色中心

	// 绘制像素的辅助函数
	setPixel := func(x, y int, b, g, r byte) {
		if x >= 0 && x < width && y >= 0 && y < height {
			i := (y*width + x) * 4
			pixels[i+0], pixels[i+1], pixels[i+2] = b, g, r
		}
	}

	// 绘制带轮廓的线段
	drawLineWithOutline := func(x1, y1, x2, y2 int) {
		if x1 == x2 { // 垂直线
			for y := min(y1, y2); y <= max(y1, y2); y++ {
				// 白色轮廓 (左右各1像素)
				setPixel(x1-1, y, outlineB, outlineG, outlineR)
				setPixel(x1+1, y, outlineB, outlineG, outlineR)
				// 红色中心
				setPixel(x1, y, centerB, centerG, centerR)
			}
			// 端点轮廓
			setPixel(x1, min(y1, y2)-1, outlineB, outlineG, outlineR)
			setPixel(x1, max(y1, y2)+1, outlineB, outlineG, outlineR)
		} else { // 水平线
			for x := min(x1, x2); x <= max(x1, x2); x++ {
				// 白色轮廓 (上下各1像素)
				setPixel(x, y1-1, outlineB, outlineG, outlineR)
				setPixel(x, y1+1, outlineB, outlineG, outlineR)
				// 红色中心
				setPixel(x, y1, centerB, centerG, centerR)
			}
			// 端点轮廓
			setPixel(min(x1, x2)-1, y1, outlineB, outlineG, outlineR)
			setPixel(max(x1, x2)+1, y1, outlineB, outlineG, outlineR)
		}
	}

	// 绘制四条臂（带中心空隙）
	// 上
	drawLineWithOutline(mx, my-centerGap-armLength, mx, my-centerGap)
	// 下
	drawLineWithOutline(mx, my+centerGap, mx, my+centerGap+armLength)
	// 左
	drawLineWithOutline(mx-centerGap-armLength, my, mx-centerGap, my)
	// 右
	drawLineWithOutline(mx+centerGap, my, mx+centerGap+armLength, my)
}

// 简化版：直接复制区域
func copyRegion(dst, src *image.RGBA, region Region) {
	draw.Draw(dst, image.Rect(region.X, region.Y, region.X+region.Width, region.Y+region.Height),
		src, image.Point{region.X, region.Y}, draw.Src)
}
