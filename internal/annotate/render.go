package annotate

import (
	"image"
	"image/color"
	"image/draw"
	"math"
)

// RenderAnnotations 将所有标注渲染到基础图片的副本上
func RenderAnnotations(base *image.RGBA, annotations []Annotation) *image.RGBA {
	// 创建副本，避免修改原图
	bounds := base.Bounds()
	result := image.NewRGBA(bounds)
	draw.Draw(result, bounds, base, bounds.Min, draw.Src)

	for i := range annotations {
		RenderSingleAnnotation(result, &annotations[i])
	}
	return result
}

// RenderSingleAnnotation 将单个标注渲染到图片上（用于实时预览）
func RenderSingleAnnotation(img *image.RGBA, a *Annotation) {
	switch a.Type {
	case ToolRect:
		renderRect(img, a)
	case ToolArrow:
		renderArrow(img, a)
	case ToolLine:
		renderLine(img, a)
	case ToolText:
		renderText(img, a)
	case ToolFreehand:
		renderFreehand(img, a)
	case ToolEllipse:
		renderEllipse(img, a)
	case ToolMosaic:
		renderMosaic(img, a)
	}
}

// ---------- 矩形 ----------

func renderRect(img *image.RGBA, a *Annotation) {
	if len(a.Points) < 2 {
		return
	}
	r := canonicalRect(a.Points[0], a.Points[1])

	if a.Filled {
		// 半透明填充
		fillColor := a.Color
		fillColor.A = 80
		for y := r.Min.Y; y < r.Max.Y; y++ {
			for x := r.Min.X; x < r.Max.X; x++ {
				setPixelBlend(img, x, y, fillColor)
			}
		}
	}

	// 描边
	drawRectStroke(img, r, a.Color, a.LineWidth)
}

// drawRectStroke 绘制矩形描边
func drawRectStroke(img *image.RGBA, r image.Rectangle, c color.RGBA, width int) {
	// 上边
	drawThickLine(img, r.Min.X, r.Min.Y, r.Max.X-1, r.Min.Y, c, width)
	// 下边
	drawThickLine(img, r.Min.X, r.Max.Y-1, r.Max.X-1, r.Max.Y-1, c, width)
	// 左边
	drawThickLine(img, r.Min.X, r.Min.Y, r.Min.X, r.Max.Y-1, c, width)
	// 右边
	drawThickLine(img, r.Max.X-1, r.Min.Y, r.Max.X-1, r.Max.Y-1, c, width)
}

// ---------- 箭头 ----------

func renderArrow(img *image.RGBA, a *Annotation) {
	if len(a.Points) < 2 {
		return
	}
	p0 := a.Points[0]
	p1 := a.Points[1]

	// 绘制主线段
	drawThickLine(img, p0.X, p0.Y, p1.X, p1.Y, a.Color, a.LineWidth)

	// 计算箭头三角形
	dx := float64(p1.X - p0.X)
	dy := float64(p1.Y - p0.Y)
	length := math.Hypot(dx, dy)
	if length < 1 {
		return
	}

	// 箭头大小与线宽成比例
	arrowLen := float64(a.LineWidth) * 5
	if arrowLen < 12 {
		arrowLen = 12
	}
	arrowWidth := arrowLen * 0.5

	// 单位方向向量
	ux := dx / length
	uy := dy / length

	// 箭头根部位置
	baseX := float64(p1.X) - ux*arrowLen
	baseY := float64(p1.Y) - uy*arrowLen

	// 垂直方向
	nx := -uy
	ny := ux

	// 箭头三个顶点
	tip := p1
	left := image.Point{
		X: int(math.Round(baseX + nx*arrowWidth)),
		Y: int(math.Round(baseY + ny*arrowWidth)),
	}
	right := image.Point{
		X: int(math.Round(baseX - nx*arrowWidth)),
		Y: int(math.Round(baseY - ny*arrowWidth)),
	}

	drawFilledTriangle(img, tip, left, right, a.Color)
}

// ---------- 直线 ----------

