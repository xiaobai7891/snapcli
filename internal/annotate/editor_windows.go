//go:build windows

package annotate

import (
	"image"
	"image/color"
	"image/draw"
	"math"
	"runtime"
	"strconv"
	"sync"
	"syscall"
	"unicode/utf16"
	"unsafe"
)

// ============================================================================
// Win32 DLL 和 API 声明
// ============================================================================

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	gdi32    = syscall.NewLazyDLL("gdi32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
)

// user32 函数
var (
	getDC               = user32.NewProc("GetDC")
	releaseDC            = user32.NewProc("ReleaseDC")
	getSystemMetrics     = user32.NewProc("GetSystemMetrics")
	getModuleHandle      = kernel32.NewProc("GetModuleHandleW")
	registerClassExW     = user32.NewProc("RegisterClassExW")
	createWindowExW      = user32.NewProc("CreateWindowExW")
	showWindow           = user32.NewProc("ShowWindow")
	updateWindow         = user32.NewProc("UpdateWindow")
	destroyWindow        = user32.NewProc("DestroyWindow")
	setForegroundWindow  = user32.NewProc("SetForegroundWindow")
	setFocus             = user32.NewProc("SetFocus")
	defWindowProcW       = user32.NewProc("DefWindowProcW")
	postQuitMessage      = user32.NewProc("PostQuitMessage")
	getMessageW          = user32.NewProc("GetMessageW")
	translateMessage     = user32.NewProc("TranslateMessage")
	dispatchMessageW     = user32.NewProc("DispatchMessageW")
	peekMessageW         = user32.NewProc("PeekMessageW")
	setCapture           = user32.NewProc("SetCapture")
	releaseCapture       = user32.NewProc("ReleaseCapture")
	setCursor            = user32.NewProc("SetCursor")
	loadCursorW          = user32.NewProc("LoadCursorW")
	invalidateRect       = user32.NewProc("InvalidateRect")
	beginPaint           = user32.NewProc("BeginPaint")
	endPaint             = user32.NewProc("EndPaint")
	getKeyState          = user32.NewProc("GetKeyState")
	fillRect             = user32.NewProc("FillRect")
	getCursorPos         = user32.NewProc("GetCursorPos")
	screenToClientProc   = user32.NewProc("ScreenToClient")
)

// gdi32 函数
var (
	createCompatibleDC = gdi32.NewProc("CreateCompatibleDC")
	createDIBSection   = gdi32.NewProc("CreateDIBSection")
	selectObject       = gdi32.NewProc("SelectObject")
	bitBlt             = gdi32.NewProc("BitBlt")
	deleteDC           = gdi32.NewProc("DeleteDC")
	deleteObject       = gdi32.NewProc("DeleteObject")
	createPen          = gdi32.NewProc("CreatePen")
	createSolidBrush   = gdi32.NewProc("CreateSolidBrush")
	rectangle          = gdi32.NewProc("Rectangle")
	ellipse            = gdi32.NewProc("Ellipse")
	moveToEx           = gdi32.NewProc("MoveToEx")
	lineTo             = gdi32.NewProc("LineTo")
	textOutW           = gdi32.NewProc("TextOutW")
	setTextColor       = gdi32.NewProc("SetTextColor")
	setBkMode          = gdi32.NewProc("SetBkMode")
	createFontW        = gdi32.NewProc("CreateFontW")
	getStockObject     = gdi32.NewProc("GetStockObject")
	setDCPenColor      = gdi32.NewProc("SetDCPenColor")
	setDCBrushColor    = gdi32.NewProc("SetDCBrushColor")
	roundRect          = gdi32.NewProc("RoundRect")
	polyBezier         = gdi32.NewProc("PolyBezier")
)

// GDI+ DLL 和函数（用于抗锯齿绘制）
var (
	gdiplusDLL = syscall.NewLazyDLL("gdiplus.dll")

	gdiplusStartup           = gdiplusDLL.NewProc("GdiplusStartup")
	gdiplusShutdown          = gdiplusDLL.NewProc("GdiplusShutdown")
	gdipCreateFromHDC        = gdiplusDLL.NewProc("GdipCreateFromHDC")
	gdipDeleteGraphics       = gdiplusDLL.NewProc("GdipDeleteGraphics")
	gdipSetSmoothingMode     = gdiplusDLL.NewProc("GdipSetSmoothingMode")
	gdipCreatePen1           = gdiplusDLL.NewProc("GdipCreatePen1")
	gdipDeletePen            = gdiplusDLL.NewProc("GdipDeletePen")
	gdipSetPenLineCap197819  = gdiplusDLL.NewProc("GdipSetPenLineCap197819")
	gdipSetPenLineJoin       = gdiplusDLL.NewProc("GdipSetPenLineJoin")
	gdipDrawLineI            = gdiplusDLL.NewProc("GdipDrawLineI")
	gdipDrawEllipseI         = gdiplusDLL.NewProc("GdipDrawEllipseI")
	gdipFillEllipseI         = gdiplusDLL.NewProc("GdipFillEllipseI")
	gdipDrawBezierI          = gdiplusDLL.NewProc("GdipDrawBezierI")
	gdipCreatePath           = gdiplusDLL.NewProc("GdipCreatePath")
	gdipDeletePath           = gdiplusDLL.NewProc("GdipDeletePath")
	gdipAddPathArcI          = gdiplusDLL.NewProc("GdipAddPathArcI")
	gdipClosePathFigure      = gdiplusDLL.NewProc("GdipClosePathFigure")
	gdipDrawPath             = gdiplusDLL.NewProc("GdipDrawPath")
	gdipFillPath             = gdiplusDLL.NewProc("GdipFillPath")
	gdipCreateSolidFill      = gdiplusDLL.NewProc("GdipCreateSolidFill")
	gdipDeleteBrush          = gdiplusDLL.NewProc("GdipDeleteBrush")
)

// ============================================================================
// Win32 常量
// ============================================================================

const (
	wsPopup      = 0x80000000
	wsVisible    = 0x10000000
	wsExTopmost  = 0x00000008
	wsExToolWin  = 0x00000080

	swShow = 5

	wmDestroy      = 0x0002
	wmPaint        = 0x000F
	wmQuit         = 0x0012
	wmEraseBkgnd   = 0x0014
	wmSetCursor    = 0x0020
	wmKeyDown      = 0x0100
	wmChar         = 0x0102
	wmMouseMove    = 0x0200
	wmLButtonDown  = 0x0201
	wmLButtonUp    = 0x0202
	wmLButtonDblClk = 0x0203
	wmRButtonDown  = 0x0204

	vkEscape    = 0x1B
	vkReturn    = 0x0D
	vkBack      = 0x08
	vkControl   = 0x11
	vkShift     = 0x10
	vkZ         = 0x5A
	vkY         = 0x59
	vk1         = 0x31

	idcCross   = 32515
	idcArrow   = 32512
	idcSizeAll = 32646 // 移动光标
	idcSizeNWSE = 32642 // 左上↔右下
	idcSizeNESW = 32643 // 右上↔左下
	idcSizeNS   = 32645 // 上↔下
	idcSizeWE   = 32644 // 左↔右

	pmRemove = 0x0001

	smCxScreen       = 0
	smCyScreen       = 1
	smXVirtualScreen = 76
	smYVirtualScreen = 77

	srccopy    = 0x00CC0020
	biRGB      = 0
	dibRGBColors = 0

	psSOLID   = 0
	psDOT     = 2
	nullBrush = 5
	dcPen     = 19
	dcBrush   = 18

	transparent = 1

	// CS_DBLCLKS 窗口类样式：允许接收双击消息
	csDblClks = 0x0008
)

// ============================================================================
// Win32 结构体
// ============================================================================

