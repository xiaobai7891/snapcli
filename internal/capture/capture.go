package capture

import (
	"image"
)

// Region 截图区域
type Region struct {
	X      int
	Y      int
	Width  int
	Height int
}

// Display 显示器信息
type Display struct {
	Index       int
	X           int
	Y           int
	Width       int
	Height      int
	ScaleFactor float64
}

// Capturer 截图接口
type Capturer interface {
	// CaptureFullScreen 全屏截图
	CaptureFullScreen() (*image.RGBA, error)

	// CaptureRegion 截取指定区域
	CaptureRegion(region Region) (*image.RGBA, error)

	// GetDisplays 获取所有显示器信息
	GetDisplays() ([]Display, error)

	// GetFullBounds 获取所有显示器的总边界
	GetFullBounds() Region
}

// BytesPerPixel RGBA 格式每像素字节数
const BytesPerPixel = 4

// CropImage 裁剪图片（使用内存直接复制，性能优化）
func CropImage(img *image.RGBA, region Region) *image.RGBA {
	bounds := img.Bounds()

	// 确保区域在图片范围内
	if region.X < bounds.Min.X {
		region.X = bounds.Min.X
	}
	if region.Y < bounds.Min.Y {
		region.Y = bounds.Min.Y
	}
	if region.X+region.Width > bounds.Max.X {
		region.Width = bounds.Max.X - region.X
	}
	if region.Y+region.Height > bounds.Max.Y {
		region.Height = bounds.Max.Y - region.Y
	}

	// 边界检查
	if region.Width <= 0 || region.Height <= 0 {
		return image.NewRGBA(image.Rect(0, 0, 0, 0))
	}

	// 创建新图片
	cropped := image.NewRGBA(image.Rect(0, 0, region.Width, region.Height))

	// 使用内存直接复制（O(n) 而非 O(n²)）
	srcStride := img.Stride
	dstStride := cropped.Stride
	bytesPerRow := region.Width * BytesPerPixel

	for y := 0; y < region.Height; y++ {
		srcStart := (region.Y+y)*srcStride + region.X*BytesPerPixel
		dstStart := y * dstStride
		copy(cropped.Pix[dstStart:dstStart+bytesPerRow], img.Pix[srcStart:srcStart+bytesPerRow])
	}

	return cropped
}
