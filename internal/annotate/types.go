package annotate

import (
	"image"
	"image/color"
)

// ToolType 标注工具类型
type ToolType int

const (
	ToolRect     ToolType = iota // 矩形
	ToolArrow                   // 箭头
	ToolLine                    // 直线
	ToolText                    // 文本
	ToolFreehand                // 自由画笔
	ToolMosaic                  // 马赛克/模糊
	ToolEllipse                 // 椭圆
	ToolCount                   // 工具总数（用于遍历）
)

// ToolName 工具显示名称
var ToolName = map[ToolType]string{
	ToolRect:     "矩形",
	ToolArrow:    "箭头",
	ToolLine:     "直线",
	ToolText:     "文本",
	ToolFreehand: "画笔",
	ToolMosaic:   "马赛克",
	ToolEllipse:  "椭圆",
}

// Annotation 单个标注
type Annotation struct {
	Type      ToolType      // 标注类型
	Points    []image.Point // 路径点（矩形/箭头/直线用前两个点，画笔用所有点）
	Color     color.RGBA    // 颜色
	LineWidth int           // 线宽
	Text      string        // 文本内容（仅 ToolText 使用）
	FontSize  int           // 字号（仅 ToolText 使用）
	Filled    bool          // 是否填充（矩形/椭圆）
	MosaicPx  int           // 马赛克像素块大小
}

// Bounds 获取标注的边界矩形
func (a *Annotation) Bounds() image.Rectangle {
	if len(a.Points) == 0 {
		return image.Rectangle{}
	}

	minX, minY := a.Points[0].X, a.Points[0].Y
	maxX, maxY := minX, minY

	for _, p := range a.Points[1:] {
		if p.X < minX {
			minX = p.X
		}
		if p.Y < minY {
			minY = p.Y
		}
		if p.X > maxX {
			maxX = p.X
		}
		if p.Y > maxY {
			maxY = p.Y
		}
	}

	// 扩展线宽
	pad := a.LineWidth/2 + 1
	return image.Rect(minX-pad, minY-pad, maxX+pad, maxY+pad)
}

// DefaultColors 预设颜色面板
var DefaultColors = []color.RGBA{
	{255, 0, 0, 255},     // 红色
	{0, 180, 0, 255},     // 绿色
	{0, 120, 255, 255},   // 蓝色
	{255, 200, 0, 255},   // 黄色
	{255, 128, 0, 255},   // 橙色
	{180, 0, 255, 255},   // 紫色
	{255, 255, 255, 255}, // 白色
	{0, 0, 0, 255},       // 黑色
}

// DefaultLineWidths 预设线宽
var DefaultLineWidths = []int{2, 3, 5, 8}

// DefaultFontSizes 预设字号
var DefaultFontSizes = []int{16, 20, 28, 36}

// EditorResult 编辑器返回结果
type EditorResult struct {
	Image     *image.RGBA  // 最终带标注的图片
	Cancelled bool         // 用户是否取消
}