type wndClassExW struct {
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

type msg struct {
	HWnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      point
}

type point struct {
	X int32
	Y int32
}

type rect struct {
	Left   int32
	Top    int32
	Right  int32
	Bottom int32
}

type paintStruct struct {
	Hdc         uintptr
	FErase      int32
	RcPaint     rect
	FRestore    int32
	FIncUpdate  int32
	RgbReserved [32]byte
}

type bitmapInfoHeader struct {
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

type bitmapInfo struct {
	BmiHeader bitmapInfoHeader
	BmiColors [1]uint32
}

// ============================================================================
// 浮动工具栏常量
// ============================================================================

const (
	// 主工具栏 (Apple dark mode style)
	toolbarBtnSize  = 48 // 按钮大小
	toolbarBtnGap   = 3  // 按钮间距
	toolbarPadding  = 10 // 工具栏内边距
	toolbarRadius   = 14 // 圆角半径 (pill shape)
	toolbarGap      = 10 // 工具栏与截图的间距
	toolbarSepWidth = 14 // 分隔符宽度
	toolbarHeight   = 58 // macOS 标准高度

	// 二级面板（颜色和线宽选择）
	subToolbarHeight = 48 // 二级面板高度
	subToolbarGap    = 6  // 二级面板与主工具栏间距
	subColorSize     = 28 // 颜色圆圈大小
	subColorGap      = 5  // 颜色间距
	subLineWidthGap  = 14 // 线宽按钮间距

	// GDI COLORREF 颜色 (0x00BBGGRR) - Apple dark mode palette
	colorToolbarBg     = 0x003C3A3A // #3A3A3C Apple dark mode card
	colorToolbarBorder = 0x005A5858 // #58585A subtle border
	colorToolbarHover  = 0x004E4C4C // #4C4C4E hover bg
	colorBtnSelectedBg = 0x005C3A1A // dark blue bg for selected tool
	colorIconNormal    = 0x00E0E0E0 // #E0E0E0 soft white
	colorIconHover     = 0x00FFFFFF // #FFFFFF pure white on hover
	colorIconSelected  = 0x00FFFFFF // white icon on blue bg
	colorAccent        = 0x00FF840A // #0A84FF Apple blue
	colorShadow        = 0x00181818 // shadow
	colorSeparator     = 0x00555555 // separator line
	colorSaveGreen     = 0x0059C734 // Apple green #34C759 → BGR
	colorCancelRed     = 0x003C3AFF // Apple red #FF3A3C → BGR

	// GDI+ 常量
	gdipSmoothingModeAntiAlias = 4
	gdipUnitPixel              = 2
	gdipLineCapRound           = 2
	gdipLineJoinRound          = 3
	gdipFillModeAlternate      = 0
)

// GDI+ 全局 token
var gdiplusToken uintptr

// gdipInit 初始化 GDI+
func gdipInit() {
	type startupInput struct {
		Version                uint32
		DebugEventCallback     uintptr
		SuppressBackgroundThread int32
		SuppressExternalCodecs int32
	}
	input := startupInput{Version: 1}
	gdiplusStartup.Call(
		uintptr(unsafe.Pointer(&gdiplusToken)),
		uintptr(unsafe.Pointer(&input)),
		0,
	)
}

// gdipShutdown 关闭 GDI+
func gdipShutdown() {
	if gdiplusToken != 0 {
		gdiplusShutdown.Call(gdiplusToken)
		gdiplusToken = 0
	}
}

// f32 将 float32 转为 uintptr（用于 GDI+ syscall 传递浮点参数）
func f32(v float32) uintptr {
	return uintptr(math.Float32bits(v))
}

// colorrefToARGB 将 GDI COLORREF (0x00BBGGRR) 转为 GDI+ ARGB (0xAARRGGBB)
func colorrefToARGB(c uintptr) uintptr {
	r := (c >> 0) & 0xFF
	g := (c >> 8) & 0xFF
	b := (c >> 16) & 0xFF
	return 0xFF000000 | (r << 16) | (g << 8) | b
}

// gdipNewGraphics 从 HDC 创建抗锯齿 GDI+ Graphics
func gdipNewGraphics(hdc uintptr) uintptr {
	var graphics uintptr
	gdipCreateFromHDC.Call(hdc, uintptr(unsafe.Pointer(&graphics)))
	gdipSetSmoothingMode.Call(graphics, gdipSmoothingModeAntiAlias)
	return graphics
}

// gdipNewPen 创建圆头圆角 GDI+ 画笔
func gdipNewPen(colorRef uintptr, width float32) uintptr {
	var pen uintptr
	gdipCreatePen1.Call(colorrefToARGB(colorRef), f32(width), gdipUnitPixel, uintptr(unsafe.Pointer(&pen)))
	gdipSetPenLineCap197819.Call(pen, gdipLineCapRound, gdipLineCapRound, 0)
	gdipSetPenLineJoin.Call(pen, gdipLineJoinRound)
	return pen
}

// gdipNewBrush 创建 GDI+ 实心画刷
func gdipNewBrush(colorRef uintptr) uintptr {
	var brush uintptr
	gdipCreateSolidFill.Call(colorrefToARGB(colorRef), uintptr(unsafe.Pointer(&brush)))
	return brush
}

// gdipDrawRoundRect 使用 GDI+ Path 绘制抗锯齿圆角矩形（空心）
func gdipDrawRoundRect(graphics, pen uintptr, x, y, w, h, r int) {
	var path uintptr
	gdipCreatePath.Call(gdipFillModeAlternate, uintptr(unsafe.Pointer(&path)))
	d := r * 2
	gdipAddPathArcI.Call(path, uintptr(x), uintptr(y), uintptr(d), uintptr(d), f32(180), f32(90))
	gdipAddPathArcI.Call(path, uintptr(x+w-d), uintptr(y), uintptr(d), uintptr(d), f32(270), f32(90))
	gdipAddPathArcI.Call(path, uintptr(x+w-d), uintptr(y+h-d), uintptr(d), uintptr(d), f32(0), f32(90))
	gdipAddPathArcI.Call(path, uintptr(x), uintptr(y+h-d), uintptr(d), uintptr(d), f32(90), f32(90))
	gdipClosePathFigure.Call(path)
	gdipDrawPath.Call(graphics, pen, path)
	gdipDeletePath.Call(path)
}

// gdipFillRoundRect 使用 GDI+ Path 填充抗锯齿圆角矩形
func gdipFillRoundRect(graphics, brush uintptr, x, y, w, h, r int) {
	var path uintptr
	gdipCreatePath.Call(gdipFillModeAlternate, uintptr(unsafe.Pointer(&path)))
	d := r * 2
	gdipAddPathArcI.Call(path, uintptr(x), uintptr(y), uintptr(d), uintptr(d), f32(180), f32(90))
	gdipAddPathArcI.Call(path, uintptr(x+w-d), uintptr(y), uintptr(d), uintptr(d), f32(270), f32(90))
	gdipAddPathArcI.Call(path, uintptr(x+w-d), uintptr(y+h-d), uintptr(d), uintptr(d), f32(0), f32(90))
	gdipAddPathArcI.Call(path, uintptr(x), uintptr(y+h-d), uintptr(d), uintptr(d), f32(90), f32(90))
	gdipClosePathFigure.Call(path)
	gdipFillPath.Call(graphics, brush, path)
	gdipDeletePath.Call(path)
}

// mainToolBtnOrder 主工具栏中工具按钮的显示顺序
// 矩形 → 椭圆 → 箭头 → 直线 → 画笔 → 文本 → 马赛克
var mainToolBtnOrder = []ToolType{
	ToolRect, ToolEllipse, ToolArrow, ToolLine, ToolFreehand, ToolText, ToolMosaic,
}

// subToolbarLineWidths 二级面板中的线宽选项 (2px / 4px / 8px)
var subToolbarLineWidths = []int{2, 4, 8}

// ============================================================================
// Editor 结构体
// ============================================================================

// Editor 标注编辑器
type Editor struct {
	hwnd         uintptr     // 编辑器窗口句柄
	fullscreen   *image.RGBA // 全屏截图引用（用于移动/调整选区时重新裁剪）
	background   *image.RGBA // 选区裁剪图（用于标注渲染）
	dimmedPixels []byte      // 预计算的暗化全屏像素 (BGRA格式，用于遮罩背景)
	history      *History    // 标注历史

	// 当前工具状态
	currentTool  ToolType
	currentColor color.RGBA
	lineWidth    int
	fontSize     int

	// 绘制状态
	drawing        bool           // 是否正在绘制
	startPt        image.Point    // 起始点
	currentPt      image.Point    // 当前点
	tempAnnotation *Annotation    // 正在绘制的临时标注（用于预览）
	freehandPts    []image.Point  // 自由画笔的点集

	// 文本输入状态
	textInput  bool        // 是否在文本输入模式
	textBuffer string      // 文本缓冲
	textPos    image.Point // 文本位置

	// 选区拖拽状态（移动/调整大小）
	draggingSelection bool            // 是否正在拖拽选区
	dragMode          int             // 0=移动, 1-8=调整手柄
	dragStartMouse    image.Point     // 拖拽起始鼠标位置
	dragStartRect     image.Rectangle // 拖拽起始选区矩形

	// UI 布局
	screenWidth  int // 屏幕宽度
	screenHeight int // 屏幕高度

	// 截图在屏幕上的位置
	imageRect image.Rectangle // 截图在屏幕上的矩形区域

	// 浮动工具栏布局
	mainToolbarRect image.Rectangle // 主工具栏位置
	subToolbarRect  image.Rectangle // 二级面板位置
	showSubToolbar  bool            // 是否显示二级面板
	hoverBtnIndex   int             // 当前悬停的主工具栏按钮索引 (-1=无)
	hoverSubIndex   int             // 当前悬停的二级面板按钮索引 (-1=无)

	// GDI 缓存
	memDC      uintptr
	memBitmap  uintptr
	memBits    uintptr
	bufWidth   int
	bufHeight  int

	// 结果
	result *EditorResult
	done   bool
}

// ============================================================================
// 工具栏按钮定义
// ============================================================================

// toolbarButton 工具栏按钮
type toolbarButton struct {
	x, y, w, h int
	kind        string // "tool", "color", "linewidth", "action"
	toolType    ToolType
	colorIndex  int
	lineWidth   int
	action      string // "undo", "redo", "save", "cancel"
}

// ============================================================================
// 全局状态（用于窗口回调）
// ============================================================================

var (
	editorMutex           sync.Mutex
	editorInstance        *Editor
	editorClassRegistered bool
)

// DebugLogFunc 调试日志回调（由主程序设置）
var DebugLogFunc func(format string, args ...interface{})

// ============================================================================
// 公开接口
// ============================================================================

// OpenEditor 打开标注编辑器，返回编辑结果
// fullscreen 是全屏截图，selection 是用户选择的区域矩形
func OpenEditor(fullscreen *image.RGBA, selection image.Rectangle) *EditorResult {
	editorMutex.Lock()
	defer editorMutex.Unlock()

	// 锁定当前 goroutine 到 OS 线程，确保 Win32 窗口消息循环的线程亲和性
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	// 使用全屏截图实际尺寸作为屏幕尺寸，确保坐标系与截图完全一致（避免 DPI 缩放偏移）
	screenW := fullscreen.Bounds().Dx()
	screenH := fullscreen.Bounds().Dy()

	if DebugLogFunc != nil {
		DebugLogFunc("编辑器: fullscreen=%dx%d, selection=%v", screenW, screenH, selection)
	}

	// 选区与全屏图片的交集
	sel := selection.Intersect(fullscreen.Bounds())
	selW := sel.Dx()
	selH := sel.Dy()
	if selW <= 0 || selH <= 0 {
		return &EditorResult{Cancelled: true}
	}

	if DebugLogFunc != nil {
		DebugLogFunc("编辑器: sel=%v, selW=%d, selH=%d", sel, selW, selH)
	}

	// 从全屏截图中裁剪选区（用于标注渲染）
	background := image.NewRGBA(image.Rect(0, 0, selW, selH))
	for y := 0; y < selH; y++ {
		srcOff := (sel.Min.Y+y)*fullscreen.Stride + sel.Min.X*4
		dstOff := y * background.Stride
		copy(background.Pix[dstOff:dstOff+selW*4], fullscreen.Pix[srcOff:srcOff+selW*4])
	}

	// 预计算暗化的全屏像素 (BGRA格式, 50%亮度) 用于遮罩背景
	dimmedPixels := make([]byte, screenW*screenH*4)
	fb := fullscreen.Bounds()
	for y := 0; y < fb.Dy() && y < screenH; y++ {
		for x := 0; x < fb.Dx() && x < screenW; x++ {
			si := y*fullscreen.Stride + x*4
			di := (y*screenW + x) * 4
			// RGBA -> BGRA, 50% brightness
			dimmedPixels[di+0] = fullscreen.Pix[si+2] >> 1 // B
			dimmedPixels[di+1] = fullscreen.Pix[si+1] >> 1 // G
			dimmedPixels[di+2] = fullscreen.Pix[si+0] >> 1 // R
			dimmedPixels[di+3] = 255                         // A
		}
	}

	e := &Editor{
		fullscreen:    fullscreen,
		background:    background,
		dimmedPixels:  dimmedPixels,
		history:       NewHistory(50),
		currentTool:   ToolRect,
		currentColor:  DefaultColors[0], // 默认红色
		lineWidth:     subToolbarLineWidths[0],
		fontSize:      DefaultFontSizes[1],
		hoverBtnIndex: -1,
		hoverSubIndex: -1,
		screenWidth:   screenW,
		screenHeight:  screenH,
		imageRect:     sel, // 选区在屏幕上的原始位置
		done:          false,
	}

	editorInstance = e

	// 初始化 GDI+（用于抗锯齿图标绘制）
	gdipInit()
	defer gdipShutdown()

	// 清理消息队列中可能残留的 WM_QUIT 消息
	editorDrainQuitMessages()

	// 计算浮动工具栏位置
	e.calculateToolbarPosition()
	// 默认选中绘图工具时显示二级面板
	e.updateSubToolbarVisibility()

	// 获取模块句柄
	hInstance, _, _ := getModuleHandle.Call(0)

	// 注册窗口类（只注册一次，带 CS_DBLCLKS 支持双击）
	className := syscall.StringToUTF16Ptr("SnapCLIEditorClass")
	if !editorClassRegistered {
		var wc wndClassExW
		wc.CbSize = uint32(unsafe.Sizeof(wc))
		wc.Style = csDblClks // 允许双击消息
		wc.LpfnWndProc = syscall.NewCallback(editorWndProc)
		wc.HInstance = hInstance
		wc.HCursor, _, _ = loadCursorW.Call(0, uintptr(idcArrow))
		wc.LpszClassName = className

		registerClassExW.Call(uintptr(unsafe.Pointer(&wc)))
		editorClassRegistered = true
	}

	// 获取虚拟屏幕原点（与选区器保持一致的坐标系）
	// 注意: GetSystemMetrics 返回 int32，但 Call() 返回 uintptr，必须做符号扩展
	vsXRaw, _, _ := getSystemMetrics.Call(smXVirtualScreen)
	vsYRaw, _, _ := getSystemMetrics.Call(smYVirtualScreen)
	vsX := int(int32(vsXRaw))
	vsY := int(int32(vsYRaw))

	if DebugLogFunc != nil {
		DebugLogFunc("编辑器窗口: vsX=%d, vsY=%d (raw: %d, %d)", vsX, vsY, vsXRaw, vsYRaw)
		DebugLogFunc("编辑器窗口尺寸: screenW=%d, screenH=%d", e.screenWidth, e.screenHeight)
		DebugLogFunc("imageRect=%v", e.imageRect)
	}

	// 创建全屏置顶窗口
	hwnd, _, _ := createWindowExW.Call(
		wsExTopmost|wsExToolWin,
		uintptr(unsafe.Pointer(className)),
		0,
		wsPopup|wsVisible,
		uintptr(vsX), uintptr(vsY),
		uintptr(e.screenWidth), uintptr(e.screenHeight),
		0, 0, hInstance, 0,
	)

	if hwnd == 0 {
		return &EditorResult{Cancelled: true}
	}

	e.hwnd = hwnd

	showWindow.Call(hwnd, swShow)
	updateWindow.Call(hwnd)
	setForegroundWindow.Call(hwnd)
	setFocus.Call(hwnd)
	invalidateRect.Call(hwnd, 0, 1)

	// 消息循环
	var m msg
	for !e.done {
		ret, _, _ := getMessageW.Call(
			uintptr(unsafe.Pointer(&m)),
			0, 0, 0,
		)
		if ret == 0 || ret == uintptr(^uintptr(0)) {
			break
		}
		translateMessage.Call(uintptr(unsafe.Pointer(&m)))
		dispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}

	destroyWindow.Call(hwnd)

	// 清理 GDI 缓存
	e.cleanupGDICache()

	result := e.result
	e.hwnd = 0
	e.fullscreen = nil
	e.background = nil
	e.dimmedPixels = nil
	editorInstance = nil

	if result == nil {
		return &EditorResult{Cancelled: true}
	}
	return result
}

// ============================================================================
// 工具栏位置计算
// ============================================================================

// calculateToolbarPosition 计算浮动工具栏位置
func (e *Editor) calculateToolbarPosition() {
	// 计算主工具栏宽度
	// 7个工具按钮 + 分隔符 + 撤销/重做 + 分隔符 + 保存/取消
	numTools := len(mainToolBtnOrder)
	toolsWidth := numTools*toolbarBtnSize + (numTools-1)*toolbarBtnGap
	undoRedoWidth := 2*toolbarBtnSize + toolbarBtnGap
	saveWidth := 2*toolbarBtnSize + toolbarBtnGap
	totalBtnWidth := toolsWidth + toolbarSepWidth + undoRedoWidth + toolbarSepWidth + saveWidth
	mainW := toolbarPadding + totalBtnWidth + toolbarPadding
	mainH := toolbarHeight

	// 位置: 截图底部右对齐, 向下偏移 toolbarGap
	mainLeft := e.imageRect.Max.X - mainW
	mainTop := e.imageRect.Max.Y + toolbarGap

	// 边界检测: 如果向左溢出，调整到左边缘
	if mainLeft < 0 {
		mainLeft = 0
	}

	// 如果超出屏幕底部则翻转到截图上方
	if mainTop+mainH > e.screenHeight {
		mainTop = e.imageRect.Min.Y - toolbarGap - mainH
		if mainTop < 0 {
			mainTop = 0
		}
	}

	// 如果向右溢出
	if mainLeft+mainW > e.screenWidth {
		mainLeft = e.screenWidth - mainW
		if mainLeft < 0 {
			mainLeft = 0
		}
	}

	e.mainToolbarRect = image.Rect(mainLeft, mainTop, mainLeft+mainW, mainTop+mainH)

	// 计算二级面板位置: 主工具栏上方 subToolbarGap 处
	e.calculateSubToolbarPosition()
}

// calculateSubToolbarPosition 计算二级面板位置
func (e *Editor) calculateSubToolbarPosition() {
	// 二级面板内容: 3个线宽圆点 + 分隔符 + 8个颜色方块
	numLineWidths := len(subToolbarLineWidths)
	numColors := len(DefaultColors)

	// 线宽区域: 每个圆点占 subColorSize 宽度, 间距 subLineWidthGap
	lineWidthAreaW := numLineWidths*subColorSize + (numLineWidths-1)*subLineWidthGap
	// 颜色区域: 每个方块 subColorSize, 间距 subColorGap
	colorAreaW := numColors*subColorSize + (numColors-1)*subColorGap

	subW := toolbarPadding + lineWidthAreaW + toolbarSepWidth + colorAreaW + toolbarPadding
	subH := subToolbarHeight

	// 右对齐到主工具栏
	subLeft := e.mainToolbarRect.Max.X - subW
	subTop := e.mainToolbarRect.Min.Y - subToolbarGap - subH

	// 边界检测
	if subLeft < 0 {
		subLeft = 0
	}
	if subTop < 0 {
		subTop = 0
	}

	e.subToolbarRect = image.Rect(subLeft, subTop, subLeft+subW, subTop+subH)
}

// updateSubToolbarVisibility 根据当前工具更新二级面板可见性
func (e *Editor) updateSubToolbarVisibility() {
	switch e.currentTool {
	case ToolRect, ToolEllipse, ToolArrow, ToolLine, ToolFreehand, ToolMosaic:
		e.showSubToolbar = true
	default:
		e.showSubToolbar = false
	}
}

// ============================================================================
// 窗口过程
// ============================================================================

func editorWndProc(hwnd, umsg, wParam, lParam uintptr) uintptr {
	e := editorInstance
	if e == nil {
		ret, _, _ := defWindowProcW.Call(hwnd, umsg, wParam, lParam)
		return ret
	}

	switch umsg {
	case wmSetCursor:
		// 根据鼠标位置设置光标
		var pt point
		getCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
		screenToClientProc.Call(hwnd, uintptr(unsafe.Pointer(&pt)))
		mx, my := int(pt.X), int(pt.Y)

		cursorID := uintptr(idcArrow)

		// 优先检查工具栏区域
		if e.isInMainToolbar(mx, my) || e.isInSubToolbar(mx, my) {
			cursorID = uintptr(idcArrow)
		} else if hIdx := e.hitTestHandle(mx, my); hIdx >= 0 {
			// 在手柄上：显示调整光标
			cursorID = uintptr(e.handleCursor(hIdx))
		} else if e.isOnSelectionBorder(mx, my) {
			// 在选区边框上：显示移动光标
			cursorID = uintptr(idcSizeAll)
		} else if mx >= e.imageRect.Min.X && mx < e.imageRect.Max.X &&
			my >= e.imageRect.Min.Y && my < e.imageRect.Max.Y {
			// 在选区内部：显示十字光标
			cursorID = uintptr(idcCross)
		}

		cursor, _, _ := loadCursorW.Call(0, cursorID)
		setCursor.Call(cursor)
		return 1

	case wmEraseBkgnd:
		return 1 // 阻止系统擦除背景

	case wmPaint:
		var ps paintStruct
		hdc, _, _ := beginPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
		e.onPaint(hdc)
		endPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
		return 0

	case wmKeyDown:
		e.onKeyDown(wParam)
		return 0

	case wmChar:
		e.onChar(wParam)
		return 0

	case wmLButtonDown:
		mx := int(int16(lParam & 0xFFFF))
		my := int(int16((lParam >> 16) & 0xFFFF))
		e.onLButtonDown(mx, my, hwnd)
		return 0

	case wmLButtonDblClk:
		// 双击画布区域：保存并退出
		mx := int(int16(lParam & 0xFFFF))
		my := int(int16((lParam >> 16) & 0xFFFF))
		if !e.isInMainToolbar(mx, my) && !e.isInSubToolbar(mx, my) {
			e.saveAndExit()
		}
		return 0

	case wmMouseMove:
		mx := int(int16(lParam & 0xFFFF))
		my := int(int16((lParam >> 16) & 0xFFFF))
		e.onMouseMove(mx, my)
		return 0

	case wmLButtonUp:
		mx := int(int16(lParam & 0xFFFF))
		my := int(int16((lParam >> 16) & 0xFFFF))
		e.onLButtonUp(mx, my)
		return 0

	case wmRButtonDown:
		// 右键：取消当前绘制或关闭编辑器
		if e.drawing {
			e.drawing = false
			e.tempAnnotation = nil
			e.freehandPts = nil
			invalidateRect.Call(hwnd, 0, 0)
		} else if e.textInput {
			e.textInput = false
			e.textBuffer = ""
			invalidateRect.Call(hwnd, 0, 0)
		} else {
			e.result = &EditorResult{Cancelled: true}
			e.done = true
			postQuitMessage.Call(0)
		}
		return 0

	case wmDestroy:
		return 0
	}

	ret, _, _ := defWindowProcW.Call(hwnd, umsg, wParam, lParam)
	return ret
}

// ============================================================================
// 事件处理
// ============================================================================

// onKeyDown 处理键盘按下事件
func (e *Editor) onKeyDown(wParam uintptr) {
	// 检查 Ctrl 键状态
	ctrlState, _, _ := getKeyState.Call(uintptr(vkControl))
	ctrlDown := int16(ctrlState) < 0
	shiftState, _, _ := getKeyState.Call(uintptr(vkShift))
	shiftDown := int16(shiftState) < 0

	switch {
	case wParam == uintptr(vkEscape):
		if e.textInput {
			e.textInput = false
			e.textBuffer = ""
			invalidateRect.Call(e.hwnd, 0, 0)
		} else if e.drawing {
			e.drawing = false
			e.tempAnnotation = nil
			e.freehandPts = nil
			invalidateRect.Call(e.hwnd, 0, 0)
		} else {
			e.result = &EditorResult{Cancelled: true}
			e.done = true
			postQuitMessage.Call(0)
		}

	case wParam == uintptr(vkReturn):
		if e.textInput {
			e.commitText()
		} else {
			e.saveAndExit()
		}

	case ctrlDown && wParam == uintptr(vkZ):
		if shiftDown {
			e.history.Redo()
			invalidateRect.Call(e.hwnd, 0, 0)
		} else {
			e.history.Undo()
			invalidateRect.Call(e.hwnd, 0, 0)
		}

	case ctrlDown && wParam == uintptr(vkY):
		e.history.Redo()
		invalidateRect.Call(e.hwnd, 0, 0)

	case !ctrlDown && !e.textInput:
		// 数字键 1-7 切换工具（按新顺序）
		key := int(wParam)
		if key >= vk1 && key < vk1+len(mainToolBtnOrder) {
			e.currentTool = mainToolBtnOrder[key-vk1]
			e.updateSubToolbarVisibility()
			invalidateRect.Call(e.hwnd, 0, 0)
		}
	}
}

// onChar 处理字符输入（文本模式）
func (e *Editor) onChar(wParam uintptr) {
	if !e.textInput {
		return
	}

	ch := rune(wParam)
	switch {
	case ch == '\b':
		if len(e.textBuffer) > 0 {
			runes := []rune(e.textBuffer)
			e.textBuffer = string(runes[:len(runes)-1])
			invalidateRect.Call(e.hwnd, 0, 0)
		}
	case ch == '\r' || ch == '\n':
		// Enter 已在 WM_KEYDOWN 中处理
	case ch == 27:
		// ESC 已在 WM_KEYDOWN 中处理
	case ch >= 32:
		e.textBuffer += string(ch)
		invalidateRect.Call(e.hwnd, 0, 0)
	}
}

// onLButtonDown 处理鼠标左键按下
func (e *Editor) onLButtonDown(mx, my int, hwnd uintptr) {
	// 1. 检查是否点击二级面板
	if e.showSubToolbar && e.isInSubToolbar(mx, my) {
		e.handleSubToolbarClick(mx, my)
		return
	}

	// 2. 检查是否点击主工具栏
	if e.isInMainToolbar(mx, my) {
		e.handleMainToolbarClick(mx, my)
		return
	}

	// 3. 检查是否拖拽手柄（调整选区大小）
	if hIdx := e.hitTestHandle(mx, my); hIdx >= 0 {
		e.draggingSelection = true
		e.dragMode = hIdx + 1 // 1-8 表示手柄
		e.dragStartMouse = image.Point{X: mx, Y: my}
		e.dragStartRect = e.imageRect
		setCapture.Call(hwnd)
		return
	}

	// 4. 检查是否拖拽选区边框（移动选区）
	if e.isOnSelectionBorder(mx, my) {
		e.draggingSelection = true
		e.dragMode = 0 // 0 表示移动
		e.dragStartMouse = image.Point{X: mx, Y: my}
		e.dragStartRect = e.imageRect
		setCapture.Call(hwnd)
		return
	}

	// 5. 否则处理画布绘制
	// 如果在文本输入模式，先提交当前文本
	if e.textInput {
		e.commitText()
	}

	// 转换为画布坐标
	cx, cy := e.screenToCanvas(mx, my)
	if !e.isInCanvas(cx, cy) {
		return
	}

	if e.currentTool == ToolText {
		e.textInput = true
		e.textBuffer = ""
		e.textPos = image.Point{X: cx, Y: cy}
		invalidateRect.Call(e.hwnd, 0, 0)
		return
	}

	// 开始绘制
	e.drawing = true
	e.startPt = image.Point{X: cx, Y: cy}
	e.currentPt = e.startPt

	if e.currentTool == ToolFreehand {
		e.freehandPts = []image.Point{e.startPt}
	}

	e.updateTempAnnotation()
	setCapture.Call(hwnd)
}

// invalidateToolbarArea 仅失效工具栏区域（避免整个3840x1281窗口重绘）
func (e *Editor) invalidateToolbarArea() {
	// 失效主工具栏区域（带4px边距覆盖阴影）
	r := rect{
		Left:   int32(e.mainToolbarRect.Min.X - 4),
		Top:    int32(e.mainToolbarRect.Min.Y - 4),
		Right:  int32(e.mainToolbarRect.Max.X + 4),
		Bottom: int32(e.mainToolbarRect.Max.Y + 4),
	}
	invalidateRect.Call(e.hwnd, uintptr(unsafe.Pointer(&r)), 0)

	// 如果二级面板可见，也失效其区域
	if e.showSubToolbar {
		r2 := rect{
			Left:   int32(e.subToolbarRect.Min.X - 4),
			Top:    int32(e.subToolbarRect.Min.Y - 4),
			Right:  int32(e.subToolbarRect.Max.X + 4),
			Bottom: int32(e.subToolbarRect.Max.Y + 4),
		}
		invalidateRect.Call(e.hwnd, uintptr(unsafe.Pointer(&r2)), 0)
	}
}

// onMouseMove 处理鼠标移动
func (e *Editor) onMouseMove(mx, my int) {
	// 处理选区拖拽（移动/调整大小）
	if e.draggingSelection {
		dx := mx - e.dragStartMouse.X
		dy := my - e.dragStartMouse.Y
		e.applySelectionDrag(dx, dy)
		invalidateRect.Call(e.hwnd, 0, 0)
		return
	}

	// 更新主工具栏悬停状态
	oldHover := e.hoverBtnIndex
	oldSubHover := e.hoverSubIndex
	e.hoverBtnIndex = -1
	e.hoverSubIndex = -1

	if e.isInMainToolbar(mx, my) {
		btns := e.getMainToolbarButtons()
		for i, btn := range btns {
			if mx >= btn.x && mx < btn.x+btn.w && my >= btn.y && my < btn.y+btn.h {
				e.hoverBtnIndex = i
				break
			}
		}
	} else if e.showSubToolbar && e.isInSubToolbar(mx, my) {
		btns := e.getSubToolbarButtons()
		for i, btn := range btns {
			if mx >= btn.x && mx < btn.x+btn.w && my >= btn.y && my < btn.y+btn.h {
				e.hoverSubIndex = i
				break
			}
		}
	}

	// 如果悬停状态变化，仅重绘工具栏区域（避免闪烁）
	if oldHover != e.hoverBtnIndex || oldSubHover != e.hoverSubIndex {
		e.invalidateToolbarArea()
	}

	// 处理绘制拖拽
	if !e.drawing {
		return
	}

	cx, cy := e.screenToCanvas(mx, my)
	e.currentPt = image.Point{X: cx, Y: cy}

	if e.currentTool == ToolFreehand {
		e.freehandPts = append(e.freehandPts, e.currentPt)
	}

	e.updateTempAnnotation()
	invalidateRect.Call(e.hwnd, 0, 0)
}

// onLButtonUp 处理鼠标左键释放
func (e *Editor) onLButtonUp(mx, my int) {
	// 结束选区拖拽
	if e.draggingSelection {
		releaseCapture.Call()
		e.draggingSelection = false
		// 重新裁剪背景、重新计算工具栏位置
		e.updateSelectionCrop()
		e.calculateToolbarPosition()
		invalidateRect.Call(e.hwnd, 0, 0)
		return
	}

	if !e.drawing {
		return
	}

	releaseCapture.Call()
	e.drawing = false

	cx, cy := e.screenToCanvas(mx, my)
	e.currentPt = image.Point{X: cx, Y: cy}

	if e.currentTool == ToolFreehand {
		e.freehandPts = append(e.freehandPts, e.currentPt)
	}

	// 完成标注，添加到历史
	e.updateTempAnnotation()
	if e.tempAnnotation != nil {
		if e.currentTool == ToolFreehand {
			if len(e.freehandPts) > 2 {
				e.history.AddAnnotation(*e.tempAnnotation)
			}
		} else {
			dx := e.currentPt.X - e.startPt.X
			dy := e.currentPt.Y - e.startPt.Y
			if dx*dx+dy*dy > 9 {
				e.history.AddAnnotation(*e.tempAnnotation)
			}
		}
	}

	e.tempAnnotation = nil
	e.freehandPts = nil
	invalidateRect.Call(e.hwnd, 0, 0)
}

// ============================================================================
// 工具栏 Hit Test
// ============================================================================

// isInMainToolbar 检查鼠标是否在主工具栏区域内
func (e *Editor) isInMainToolbar(mx, my int) bool {
	r := e.mainToolbarRect
	return mx >= r.Min.X && mx < r.Max.X && my >= r.Min.Y && my < r.Max.Y
}

// isInSubToolbar 检查鼠标是否在二级面板区域内
func (e *Editor) isInSubToolbar(mx, my int) bool {
	if !e.showSubToolbar {
		return false
	}
	r := e.subToolbarRect
	return mx >= r.Min.X && mx < r.Max.X && my >= r.Min.Y && my < r.Max.Y
}

// ============================================================================
// 工具栏按钮布局
// ============================================================================

// getMainToolbarButtons 计算主工具栏按钮的布局位置（使用绝对屏幕坐标）
func (e *Editor) getMainToolbarButtons() []toolbarButton {
	var buttons []toolbarButton
	baseX := e.mainToolbarRect.Min.X + toolbarPadding
	baseY := e.mainToolbarRect.Min.Y + (toolbarHeight-toolbarBtnSize)/2

	x := baseX

	// 工具按钮组（按指定顺序）
	for _, tt := range mainToolBtnOrder {
		buttons = append(buttons, toolbarButton{
			x: x, y: baseY, w: toolbarBtnSize, h: toolbarBtnSize,
			kind:     "tool",
			toolType: tt,
		})
		x += toolbarBtnSize + toolbarBtnGap
	}

	// 分隔符
	x += toolbarSepWidth - toolbarBtnGap

	// 撤销
	buttons = append(buttons, toolbarButton{
		x: x, y: baseY, w: toolbarBtnSize, h: toolbarBtnSize,
		kind:   "action",
		action: "undo",
	})
	x += toolbarBtnSize + toolbarBtnGap

	// 重做
	buttons = append(buttons, toolbarButton{
		x: x, y: baseY, w: toolbarBtnSize, h: toolbarBtnSize,
		kind:   "action",
		action: "redo",
	})
	x += toolbarBtnSize + toolbarBtnGap

	// 分隔符
	x += toolbarSepWidth - toolbarBtnGap

	// 保存按钮（绿色 ✓）
	buttons = append(buttons, toolbarButton{
		x: x, y: baseY, w: toolbarBtnSize, h: toolbarBtnSize,
		kind:   "action",
		action: "save",
	})
	x += toolbarBtnSize + toolbarBtnGap

	// 取消按钮（红色 X）
	buttons = append(buttons, toolbarButton{
		x: x, y: baseY, w: toolbarBtnSize, h: toolbarBtnSize,
		kind:   "action",
		action: "cancel",
	})

	return buttons
}

// getSubToolbarButtons 计算二级面板按钮的布局位置
func (e *Editor) getSubToolbarButtons() []toolbarButton {
	var buttons []toolbarButton
	baseX := e.subToolbarRect.Min.X + toolbarPadding
	baseY := e.subToolbarRect.Min.Y + (subToolbarHeight-subColorSize)/2

	x := baseX

	// 线宽选择 (3个)
	for _, lw := range subToolbarLineWidths {
		buttons = append(buttons, toolbarButton{
			x: x, y: baseY, w: subColorSize, h: subColorSize,
			kind:      "linewidth",
			lineWidth: lw,
		})
		x += subColorSize + subLineWidthGap
	}

	// 分隔符
	x += toolbarSepWidth - subLineWidthGap

	// 颜色选择 (8个)
	for i := range DefaultColors {
		buttons = append(buttons, toolbarButton{
			x: x, y: baseY, w: subColorSize, h: subColorSize,
			kind:       "color",
			colorIndex: i,
		})
		x += subColorSize + subColorGap
	}

	return buttons
}

// ============================================================================
// 工具栏点击处理
// ============================================================================

// handleMainToolbarClick 处理主工具栏点击
func (e *Editor) handleMainToolbarClick(mx, my int) {
	buttons := e.getMainToolbarButtons()
	for _, btn := range buttons {
		if mx >= btn.x && mx < btn.x+btn.w && my >= btn.y && my < btn.y+btn.h {
			switch btn.kind {
			case "tool":
				if e.textInput {
					e.commitText()
				}
				e.currentTool = btn.toolType
				e.updateSubToolbarVisibility()
			case "action":
				switch btn.action {
				case "undo":
					e.history.Undo()
				case "redo":
					e.history.Redo()
				case "save":
					e.saveAndExit()
					return
				case "cancel":
					e.result = &EditorResult{Cancelled: true}
					e.done = true
					postQuitMessage.Call(0)
					return
				}
			}
			invalidateRect.Call(e.hwnd, 0, 0)
			return
		}
	}
}

// handleSubToolbarClick 处理二级面板点击
func (e *Editor) handleSubToolbarClick(mx, my int) {
	buttons := e.getSubToolbarButtons()
	for _, btn := range buttons {
		if mx >= btn.x && mx < btn.x+btn.w && my >= btn.y && my < btn.y+btn.h {
			switch btn.kind {
			case "linewidth":
				e.lineWidth = btn.lineWidth
			case "color":
				if btn.colorIndex >= 0 && btn.colorIndex < len(DefaultColors) {
					e.currentColor = DefaultColors[btn.colorIndex]
				}
			}
			invalidateRect.Call(e.hwnd, 0, 0)
			return
		}
	}
}

// ============================================================================
// 坐标转换
// ============================================================================

// screenToCanvas 屏幕坐标转画布坐标
func (e *Editor) screenToCanvas(sx, sy int) (int, int) {
	return sx - e.imageRect.Min.X, sy - e.imageRect.Min.Y
}

// canvasToScreen 画布坐标转屏幕坐标
func (e *Editor) canvasToScreen(cx, cy int) (int, int) {
	return cx + e.imageRect.Min.X, cy + e.imageRect.Min.Y
}

// isInCanvas 检查画布坐标是否在截图范围内
func (e *Editor) isInCanvas(cx, cy int) bool {
	b := e.background.Bounds()
	return cx >= 0 && cy >= 0 && cx < b.Dx() && cy < b.Dy()
}

// ============================================================================
// 选区拖拽（移动/调整大小）
// ============================================================================

const borderHitSize = 5 // 边框命中区域宽度（像素）
const handleHitSize = 8 // 手柄命中区域半径（像素）

// hitTestHandle 检测鼠标是否在某个调整手柄上，返回 0-7 或 -1
// 0=左上, 1=上中, 2=右上, 3=右中, 4=右下, 5=下中, 6=左下, 7=左中
func (e *Editor) hitTestHandle(mx, my int) int {
	r := e.imageRect
	midX := (r.Min.X + r.Max.X) / 2
	midY := (r.Min.Y + r.Max.Y) / 2

	handles := [8]image.Point{
		{r.Min.X, r.Min.Y}, // 0: 左上
		{midX, r.Min.Y},    // 1: 上中
		{r.Max.X, r.Min.Y}, // 2: 右上
		{r.Max.X, midY},    // 3: 右中
		{r.Max.X, r.Max.Y}, // 4: 右下
		{midX, r.Max.Y},    // 5: 下中
		{r.Min.X, r.Max.Y}, // 6: 左下
		{r.Min.X, midY},    // 7: 左中
	}

	for i, h := range handles {
		if mx >= h.X-handleHitSize && mx <= h.X+handleHitSize &&
			my >= h.Y-handleHitSize && my <= h.Y+handleHitSize {
			return i
		}
	}
	return -1
}

// isOnSelectionBorder 检测鼠标是否在选区边框上
func (e *Editor) isOnSelectionBorder(mx, my int) bool {
	r := e.imageRect
	outer := image.Rect(r.Min.X-borderHitSize, r.Min.Y-borderHitSize,
		r.Max.X+borderHitSize, r.Max.Y+borderHitSize)
	inner := image.Rect(r.Min.X+borderHitSize, r.Min.Y+borderHitSize,
		r.Max.X-borderHitSize, r.Max.Y-borderHitSize)
	pt := image.Point{X: mx, Y: my}
	return pt.In(outer) && !pt.In(inner)
}

// handleCursor 根据手柄索引返回对应的光标 ID
func (e *Editor) handleCursor(hIdx int) int {
	switch hIdx {
	case 0, 4: // 左上、右下
		return idcSizeNWSE
	case 2, 6: // 右上、左下
		return idcSizeNESW
	case 1, 5: // 上中、下中
		return idcSizeNS
	case 3, 7: // 右中、左中
		return idcSizeWE
	default:
		return idcArrow
	}
}

// applySelectionDrag 应用选区拖拽偏移
func (e *Editor) applySelectionDrag(dx, dy int) {
	r := e.dragStartRect

	if e.dragMode == 0 {
		// 移动模式：整体偏移
		newRect := r.Add(image.Point{X: dx, Y: dy})
		// 限制在屏幕范围内
		if newRect.Min.X < 0 {
			newRect = newRect.Add(image.Point{X: -newRect.Min.X})
		}
		if newRect.Min.Y < 0 {
			newRect = newRect.Add(image.Point{Y: -newRect.Min.Y})
		}
		if newRect.Max.X > e.screenWidth {
			newRect = newRect.Sub(image.Point{X: newRect.Max.X - e.screenWidth})
		}
		if newRect.Max.Y > e.screenHeight {
			newRect = newRect.Sub(image.Point{Y: newRect.Max.Y - e.screenHeight})
		}
		e.imageRect = newRect
	} else {
		// 调整大小模式：根据手柄索引修改对应边
		switch e.dragMode {
		case 1: // 左上
			r.Min.X += dx; r.Min.Y += dy
		case 2: // 上中
			r.Min.Y += dy
		case 3: // 右上
			r.Max.X += dx; r.Min.Y += dy
		case 4: // 右中
			r.Max.X += dx
		case 5: // 右下
			r.Max.X += dx; r.Max.Y += dy
		case 6: // 下中
			r.Max.Y += dy
		case 7: // 左下
			r.Min.X += dx; r.Max.Y += dy
		case 8: // 左中
			r.Min.X += dx
		}
		// 确保最小尺寸
		if r.Dx() < 20 {
			if e.dragMode == 1 || e.dragMode == 7 || e.dragMode == 8 {
				r.Min.X = r.Max.X - 20
			} else {
				r.Max.X = r.Min.X + 20
			}
		}
		if r.Dy() < 20 {
			if e.dragMode == 1 || e.dragMode == 2 || e.dragMode == 3 {
				r.Min.Y = r.Max.Y - 20
			} else {
				r.Max.Y = r.Min.Y + 20
			}
		}
		// 限制在全屏图片范围内
		fb := e.fullscreen.Bounds()
		if r.Min.X < fb.Min.X { r.Min.X = fb.Min.X }
		if r.Min.Y < fb.Min.Y { r.Min.Y = fb.Min.Y }
		if r.Max.X > fb.Max.X { r.Max.X = fb.Max.X }
		if r.Max.Y > fb.Max.Y { r.Max.Y = fb.Max.Y }
		e.imageRect = r
	}

	// 实时更新工具栏位置
	e.calculateToolbarPosition()
}

// updateSelectionCrop 重新从全屏截图裁剪选区
func (e *Editor) updateSelectionCrop() {
	sel := e.imageRect.Intersect(e.fullscreen.Bounds())
	selW := sel.Dx()
	selH := sel.Dy()
	if selW <= 0 || selH <= 0 {
		return
	}

	e.background = image.NewRGBA(image.Rect(0, 0, selW, selH))
	for y := 0; y < selH; y++ {
		srcOff := (sel.Min.Y+y)*e.fullscreen.Stride + sel.Min.X*4
		dstOff := y * e.background.Stride
		copy(e.background.Pix[dstOff:dstOff+selW*4], e.fullscreen.Pix[srcOff:srcOff+selW*4])
	}
}

// ============================================================================
// 标注逻辑
// ============================================================================

// updateTempAnnotation 更新临时标注（预览用）
func (e *Editor) updateTempAnnotation() {
	a := &Annotation{
		Type:      e.currentTool,
		Color:     e.currentColor,
		LineWidth: e.lineWidth,
		FontSize:  e.fontSize,
		MosaicPx:  12,
	}

	switch e.currentTool {
	case ToolFreehand:
		a.Points = make([]image.Point, len(e.freehandPts))
		copy(a.Points, e.freehandPts)
	default:
		a.Points = []image.Point{e.startPt, e.currentPt}
	}

	e.tempAnnotation = a
}

// commitText 提交文本输入
func (e *Editor) commitText() {
	if e.textBuffer != "" {
		a := Annotation{
			Type:     ToolText,
			Points:   []image.Point{e.textPos},
			Color:    e.currentColor,
			FontSize: e.fontSize,
			Text:     e.textBuffer,
		}
		e.history.AddAnnotation(a)
	}
	e.textInput = false
	e.textBuffer = ""
	invalidateRect.Call(e.hwnd, 0, 0)
}

// saveAndExit 保存标注结果并退出
func (e *Editor) saveAndExit() {
	if e.textInput {
		e.commitText()
	}

	annotations := e.history.GetAnnotations()
	var finalImg *image.RGBA
	if len(annotations) > 0 {
		finalImg = RenderAnnotations(e.background, annotations)
	} else {
		b := e.background.Bounds()
		finalImg = image.NewRGBA(b)
		draw.Draw(finalImg, b, e.background, b.Min, draw.Src)
	}

	e.result = &EditorResult{
		Image:     finalImg,
		Cancelled: false,
	}
	e.done = true
	postQuitMessage.Call(0)
}

// ============================================================================
// 渲染
// ============================================================================

// onPaint 处理 WM_PAINT 消息
func (e *Editor) onPaint(hdc uintptr) {
	w := e.screenWidth
	h := e.screenHeight

	// 确保 GDI 缓冲已创建
	e.ensureGDIBuffer(hdc, w, h)
	if e.memDC == 0 {
		return
	}

	totalBytes := w * h * 4
	pixels := unsafe.Slice((*byte)(unsafe.Pointer(e.memBits)), totalBytes)

	// 1. 复制预计算的暗化全屏作为遮罩背景（高效 memcpy）
	copy(pixels, e.dimmedPixels)

	// 2. 在选区位置绘制全亮度截图 + 标注
	e.drawAnnotationsToBuffer(pixels, w)

	// 3. 绘制文本输入光标
	if e.textInput {
		e.drawTextCursor(pixels, w)
	}

	// 4. 在离屏缓冲区绘制所有 GDI/GDI+ 元素（避免闪烁）
	e.drawSelectionBorder(e.memDC)
	e.drawResizeHandles(e.memDC)
	e.drawSizeIndicator(e.memDC)
	e.drawToolbar(e.memDC)

	// 5. 一次性 BitBlt 到屏幕（零闪烁）
	bitBlt.Call(hdc, 0, 0, uintptr(w), uintptr(h),
		e.memDC, 0, 0, srccopy)
}

// ensureGDIBuffer 确保 GDI 离屏缓冲区已创建
func (e *Editor) ensureGDIBuffer(hdc uintptr, w, h int) {
	if e.memDC != 0 && e.bufWidth == w && e.bufHeight == h {
		return
	}

	e.cleanupGDICache()

	e.memDC, _, _ = createCompatibleDC.Call(hdc)

	var bi bitmapInfo
	bi.BmiHeader.BiSize = uint32(unsafe.Sizeof(bi.BmiHeader))
	bi.BmiHeader.BiWidth = int32(w)
	bi.BmiHeader.BiHeight = -int32(h) // 自顶向下
	bi.BmiHeader.BiPlanes = 1
	bi.BmiHeader.BiBitCount = 32
	bi.BmiHeader.BiCompression = biRGB

	e.memBitmap, _, _ = createDIBSection.Call(
		e.memDC,
		uintptr(unsafe.Pointer(&bi)),
		dibRGBColors,
		uintptr(unsafe.Pointer(&e.memBits)),
		0, 0,
	)

	if e.memBitmap == 0 {
		deleteDC.Call(e.memDC)
		e.memDC = 0
		return
	}

	selectObject.Call(e.memDC, e.memBitmap)
	e.bufWidth = w
	e.bufHeight = h
}

// cleanupGDICache 清理 GDI 缓存资源
func (e *Editor) cleanupGDICache() {
	if e.memBitmap != 0 {
		deleteObject.Call(e.memBitmap)
		e.memBitmap = 0
	}
	if e.memDC != 0 {
		deleteDC.Call(e.memDC)
		e.memDC = 0
	}
	e.memBits = 0
	e.bufWidth = 0
	e.bufHeight = 0
}

// drawAnnotationsToBuffer 将截图和标注绘制到像素缓冲区
func (e *Editor) drawAnnotationsToBuffer(pixels []byte, stride int) {
	ox := e.imageRect.Min.X
	oy := e.imageRect.Min.Y

	// 拖拽选区时：直接从全屏截图读取当前位置的内容（实时更新，不带标注）
	if e.draggingSelection {
		sel := e.imageRect.Intersect(e.fullscreen.Bounds())
		selW := sel.Dx()
		if selW <= 0 {
			return
		}
		for y := sel.Min.Y; y < sel.Max.Y; y++ {
			if y < 0 || y >= e.screenHeight {
				continue
			}
			srcOff := y*e.fullscreen.Stride + sel.Min.X*4
			dstOff := (y*stride + sel.Min.X) * 4
			// 逐行转换 RGBA → BGRA
			for x := 0; x < selW; x++ {
				si := srcOff + x*4
				di := dstOff + x*4
				pixels[di+0] = e.fullscreen.Pix[si+2]
				pixels[di+1] = e.fullscreen.Pix[si+1]
				pixels[di+2] = e.fullscreen.Pix[si+0]
				pixels[di+3] = 255
			}
		}
		return
	}

	annotations := e.history.GetAnnotations()

	bg := e.background
	b := bg.Bounds()

	var canvas *image.RGBA
	if len(annotations) > 0 {
		canvas = RenderAnnotations(bg, annotations)
	} else {
		canvas = image.NewRGBA(b)
		draw.Draw(canvas, b, bg, b.Min, draw.Src)
	}

	// 渲染临时标注（正在绘制的预览）
	if e.tempAnnotation != nil {
		RenderSingleAnnotation(canvas, e.tempAnnotation)
	}

	// 将渲染结果复制到像素缓冲区的 imageRect 位置
	imgW := b.Dx()
	imgH := b.Dy()

	for y := 0; y < imgH; y++ {
		dy := oy + y
		if dy < 0 || dy >= e.screenHeight {
			continue
		}
		srcOff := y * canvas.Stride
		for x := 0; x < imgW; x++ {
			dx := ox + x
			if dx < 0 || dx >= e.screenWidth {
				continue
			}
			si := srcOff + x*4
			di := (dy*stride + dx) * 4
			// RGBA -> BGRA
			pixels[di+0] = canvas.Pix[si+2]
			pixels[di+1] = canvas.Pix[si+1]
			pixels[di+2] = canvas.Pix[si+0]
			pixels[di+3] = 255
		}
	}
}

// drawTextCursor 绘制文本输入光标
func (e *Editor) drawTextCursor(pixels []byte, stride int) {
	if !e.textInput {
		return
	}

	sx, sy := e.canvasToScreen(e.textPos.X, e.textPos.Y)
	textWidthPx := int(float64(len([]rune(e.textBuffer))) * float64(e.fontSize) * 0.6)
	cursorX := sx + textWidthPx
	cursorH := e.fontSize + 4

	for cy := sy; cy < sy+cursorH && cy < e.screenHeight; cy++ {
		if cy < 0 {
			continue
		}
		if cursorX >= 0 && cursorX < e.screenWidth {
			i := (cy*stride + cursorX) * 4
			pixels[i+0] = 255
			pixels[i+1] = 255
			pixels[i+2] = 255
			pixels[i+3] = 255
		}
	}
}

// ============================================================================
// 选区边框、调整手柄、尺寸指示器（仿微信截图风格）
// ============================================================================

// drawSelectionBorder 绘制选区的绿色边框（1px实线）
func (e *Editor) drawSelectionBorder(hdc uintptr) {
	r := e.imageRect
	pen, _, _ := createPen.Call(psSOLID, 1, 0x0000FF00) // 绿色 (BGR)
	oldPen, _, _ := selectObject.Call(hdc, pen)
	nullBr, _, _ := getStockObject.Call(nullBrush)
	oldBrush, _, _ := selectObject.Call(hdc, nullBr)

	rectangle.Call(hdc,
		uintptr(r.Min.X-1), uintptr(r.Min.Y-1),
		uintptr(r.Max.X+1), uintptr(r.Max.Y+1))

	selectObject.Call(hdc, oldPen)
	selectObject.Call(hdc, oldBrush)
	deleteObject.Call(pen)
}

// drawResizeHandles 绘制选区的8个绿色调整手柄（四角+四边中点）
func (e *Editor) drawResizeHandles(hdc uintptr) {
	r := e.imageRect
	const hs = 3 // 手柄半尺寸 (6x6 方块)

	brush, _, _ := createSolidBrush.Call(0x0000FF00) // 绿色

	// 8个手柄位置: 四角 + 四边中点
	midX := (r.Min.X + r.Max.X) / 2
	midY := (r.Min.Y + r.Max.Y) / 2
	handles := [8]point{
		{int32(r.Min.X), int32(r.Min.Y)}, // 左上
		{int32(midX), int32(r.Min.Y)},     // 上中
		{int32(r.Max.X), int32(r.Min.Y)}, // 右上
		{int32(r.Max.X), int32(midY)},     // 右中
		{int32(r.Max.X), int32(r.Max.Y)}, // 右下
		{int32(midX), int32(r.Max.Y)},     // 下中
		{int32(r.Min.X), int32(r.Max.Y)}, // 左下
		{int32(r.Min.X), int32(midY)},     // 左中
	}

	for _, p := range handles {
		rc := rect{
			Left:   p.X - hs,
			Top:    p.Y - hs,
			Right:  p.X + hs,
			Bottom: p.Y + hs,
		}
		fillRect.Call(hdc, uintptr(unsafe.Pointer(&rc)), brush)
	}

	deleteObject.Call(brush)
}

// drawSizeIndicator 绘制选区尺寸指示器（如 "968 x 511"）
func (e *Editor) drawSizeIndicator(hdc uintptr) {
	selW := e.imageRect.Dx()
	selH := e.imageRect.Dy()
	text := strconv.Itoa(selW) + " x " + strconv.Itoa(selH)

	// 指示器位置: 选区左上方
	textH := 20
	charW := 8
	textW := len(text)*charW + 12
	x := e.imageRect.Min.X
	y := e.imageRect.Min.Y - textH - 4

	// 如果上方空间不足，放到选区内部顶部
	if y < 0 {
		y = e.imageRect.Min.Y + 4
	}

	// 绘制暗色背景
	bgBrush, _, _ := createSolidBrush.Call(0x00333333) // 深灰
	bgRect := rect{
		Left:   int32(x),
		Top:    int32(y),
		Right:  int32(x + textW),
		Bottom: int32(y + textH),
	}
	fillRect.Call(hdc, uintptr(unsafe.Pointer(&bgRect)), bgBrush)
	deleteObject.Call(bgBrush)

	// 绘制白色文本
	setBkMode.Call(hdc, transparent)
	setTextColor.Call(hdc, 0x00FFFFFF)

	hFont, _, _ := createFontW.Call(
		uintptr(14), 0, 0, 0,
		400, // 正常粗细
		0, 0, 0, 1, 0, 0, 0, 0,
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("Consolas"))),
	)
	oldFont, _, _ := selectObject.Call(hdc, hFont)

	utf16Text := utf16.Encode([]rune(text))
	textOutW.Call(hdc, uintptr(x+6), uintptr(y+3),
		uintptr(unsafe.Pointer(&utf16Text[0])),
		uintptr(len(utf16Text)))

	selectObject.Call(hdc, oldFont)
	deleteObject.Call(hFont)
}

