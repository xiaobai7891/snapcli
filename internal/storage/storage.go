package storage

import (
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"time"
)

// Storage 存储管理
type Storage struct {
	directory string
	format    string
	quality   int
}

// NewStorage 创建存储管理器
func NewStorage(directory, format string, quality int) *Storage {
	return &Storage{
		directory: directory,
		format:    format,
		quality:   quality,
	}
}

// SetDirectory 设置保存目录
func (s *Storage) SetDirectory(dir string) error {
	// 展开 ~
	if len(dir) > 0 && dir[0] == '~' {
		homeDir, _ := os.UserHomeDir()
		dir = filepath.Join(homeDir, dir[1:])
	}

	s.directory = dir
	return os.MkdirAll(dir, 0755)
}

// Save 保存图片，返回文件路径
func (s *Storage) Save(img image.Image) (string, error) {
	// 确保目录存在
	if err := os.MkdirAll(s.directory, 0755); err != nil {
		return "", fmt.Errorf("无法创建目录: %v", err)
	}

	// 生成文件名
	timestamp := time.Now().Format("20060102_150405")
	ext := s.format
	if ext == "" {
		ext = "png"
	}
	filename := fmt.Sprintf("screenshot_%s.%s", timestamp, ext)
	filepath := filepath.Join(s.directory, filename)

	// 创建文件
	file, err := os.Create(filepath)
	if err != nil {
		return "", fmt.Errorf("无法创建文件: %v", err)
	}
	defer file.Close()

	// 编码并保存
	switch s.format {
	case "jpg", "jpeg":
		err = jpeg.Encode(file, img, &jpeg.Options{Quality: s.quality})
	default:
		err = png.Encode(file, img)
	}

	if err != nil {
		return "", fmt.Errorf("无法保存图片: %v", err)
	}

	return filepath, nil
}

// Cleanup 清理旧截图
func (s *Storage) Cleanup(olderThan time.Duration) error {
	entries, err := os.ReadDir(s.directory)
	if err != nil {
		return err
	}

	cutoff := time.Now().Add(-olderThan)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		if info.ModTime().Before(cutoff) {
			os.Remove(filepath.Join(s.directory, entry.Name()))
		}
	}

	return nil
}

// GetDirectory 获取保存目录
func (s *Storage) GetDirectory() string {
	return s.directory
}