func renderLine(img *image.RGBA, a *Annotation) {
	if len(a.Points) < 2 {
		return
	}
	drawThickLine(img, a.Points[0].X, a.Points[0].Y, a.Points[1].X, a.Points[1].Y, a.Color, a.LineWidth)
}

// ---------- 文本 ----------

func renderText(img *image.RGBA, a *Annotation) {
	if len(a.Points) < 1 || a.Text == "" {
		return
	}

	fontSize := a.FontSize
	if fontSize <= 0 {
		fontSize = 16
	}

	// 简单位图字体：每个字符的宽高
	charW := fontSize * 3 / 5 // 字符宽度约为字号的 0.6
	charH := fontSize

	x0 := a.Points[0].X
	y0 := a.Points[0].Y

	// 计算文本总宽度和高度（支持多行）
	lines := splitLines(a.Text)
	maxWidth := 0
	for _, line := range lines {
		w := len(line) * charW
		if w > maxWidth {
			maxWidth = w
		}
	}
	totalH := len(lines) * charH

	// 绘制文本背景（半透明黑色）
	padding := 4
	bgColor := color.RGBA{0, 0, 0, 160}
	for y := y0 - padding; y < y0+totalH+padding; y++ {
		for x := x0 - padding; x < x0+maxWidth+padding; x++ {
			setPixelBlend(img, x, y, bgColor)
		}
	}

	// 绘制文本
	textColor := a.Color
	if textColor.A == 0 {
		textColor = color.RGBA{255, 255, 255, 255}
	}

	for lineIdx, line := range lines {
		cy := y0 + lineIdx*charH
		cx := x0
		for _, ch := range line {
			if ch > 127 {
				// 非 ASCII 字符：绘制占位方块
				drawCharBlock(img, cx, cy, charW, charH, textColor)
			} else {
				drawBitmapChar(img, cx, cy, charW, charH, byte(ch), textColor)
			}
			cx += charW
		}
	}
}

// splitLines 按换行符拆分字符串
func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	lines = append(lines, s[start:])
	return lines
}

// drawCharBlock 绘制占位方块（用于非 ASCII 字符）
func drawCharBlock(img *image.RGBA, x, y, w, h int, c color.RGBA) {
	margin := 1
	for dy := margin; dy < h-margin; dy++ {
		for dx := margin; dx < w-margin; dx++ {
			setPixelBlend(img, x+dx, y+dy, c)
		}
	}
}

// drawBitmapChar 使用简单的 5x7 位图字体绘制 ASCII 字符
func drawBitmapChar(img *image.RGBA, x, y, w, h int, ch byte, c color.RGBA) {
	glyph := getGlyph(ch)
	if glyph == nil {
		// 无法识别的字符，绘制方块
		drawCharBlock(img, x, y, w, h, c)
		return
	}

	// 将 5x7 栅格缩放到目标尺寸
	scaleX := float64(w) / 5.0
	scaleY := float64(h) / 7.0

	for row := 0; row < 7; row++ {
		for col := 0; col < 5; col++ {
			if glyph[row]&(1<<(4-col)) != 0 {
				// 填充缩放后的像素区域
				px0 := int(float64(col) * scaleX)
				px1 := int(float64(col+1) * scaleX)
				py0 := int(float64(row) * scaleY)
				py1 := int(float64(row+1) * scaleY)
				for py := py0; py < py1; py++ {
					for px := px0; px < px1; px++ {
						setPixelBlend(img, x+px, y+py, c)
					}
				}
			}
		}
	}
}

// getGlyph 获取 ASCII 字符的 5x7 位图数据
// 每行用 uint8 表示，高 5 位有效（bit4..bit0 对应从左到右的像素）
func getGlyph(ch byte) []uint8 {
	if ch < 32 || ch > 126 {
		return nil
	}
	idx := int(ch) - 32
	if idx >= len(font5x7) {
		return nil
	}
	return font5x7[idx][:]
}