// ============================================================================
// 浮动工具栏绘制（使用 GDI 直接绘制到窗口 DC）
// ============================================================================

// drawToolbar 绘制浮动工具栏
func (e *Editor) drawToolbar(hdc uintptr) {
	setBkMode.Call(hdc, transparent)

	// 绘制二级面板（在主工具栏下面，所以先绘制以免被遮挡）
	if e.showSubToolbar {
		e.drawSubToolbar(hdc)
	}

	// 绘制主工具栏
	e.drawMainToolbar(hdc)
}

// drawToolbarBackground 绘制工具栏背景（GDI+ 抗锯齿圆角矩形 + 阴影 + 边框）
func (e *Editor) drawToolbarBackground(hdc uintptr, r image.Rectangle) {
	g := gdipNewGraphics(hdc)
	defer gdipDeleteGraphics.Call(g)

	w := r.Dx()
	h := r.Dy()
	rad := toolbarRadius

	// 阴影（向右下偏移2px）
	shadowBrush := gdipNewBrush(colorShadow)
	gdipFillRoundRect(g, shadowBrush, r.Min.X+2, r.Min.Y+2, w, h, rad)
	gdipDeleteBrush.Call(shadowBrush)

	// 主背景
	bgBrush := gdipNewBrush(colorToolbarBg)
	gdipFillRoundRect(g, bgBrush, r.Min.X, r.Min.Y, w, h, rad)
	gdipDeleteBrush.Call(bgBrush)

	// 1px 边框
	borderPen := gdipNewPen(colorToolbarBorder, 1)
	gdipDrawRoundRect(g, borderPen, r.Min.X, r.Min.Y, w, h, rad)
	gdipDeletePen.Call(borderPen)
}

