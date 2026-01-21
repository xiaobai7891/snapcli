//go:build darwin

package capture

import (
	"fmt"
	"image"
	"os/exec"
	"strconv"
	"strings"
)

// DarwinCapturer macOS截图实现
type DarwinCapturer struct {
	displays []Display
}

// NewCapturer 创建截图器
func NewCapturer() Capturer {
	return &DarwinCapturer{}
}

// GetDisplays 获取所有显示器信息
func (c *DarwinCapturer) GetDisplays() ([]Display, error) {
	// 使用 system_profiler 获取显示器信息
	cmd := exec.Command("system_profiler", "SPDisplaysDataType", "-json")
	output, err := cmd.Output()
	if err != nil {
		// 返回默认显示器
		c.displays = []Display{{
			Index:       0,
			X:           0,
			Y:           0,
			Width:       1920,
			Height:      1080,
			ScaleFactor: 2.0,
		}}
		return c.displays, nil
	}

	// 简单解析，实际应该用JSON解析
	_ = output
	c.displays = []Display{{
		Index:       0,
		X:           0,
		Y:           0,
		Width:       1920,
		Height:      1080,
		ScaleFactor: 2.0,
	}}

	return c.displays, nil
}

// GetFullBounds 获取所有显示器的总边界
func (c *DarwinCapturer) GetFullBounds() Region {
	// 使用 screencapture 获取全屏尺寸
	// 这里简化处理，返回主显示器大小
	cmd := exec.Command("osascript", "-e", `tell application "Finder" to get bounds of window of desktop`)
	output, err := cmd.Output()
	if err == nil {
		parts := strings.Split(strings.TrimSpace(string(output)), ", ")
		if len(parts) == 4 {
			width, _ := strconv.Atoi(parts[2])
			height, _ := strconv.Atoi(parts[3])
			return Region{X: 0, Y: 0, Width: width, Height: height}
		}
	}

	return Region{X: 0, Y: 0, Width: 1920, Height: 1080}
}

// CaptureFullScreen 全屏截图
func (c *DarwinCapturer) CaptureFullScreen() (*image.RGBA, error) {
	bounds := c.GetFullBounds()
	return c.CaptureRegion(bounds)
}

// CaptureRegion 截取指定区域
func (c *DarwinCapturer) CaptureRegion(region Region) (*image.RGBA, error) {
	// 使用 screencapture 命令行工具
	// -x: 不播放声音
	// -R: 指定区域
	// -t png: 输出格式
	// -c: 输出到剪贴板 (我们用临时文件)

	tmpFile := "/tmp/snapcli_capture.png"
	cmd := exec.Command("screencapture",
		"-x",
		"-R", fmt.Sprintf("%d,%d,%d,%d", region.X, region.Y, region.Width, region.Height),
		tmpFile,
	)

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("截图失败: %v", err)
	}

	// 读取图片文件
	// 这里简化处理，实际应该用 image/png 解码
	return nil, fmt.Errorf("macOS截图功能需要完善")
}