// font5x7 是简化的 5x7 ASCII 位图字体（空格 0x20 到 ~ 0x7E）
// 每个字符 7 行，每行高 5 位有效
var font5x7 = [95][7]uint8{
	// 空格 (0x20)
	{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
	// ! (0x21)
	{0x04, 0x04, 0x04, 0x04, 0x04, 0x00, 0x04},
	// " (0x22)
	{0x0A, 0x0A, 0x00, 0x00, 0x00, 0x00, 0x00},
	// # (0x23)
	{0x0A, 0x1F, 0x0A, 0x0A, 0x1F, 0x0A, 0x00},
	// $ (0x24)
	{0x04, 0x0F, 0x14, 0x0E, 0x05, 0x1E, 0x04},
	// % (0x25)
	{0x18, 0x19, 0x02, 0x04, 0x08, 0x13, 0x03},
	// & (0x26)
	{0x08, 0x14, 0x14, 0x08, 0x15, 0x12, 0x0D},
	// ' (0x27)
	{0x04, 0x04, 0x00, 0x00, 0x00, 0x00, 0x00},
	// ( (0x28)
	{0x02, 0x04, 0x08, 0x08, 0x08, 0x04, 0x02},
	// ) (0x29)
	{0x08, 0x04, 0x02, 0x02, 0x02, 0x04, 0x08},
	// * (0x2A)
	{0x00, 0x0A, 0x04, 0x1F, 0x04, 0x0A, 0x00},
	// + (0x2B)
	{0x00, 0x04, 0x04, 0x1F, 0x04, 0x04, 0x00},
	// , (0x2C)
	{0x00, 0x00, 0x00, 0x00, 0x00, 0x04, 0x08},
	// - (0x2D)
	{0x00, 0x00, 0x00, 0x1F, 0x00, 0x00, 0x00},
	// . (0x2E)
	{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x04},
	// / (0x2F)
	{0x01, 0x01, 0x02, 0x04, 0x08, 0x10, 0x10},
	// 0 (0x30)
	{0x0E, 0x11, 0x13, 0x15, 0x19, 0x11, 0x0E},
	// 1 (0x31)
	{0x04, 0x0C, 0x04, 0x04, 0x04, 0x04, 0x0E},
	// 2 (0x32)
	{0x0E, 0x11, 0x01, 0x06, 0x08, 0x10, 0x1F},
	// 3 (0x33)
	{0x0E, 0x11, 0x01, 0x06, 0x01, 0x11, 0x0E},
	// 4 (0x34)
	{0x02, 0x06, 0x0A, 0x12, 0x1F, 0x02, 0x02},
	// 5 (0x35)
	{0x1F, 0x10, 0x1E, 0x01, 0x01, 0x11, 0x0E},
	// 6 (0x36)
	{0x06, 0x08, 0x10, 0x1E, 0x11, 0x11, 0x0E},
	// 7 (0x37)
	{0x1F, 0x01, 0x02, 0x04, 0x08, 0x08, 0x08},
	// 8 (0x38)
	{0x0E, 0x11, 0x11, 0x0E, 0x11, 0x11, 0x0E},
	// 9 (0x39)
	{0x0E, 0x11, 0x11, 0x0F, 0x01, 0x02, 0x0C},
	// : (0x3A)
	{0x00, 0x00, 0x04, 0x00, 0x00, 0x04, 0x00},
	// ; (0x3B)
	{0x00, 0x00, 0x04, 0x00, 0x00, 0x04, 0x08},
	// < (0x3C)
	{0x02, 0x04, 0x08, 0x10, 0x08, 0x04, 0x02},
	// = (0x3D)
	{0x00, 0x00, 0x1F, 0x00, 0x1F, 0x00, 0x00},
	// > (0x3E)
	{0x08, 0x04, 0x02, 0x01, 0x02, 0x04, 0x08},
	// ? (0x3F)
	{0x0E, 0x11, 0x01, 0x02, 0x04, 0x00, 0x04},
	// @ (0x40)
	{0x0E, 0x11, 0x17, 0x15, 0x17, 0x10, 0x0E},
	// A (0x41)
	{0x0E, 0x11, 0x11, 0x1F, 0x11, 0x11, 0x11},
	// B (0x42)
	{0x1E, 0x11, 0x11, 0x1E, 0x11, 0x11, 0x1E},
	// C (0x43)
	{0x0E, 0x11, 0x10, 0x10, 0x10, 0x11, 0x0E},
	// D (0x44)
	{0x1C, 0x12, 0x11, 0x11, 0x11, 0x12, 0x1C},
	// E (0x45)
	{0x1F, 0x10, 0x10, 0x1E, 0x10, 0x10, 0x1F},
	// F (0x46)
	{0x1F, 0x10, 0x10, 0x1E, 0x10, 0x10, 0x10},
	// G (0x47)
	{0x0E, 0x11, 0x10, 0x17, 0x11, 0x11, 0x0F},
	// H (0x48)
	{0x11, 0x11, 0x11, 0x1F, 0x11, 0x11, 0x11},
	// I (0x49)
	{0x0E, 0x04, 0x04, 0x04, 0x04, 0x04, 0x0E},
	// J (0x4A)
	{0x07, 0x02, 0x02, 0x02, 0x02, 0x12, 0x0C},
	// K (0x4B)
	{0x11, 0x12, 0x14, 0x18, 0x14, 0x12, 0x11},
	// L (0x4C)
	{0x10, 0x10, 0x10, 0x10, 0x10, 0x10, 0x1F},
	// M (0x4D)
	{0x11, 0x1B, 0x15, 0x15, 0x11, 0x11, 0x11},
	// N (0x4E)
	{0x11, 0x19, 0x15, 0x13, 0x11, 0x11, 0x11},
	// O (0x4F)
	{0x0E, 0x11, 0x11, 0x11, 0x11, 0x11, 0x0E},
	// P (0x50)
	{0x1E, 0x11, 0x11, 0x1E, 0x10, 0x10, 0x10},
	// Q (0x51)
	{0x0E, 0x11, 0x11, 0x11, 0x15, 0x12, 0x0D},
	// R (0x52)
	{0x1E, 0x11, 0x11, 0x1E, 0x14, 0x12, 0x11},
	// S (0x53)
	{0x0E, 0x11, 0x10, 0x0E, 0x01, 0x11, 0x0E},
	// T (0x54)
	{0x1F, 0x04, 0x04, 0x04, 0x04, 0x04, 0x04},
	// U (0x55)
	{0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x0E},
	// V (0x56)
	{0x11, 0x11, 0x11, 0x11, 0x0A, 0x0A, 0x04},
	// W (0x57)
	{0x11, 0x11, 0x11, 0x15, 0x15, 0x1B, 0x11},
	// X (0x58)
	{0x11, 0x11, 0x0A, 0x04, 0x0A, 0x11, 0x11},
	// Y (0x59)
	{0x11, 0x11, 0x0A, 0x04, 0x04, 0x04, 0x04},
	// Z (0x5A)
	{0x1F, 0x01, 0x02, 0x04, 0x08, 0x10, 0x1F},
	// [ (0x5B)
	{0x0E, 0x08, 0x08, 0x08, 0x08, 0x08, 0x0E},
	// \ (0x5C)
	{0x10, 0x10, 0x08, 0x04, 0x02, 0x01, 0x01},
	// ] (0x5D)
	{0x0E, 0x02, 0x02, 0x02, 0x02, 0x02, 0x0E},
	// ^ (0x5E)
	{0x04, 0x0A, 0x11, 0x00, 0x00, 0x00, 0x00},
	// _ (0x5F)
	{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x1F},
	// ` (0x60)
	{0x08, 0x04, 0x00, 0x00, 0x00, 0x00, 0x00},
	// a (0x61)
	{0x00, 0x00, 0x0E, 0x01, 0x0F, 0x11, 0x0F},
	// b (0x62)
	{0x10, 0x10, 0x1E, 0x11, 0x11, 0x11, 0x1E},
	// c (0x63)
	{0x00, 0x00, 0x0E, 0x11, 0x10, 0x11, 0x0E},
	// d (0x64)
	{0x01, 0x01, 0x0F, 0x11, 0x11, 0x11, 0x0F},
	// e (0x65)
	{0x00, 0x00, 0x0E, 0x11, 0x1F, 0x10, 0x0E},
	// f (0x66)
	{0x06, 0x08, 0x1E, 0x08, 0x08, 0x08, 0x08},
	// g (0x67)
	{0x00, 0x00, 0x0F, 0x11, 0x0F, 0x01, 0x0E},
	// h (0x68)
	{0x10, 0x10, 0x1E, 0x11, 0x11, 0x11, 0x11},
	// i (0x69)
	{0x04, 0x00, 0x0C, 0x04, 0x04, 0x04, 0x0E},
	// j (0x6A)
	{0x02, 0x00, 0x06, 0x02, 0x02, 0x12, 0x0C},
	// k (0x6B)
	{0x10, 0x10, 0x12, 0x14, 0x18, 0x14, 0x12},
	// l (0x6C)
	{0x0C, 0x04, 0x04, 0x04, 0x04, 0x04, 0x0E},
	// m (0x6D)
	{0x00, 0x00, 0x1A, 0x15, 0x15, 0x15, 0x15},
	// n (0x6E)
	{0x00, 0x00, 0x1E, 0x11, 0x11, 0x11, 0x11},
	// o (0x6F)
	{0x00, 0x00, 0x0E, 0x11, 0x11, 0x11, 0x0E},
	// p (0x70)
	{0x00, 0x00, 0x1E, 0x11, 0x1E, 0x10, 0x10},
	// q (0x71)
	{0x00, 0x00, 0x0F, 0x11, 0x0F, 0x01, 0x01},
	// r (0x72)
	{0x00, 0x00, 0x16, 0x19, 0x10, 0x10, 0x10},
	// s (0x73)
	{0x00, 0x00, 0x0F, 0x10, 0x0E, 0x01, 0x1E},
	// t (0x74)
	{0x08, 0x08, 0x1E, 0x08, 0x08, 0x09, 0x06},
	// u (0x75)
	{0x00, 0x00, 0x11, 0x11, 0x11, 0x11, 0x0F},
	// v (0x76)
	{0x00, 0x00, 0x11, 0x11, 0x11, 0x0A, 0x04},
	// w (0x77)
	{0x00, 0x00, 0x11, 0x11, 0x15, 0x15, 0x0A},
	// x (0x78)
	{0x00, 0x00, 0x11, 0x0A, 0x04, 0x0A, 0x11},
	// y (0x79)
	{0x00, 0x00, 0x11, 0x11, 0x0F, 0x01, 0x0E},
	// z (0x7A)
	{0x00, 0x00, 0x1F, 0x02, 0x04, 0x08, 0x1F},
	// { (0x7B)
	{0x02, 0x04, 0x04, 0x08, 0x04, 0x04, 0x02},
	// | (0x7C)
	{0x04, 0x04, 0x04, 0x04, 0x04, 0x04, 0x04},
	// } (0x7D)
	{0x08, 0x04, 0x04, 0x02, 0x04, 0x04, 0x08},
	// ~ (0x7E)
	{0x00, 0x00, 0x08, 0x15, 0x02, 0x00, 0x00},
}

// ---------- 自由画笔 ----------

func renderFreehand(img *image.RGBA, a *Annotation) {
	if len(a.Points) < 2 {
		return
	}
	for i := 1; i < len(a.Points); i++ {
		p0 := a.Points[i-1]
		p1 := a.Points[i]
		drawThickLine(img, p0.X, p0.Y, p1.X, p1.Y, a.Color, a.LineWidth)
	}
}

// ---------- 椭圆 ----------

func renderEllipse(img *image.RGBA, a *Annotation) {
	if len(a.Points) < 2 {
		return
	}
	r := canonicalRect(a.Points[0], a.Points[1])
	cx := (r.Min.X + r.Max.X) / 2
	cy := (r.Min.Y + r.Max.Y) / 2
	rx := (r.Max.X - r.Min.X) / 2
	ry := (r.Max.Y - r.Min.Y) / 2

	if rx <= 0 || ry <= 0 {
		return
	}

	if a.Filled {
		// 扫描线填充椭圆（半透明）
		fillColor := a.Color
		fillColor.A = 80
		fillEllipseScanline(img, cx, cy, rx, ry, fillColor)
	}

	// 描边椭圆
	strokeEllipse(img, cx, cy, rx, ry, a.Color, a.LineWidth)
}

// fillEllipseScanline 抗锯齿扫描线填充椭圆
func fillEllipseScanline(img *image.RGBA, cx, cy, rx, ry int, c color.RGBA) {
	rxf := float64(rx)
	ryf := float64(ry)
	cxf := float64(cx)

	for dy := -ry - 1; dy <= ry+1; dy++ {
		t := 1.0 - (float64(dy)*float64(dy))/(ryf*ryf)
		if t < -0.02 {
			continue
		}
		if t < 0 {
			t = 0
		}
		dxf := rxf * math.Sqrt(t)
		xLeft := cxf - dxf
		xRight := cxf + dxf
		y := cy + dy
		// 整数范围内完全填充
		for x := int(math.Ceil(xLeft)); x <= int(math.Floor(xRight)); x++ {
			setPixelBlend(img, x, y, c)
		}
		// 左右边缘抗锯齿
		lx := int(math.Floor(xLeft))
		frac := xLeft - float64(lx)
		if frac < 1 {
			ac := color.RGBA{c.R, c.G, c.B, uint8(float64(c.A) * (1 - frac))}
			setPixelBlend(img, lx, y, ac)
		}
		rx2 := int(math.Ceil(xRight))
		frac2 := float64(rx2) - xRight
		if frac2 < 1 {
			ac := color.RGBA{c.R, c.G, c.B, uint8(float64(c.A) * (1 - frac2))}
			setPixelBlend(img, rx2, y, ac)
		}
	}
}

// strokeEllipse 使用距离场抗锯齿绘制椭圆描边
func strokeEllipse(img *image.RGBA, cx, cy, rx, ry int, c color.RGBA, width int) {
	halfW := float64(width) / 2.0
	if halfW < 0.75 {
		halfW = 0.75
	}
	rxf, ryf := float64(rx), float64(ry)
	cxf, cyf := float64(cx), float64(cy)

	// 外边界和内边界用于快速裁剪
	outerRx := rxf + halfW + 1.5
	outerRy := ryf + halfW + 1.5
	innerRx := rxf - halfW - 1.5
	innerRy := ryf - halfW - 1.5

	for py := cy - int(outerRy) - 1; py <= cy+int(outerRy)+1; py++ {
		dyf := float64(py) - cyf
		for px := cx - int(outerRx) - 1; px <= cx+int(outerRx)+1; px++ {
			dxf := float64(px) - cxf

			// 快速裁剪：在外椭圆之外
			if outerRx > 0 && outerRy > 0 {
				outer := (dxf*dxf)/(outerRx*outerRx) + (dyf*dyf)/(outerRy*outerRy)
				if outer > 1.0 {
					continue
				}
			}
			// 快速裁剪：在内椭圆之内
			if innerRx > 0 && innerRy > 0 {
				inner := (dxf*dxf)/(innerRx*innerRx) + (dyf*dyf)/(innerRy*innerRy)
				if inner < 1.0 {
					continue
				}
			}

			dist := ellipsePointDist(float64(px), float64(py), cxf, cyf, rxf, ryf)
			renderAAPixel(img, px, py, c, dist, halfW)
		}
	}
}

// ellipsePointDist 计算点到椭圆的近似距离
func ellipsePointDist(px, py, cx, cy, rx, ry float64) float64 {
	dx := (px - cx) / rx
	dy := (py - cy) / ry
	r := math.Hypot(dx, dy)
	if r < 0.001 {
		if rx < ry {
			return rx
		}
		return ry
	}
	// 同方向上椭圆表面点
	t := 1.0 / r
	ex := cx + rx*dx*t
	ey := cy + ry*dy*t
	return math.Hypot(px-ex, py-ey)
}

// ---------- 马赛克 ----------

func renderMosaic(img *image.RGBA, a *Annotation) {
	if len(a.Points) < 2 {
		return
	}

	blockSize := a.MosaicPx
	if blockSize <= 0 {
		blockSize = 10
	}

	r := canonicalRect(a.Points[0], a.Points[1])
	bounds := img.Bounds()

	// 裁剪到图片范围内
	r = r.Intersect(bounds)
	if r.Empty() {
		return
	}

	// 将区域分割成 blockSize × blockSize 的块
	for by := r.Min.Y; by < r.Max.Y; by += blockSize {
		for bx := r.Min.X; bx < r.Max.X; bx += blockSize {
			// 计算当前块的实际范围
			bx1 := bx + blockSize
			by1 := by + blockSize
			if bx1 > r.Max.X {
				bx1 = r.Max.X
			}
			if by1 > r.Max.Y {
				by1 = r.Max.Y
			}

			// 计算块内所有像素的平均颜色
			var sumR, sumG, sumB uint64
			count := uint64(0)
			for y := by; y < by1; y++ {
				for x := bx; x < bx1; x++ {
					off := (y-bounds.Min.Y)*img.Stride + (x-bounds.Min.X)*4
					sumR += uint64(img.Pix[off+0])
					sumG += uint64(img.Pix[off+1])
					sumB += uint64(img.Pix[off+2])
					count++
				}
			}

			if count == 0 {
				continue
			}

			avgColor := color.RGBA{
				R: uint8(sumR / count),
				G: uint8(sumG / count),
				B: uint8(sumB / count),
				A: 255,
			}

			// 用平均颜色填充整个块
			for y := by; y < by1; y++ {
				for x := bx; x < bx1; x++ {
					off := (y-bounds.Min.Y)*img.Stride + (x-bounds.Min.X)*4
					img.Pix[off+0] = avgColor.R
					img.Pix[off+1] = avgColor.G
					img.Pix[off+2] = avgColor.B
					img.Pix[off+3] = avgColor.A
				}
			}
		}
	}
}

// ========== 辅助绘图函数 ==========

// drawThickLine 使用距离场抗锯齿绘制线段（圆头端点）
func drawThickLine(img *image.RGBA, x1, y1, x2, y2 int, c color.RGBA, width int) {
	halfW := float64(width) / 2.0
	if halfW < 0.75 {
		halfW = 0.75
	}

	dx := float64(x2 - x1)
	dy := float64(y2 - y1)
	length := math.Hypot(dx, dy)

	if length < 0.5 {
		// 两点重合，画一个圆点
		drawFilledCircleAA(img, float64(x1), float64(y1), halfW, c)
		return
	}

	// 单位方向和法向量
	ux, uy := dx/length, dy/length
	nx, ny := -uy, ux

	// 扫描包围盒
	margin := int(halfW) + 2
	bx0, bx1 := x1, x2
	if bx0 > bx1 {
		bx0, bx1 = bx1, bx0
	}
	by0, by1 := y1, y2
	if by0 > by1 {
		by0, by1 = by1, by0
	}
	bx0 -= margin
	bx1 += margin
	by0 -= margin
	by1 += margin

	x1f, y1f := float64(x1), float64(y1)
	x2f, y2f := float64(x2), float64(y2)

	for py := by0; py <= by1; py++ {
		for px := bx0; px <= bx1; px++ {
			vx := float64(px) - x1f
			vy := float64(py) - y1f
			along := vx*ux + vy*uy

			var dist float64
			if along <= 0 {
				dist = math.Hypot(vx, vy)
			} else if along >= length {
				dist = math.Hypot(float64(px)-x2f, float64(py)-y2f)
			} else {
				dist = math.Abs(vx*nx + vy*ny)
			}

			renderAAPixel(img, px, py, c, dist, halfW)
		}
	}
}

// renderAAPixel 根据距离渲染抗锯齿像素
func renderAAPixel(img *image.RGBA, x, y int, c color.RGBA, dist, halfW float64) {
	if dist > halfW+0.5 {
		return
	}
	if dist <= halfW-0.5 {
		setPixelBlend(img, x, y, c)
	} else {
		frac := halfW + 0.5 - dist
		ac := color.RGBA{c.R, c.G, c.B, uint8(float64(c.A) * frac)}
		setPixelBlend(img, x, y, ac)
	}
}

// drawFilledCircleAA 绘制抗锯齿填充圆
func drawFilledCircleAA(img *image.RGBA, cx, cy, r float64, c color.RGBA) {
	ri := int(r) + 2
	cxi, cyi := int(cx), int(cy)
	for py := cyi - ri; py <= cyi+ri; py++ {
		for px := cxi - ri; px <= cxi+ri; px++ {
			dist := math.Hypot(float64(px)-cx, float64(py)-cy)
			renderAAPixel(img, px, py, c, dist, r)
		}
	}
}

// drawFilledTriangle 绘制填充三角形（用于箭头）
func drawFilledTriangle(img *image.RGBA, p1, p2, p3 image.Point, c color.RGBA) {
	// 确定扫描线范围
	minY := min3(p1.Y, p2.Y, p3.Y)
	maxY := max3(p1.Y, p2.Y, p3.Y)

	for y := minY; y <= maxY; y++ {
		// 找到三角形在此 y 行的 x 交点
		var xs []int

		xs = appendEdgeX(xs, y, p1, p2)
		xs = appendEdgeX(xs, y, p2, p3)
		xs = appendEdgeX(xs, y, p3, p1)

		if len(xs) < 2 {
			continue
		}

		// 找最小和最大 x
		xMin := xs[0]
		xMax := xs[0]
		for _, x := range xs[1:] {
			if x < xMin {
				xMin = x
			}
			if x > xMax {
				xMax = x
			}
		}

		for x := xMin; x <= xMax; x++ {
			setPixelBlend(img, x, y, c)
		}
	}
}

// appendEdgeX 计算扫描线 y 与边 (a, b) 的交点 x
func appendEdgeX(xs []int, y int, a, b image.Point) []int {
	if a.Y > b.Y {
		a, b = b, a
	}
	if y < a.Y || y > b.Y || a.Y == b.Y {
		return xs
	}
	// 线性插值
	t := float64(y-a.Y) / float64(b.Y-a.Y)
	x := int(math.Round(float64(a.X) + t*float64(b.X-a.X)))
	return append(xs, x)
}

// setPixelBlend 混合绘制像素（支持半透明）
func setPixelBlend(img *image.RGBA, x, y int, c color.RGBA) {
	bounds := img.Bounds()
	if x < bounds.Min.X || x >= bounds.Max.X || y < bounds.Min.Y || y >= bounds.Max.Y {
		return
	}

	if c.A == 255 {
		// 不透明，直接写入
		off := (y-bounds.Min.Y)*img.Stride + (x-bounds.Min.X)*4
		img.Pix[off+0] = c.R
		img.Pix[off+1] = c.G
		img.Pix[off+2] = c.B
		img.Pix[off+3] = 255
		return
	}

	if c.A == 0 {
		return
	}

	// Alpha 混合
	off := (y-bounds.Min.Y)*img.Stride + (x-bounds.Min.X)*4
	srcA := uint32(c.A)
	invA := 255 - srcA

	img.Pix[off+0] = uint8((uint32(c.R)*srcA + uint32(img.Pix[off+0])*invA) / 255)
	img.Pix[off+1] = uint8((uint32(c.G)*srcA + uint32(img.Pix[off+1])*invA) / 255)
	img.Pix[off+2] = uint8((uint32(c.B)*srcA + uint32(img.Pix[off+2])*invA) / 255)
	img.Pix[off+3] = uint8(srcA + uint32(img.Pix[off+3])*invA/255)
}

// ========== 通用辅助函数 ==========

// canonicalRect 将两个点转换为规范化的矩形（保证 Min <= Max）
func canonicalRect(p1, p2 image.Point) image.Rectangle {
	x0, x1 := p1.X, p2.X
	y0, y1 := p1.Y, p2.Y
	if x0 > x1 {
		x0, x1 = x1, x0
	}
	if y0 > y1 {
		y0, y1 = y1, y0
	}
	return image.Rect(x0, y0, x1, y1)
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func min3(a, b, c int) int {
	m := a
	if b < m {
		m = b
	}
	if c < m {
		m = c
	}
	return m
}

func max3(a, b, c int) int {
	m := a
	if b > m {
		m = b
	}
	if c > m {
		m = c
	}
	return m
}