// drawMainToolbar 绘制主工具栏
func (e *Editor) drawMainToolbar(hdc uintptr) {
	// 绘制背景
	e.drawToolbarBackground(hdc, e.mainToolbarRect)

	// 绘制按钮
	buttons := e.getMainToolbarButtons()
	for i, btn := range buttons {
		isSelected := btn.kind == "tool" && btn.toolType == e.currentTool
		isHovered := i == e.hoverBtnIndex

		if isSelected {
			e.drawSelectedBg(hdc, btn.x, btn.y, btn.w, btn.h)
		} else if isHovered {
			e.drawBtnHoverBg(hdc, btn.x, btn.y, btn.w, btn.h)
		}

		switch btn.kind {
		case "tool":
			e.drawToolIcon(hdc, btn, isSelected || isHovered)
		case "action":
			e.drawActionButton(hdc, btn, isHovered)
		}
	}

	// 绘制分隔符
	e.drawMainToolbarSeparators(hdc)
}

// drawSubToolbar 绘制二级面板
func (e *Editor) drawSubToolbar(hdc uintptr) {
	// 绘制背景
	e.drawToolbarBackground(hdc, e.subToolbarRect)

	// 绘制按钮
	buttons := e.getSubToolbarButtons()
	for i, btn := range buttons {
		isHovered := i == e.hoverSubIndex

		// 悬停效果（颜色和线宽按钮不需要selected bg，各自有自己的选中样式）
		if isHovered {
			e.drawBtnHoverBg(hdc, btn.x, btn.y, btn.w, btn.h)
		}

		switch btn.kind {
		case "linewidth":
			e.drawLineWidthDot(hdc, btn)
		case "color":
			e.drawColorBlock(hdc, btn)
		}
	}

	// 绘制分隔符
	e.drawSubToolbarSeparator(hdc)
}

