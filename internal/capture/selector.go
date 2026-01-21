package capture

import (
	"image"
)

// Selector 选区接口
type Selector interface {
	// SelectRegion 显示选区UI，返回用户选择的区域
	// background: 全屏截图作为背景
	// 返回 nil 表示用户取消
	SelectRegion(background *image.RGBA) (*Region, error)
}