// drawBtnHoverBg 绘制按钮悬停背景（GDI+ 抗锯齿圆角矩形）
func (e *Editor) drawBtnHoverBg(hdc uintptr, x, y, w, h int) {
	g := gdipNewGraphics(hdc)
	defer gdipDeleteGraphics.Call(g)
	brush := gdipNewBrush(colorToolbarHover)
	gdipFillRoundRect(g, brush, x+2, y+2, w-4, h-4, 8)
	gdipDeleteBrush.Call(brush)
}

// drawSelectedBg 绘制选中工具的蓝色调背景（GDI+ 抗锯齿圆角矩形）
func (e *Editor) drawSelectedBg(hdc uintptr, x, y, w, h int) {
	g := gdipNewGraphics(hdc)
	defer gdipDeleteGraphics.Call(g)
	brush := gdipNewBrush(colorBtnSelectedBg)
	gdipFillRoundRect(g, brush, x+2, y+2, w-4, h-4, 8)
	gdipDeleteBrush.Call(brush)
}

// drawMainToolbarSeparators 绘制主工具栏分隔符
func (e *Editor) drawMainToolbarSeparators(hdc uintptr) {
	buttons := e.getMainToolbarButtons()
	if len(buttons) == 0 {
		return
	}

	pen, _, _ := createPen.Call(psSOLID, 1, colorSeparator)
	oldPen, _, _ := selectObject.Call(hdc, pen)

	sepY1 := e.mainToolbarRect.Min.Y + 12
	sepY2 := e.mainToolbarRect.Max.Y - 12

	// 第一个分隔符: 工具按钮和撤销/重做之间 (第7个按钮之后)
	numTools := len(mainToolBtnOrder)
	if numTools < len(buttons) {
		sepX := buttons[numTools-1].x + buttons[numTools-1].w + (toolbarSepWidth / 2)
		moveToEx.Call(hdc, uintptr(sepX), uintptr(sepY1), 0)
		lineTo.Call(hdc, uintptr(sepX), uintptr(sepY2))
	}

	// 第二个分隔符: 重做和保存之间 (第7+2=9个按钮之后)
	sepIdx := numTools + 2
	if sepIdx < len(buttons) {
		sepX := buttons[sepIdx-1].x + buttons[sepIdx-1].w + (toolbarSepWidth / 2)
		moveToEx.Call(hdc, uintptr(sepX), uintptr(sepY1), 0)
		lineTo.Call(hdc, uintptr(sepX), uintptr(sepY2))
	}

	selectObject.Call(hdc, oldPen)
	deleteObject.Call(pen)
}

// drawSubToolbarSeparator 绘制二级面板分隔符
func (e *Editor) drawSubToolbarSeparator(hdc uintptr) {
	buttons := e.getSubToolbarButtons()
	if len(buttons) == 0 {
		return
	}

	pen, _, _ := createPen.Call(psSOLID, 1, colorSeparator)
	oldPen, _, _ := selectObject.Call(hdc, pen)

	sepY1 := e.subToolbarRect.Min.Y + 10
	sepY2 := e.subToolbarRect.Max.Y - 10

	// 分隔符在线宽按钮和颜色按钮之间
	numLW := len(subToolbarLineWidths)
	if numLW < len(buttons) {
		sepX := buttons[numLW-1].x + buttons[numLW-1].w + (toolbarSepWidth / 2)
		moveToEx.Call(hdc, uintptr(sepX), uintptr(sepY1), 0)
		lineTo.Call(hdc, uintptr(sepX), uintptr(sepY2))
	}

	selectObject.Call(hdc, oldPen)
	deleteObject.Call(pen)
}

// drawToolIcon 绘制工具按钮图标（GDI+ 抗锯齿绘制）
func (e *Editor) drawToolIcon(hdc uintptr, btn toolbarButton, highlighted bool) {
	cx := btn.x + btn.w/2
	cy := btn.y + btn.h/2

	// 根据状态选择图标颜色
	var iconColor uintptr
	if btn.toolType == e.currentTool {
		iconColor = colorIconSelected
	} else if highlighted {
		iconColor = colorIconHover
	} else {
		iconColor = colorIconNormal
	}

	g := gdipNewGraphics(hdc)
	defer gdipDeleteGraphics.Call(g)

	pen := gdipNewPen(iconColor, 2.5)
	defer gdipDeletePen.Call(pen)

	switch btn.toolType {
	case ToolRect:
		// 圆角矩形（圆角半径 8，真正圆润）
		gdipDrawRoundRect(g, pen, cx-12, cy-9, 24, 18, 6)

	case ToolEllipse:
		gdipDrawEllipseI.Call(g, pen, uintptr(cx-11), uintptr(cy-11), 22, 22)

	case ToolArrow:
		// 箭头杆
		gdipDrawLineI.Call(g, pen, uintptr(cx-10), uintptr(cy+10), uintptr(cx+10), uintptr(cy-10))
		// V 形箭头头部
		gdipDrawLineI.Call(g, pen, uintptr(cx+2), uintptr(cy-9), uintptr(cx+10), uintptr(cy-10))
		gdipDrawLineI.Call(g, pen, uintptr(cx+10), uintptr(cy-10), uintptr(cx+9), uintptr(cy-2))

	case ToolLine:
		gdipDrawLineI.Call(g, pen, uintptr(cx-10), uintptr(cy+10), uintptr(cx+10), uintptr(cy-10))

	case ToolFreehand:
		// GDI+ 贝塞尔曲线（真正平滑）
		gdipDrawBezierI.Call(g, pen,
			uintptr(cx-12), uintptr(cy+4),
			uintptr(cx-5), uintptr(cy-10),
			uintptr(cx+5), uintptr(cy+10),
			uintptr(cx+12), uintptr(cy-4))

	case ToolText:
		// 绘制 "A" 字母（GDI TextOut 自带 ClearType 抗锯齿）
		gdipDeleteGraphics.Call(g)
		gdipDeletePen.Call(pen)
		e.drawGDIText(hdc, "A", cx-9, cy-12, 18, 24, iconColor)
		return

	case ToolMosaic:
		// 2x2 圆角网格
		s := 10
		gdipDrawRoundRect(g, pen, cx-s, cy-s, s*2, s*2, 4)
		gdipDrawLineI.Call(g, pen, uintptr(cx), uintptr(cy-s+2), uintptr(cx), uintptr(cy+s-2))
		gdipDrawLineI.Call(g, pen, uintptr(cx-s+2), uintptr(cy), uintptr(cx+s-2), uintptr(cy))
	}
}

// drawActionButton 绘制操作按钮（GDI+ 抗锯齿）
func (e *Editor) drawActionButton(hdc uintptr, btn toolbarButton, hovered bool) {
	cx := btn.x + btn.w/2
	cy := btn.y + btn.h/2

	var penColor uintptr
	switch btn.action {
	case "save":
		penColor = colorSaveGreen
	case "cancel":
		penColor = colorCancelRed
	default:
		if hovered {
			penColor = colorIconHover
		} else {
			penColor = colorIconNormal
		}
	}

	g := gdipNewGraphics(hdc)
	defer gdipDeleteGraphics.Call(g)
	pen := gdipNewPen(penColor, 2.5)
	defer gdipDeletePen.Call(pen)

	switch btn.action {
	case "undo":
		// 左箭头
		gdipDrawLineI.Call(g, pen, uintptr(cx+9), uintptr(cy), uintptr(cx-6), uintptr(cy))
		gdipDrawLineI.Call(g, pen, uintptr(cx-1), uintptr(cy-6), uintptr(cx-6), uintptr(cy))
		gdipDrawLineI.Call(g, pen, uintptr(cx-6), uintptr(cy), uintptr(cx-1), uintptr(cy+6))

	case "redo":
		// 右箭头
		gdipDrawLineI.Call(g, pen, uintptr(cx-9), uintptr(cy), uintptr(cx+6), uintptr(cy))
		gdipDrawLineI.Call(g, pen, uintptr(cx+1), uintptr(cy-6), uintptr(cx+6), uintptr(cy))
		gdipDrawLineI.Call(g, pen, uintptr(cx+6), uintptr(cy), uintptr(cx+1), uintptr(cy+6))

	case "save":
		// 对勾 ✓
		gdipDrawLineI.Call(g, pen, uintptr(cx-9), uintptr(cy), uintptr(cx-3), uintptr(cy+7))
		gdipDrawLineI.Call(g, pen, uintptr(cx-3), uintptr(cy+7), uintptr(cx+10), uintptr(cy-7))

	case "cancel":
		// X 号
		gdipDrawLineI.Call(g, pen, uintptr(cx-7), uintptr(cy-7), uintptr(cx+7), uintptr(cy+7))
		gdipDrawLineI.Call(g, pen, uintptr(cx+7), uintptr(cy-7), uintptr(cx-7), uintptr(cy+7))
	}
}

// drawLineWidthDot 绘制线宽选择圆点（GDI+ 抗锯齿）
func (e *Editor) drawLineWidthDot(hdc uintptr, btn toolbarButton) {
	dotRadius := btn.lineWidth + 2
	if dotRadius > btn.w/2-3 {
		dotRadius = btn.w/2 - 3
	}

	cx := btn.x + btn.w/2
	cy := btn.y + btn.h/2

	var dotColor uintptr
	if btn.lineWidth == e.lineWidth {
		dotColor = colorAccent
	} else {
		dotColor = 0x00AAAAAA
	}

	g := gdipNewGraphics(hdc)
	defer gdipDeleteGraphics.Call(g)
	brush := gdipNewBrush(dotColor)
	defer gdipDeleteBrush.Call(brush)

	gdipFillEllipseI.Call(g, brush,
		uintptr(cx-dotRadius), uintptr(cy-dotRadius),
		uintptr(dotRadius*2), uintptr(dotRadius*2))
}

// drawColorBlock 绘制颜色圆圈（GDI+ 抗锯齿，选中时带白色选中环）
func (e *Editor) drawColorBlock(hdc uintptr, btn toolbarButton) {
	if btn.colorIndex < 0 || btn.colorIndex >= len(DefaultColors) {
		return
	}

	c := DefaultColors[btn.colorIndex]
	colorRef := uintptr(uint32(c.R)) | (uintptr(uint32(c.G)) << 8) | (uintptr(uint32(c.B)) << 16)

	cx := btn.x + btn.w/2
	cy := btn.y + btn.h/2
	r := btn.w/2 - 2

	g := gdipNewGraphics(hdc)
	defer gdipDeleteGraphics.Call(g)

	// 绘制填充圆
	brush := gdipNewBrush(colorRef)
	gdipFillEllipseI.Call(g, brush, uintptr(cx-r), uintptr(cy-r), uintptr(r*2), uintptr(r*2))
	gdipDeleteBrush.Call(brush)

	// 选中时绘制白色选中环
	if c == e.currentColor {
		ringR := r + 3
		ringPen := gdipNewPen(0x00FFFFFF, 2)
		gdipDrawEllipseI.Call(g, ringPen, uintptr(cx-ringR), uintptr(cy-ringR), uintptr(ringR*2), uintptr(ringR*2))
		gdipDeletePen.Call(ringPen)
	}
}

// drawGDIText 使用 GDI 绘制文本
func (e *Editor) drawGDIText(hdc uintptr, text string, x, y, w, h int, colorRef uintptr) {
	hFont, _, _ := createFontW.Call(
		uintptr(h+2),
		0,
		0, 0,
		700,           // 粗体
		0, 0, 0,
		1,             // DEFAULT_CHARSET
		0, 0,
		5,             // CLEARTYPE_QUALITY
		0,
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("Segoe UI"))),
	)
	oldFont, _, _ := selectObject.Call(hdc, hFont)
	setTextColor.Call(hdc, colorRef)

	utf16Text := utf16.Encode([]rune(text))
	textOutW.Call(hdc, uintptr(x+w/5), uintptr(y),
		uintptr(unsafe.Pointer(&utf16Text[0])),
		uintptr(len(utf16Text)))

	selectObject.Call(hdc, oldFont)
	deleteObject.Call(hFont)
}

// ============================================================================
// 辅助函数
// ============================================================================

// editorDrainQuitMessages 清理消息队列中残留的 WM_QUIT 消息
func editorDrainQuitMessages() {
	var m msg
	for {
		ret, _, _ := peekMessageW.Call(
			uintptr(unsafe.Pointer(&m)),
			0,
			wmQuit,
			wmQuit,
			pmRemove,
		)
		if ret == 0 {
			break
		}
	}
}
