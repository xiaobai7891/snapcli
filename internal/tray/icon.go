package tray

// 生成一个简单的16x16蓝色相机图标 (ICO格式)
func getIcon() []byte {
	// 16x16 32位 ICO 图标
	// 这是一个有效的ICO文件格式
	return createSimpleIcon()
}

func createSimpleIcon() []byte {
	width := 16
	height := 16

	// ICO 文件头
	header := []byte{
		0x00, 0x00, // Reserved
		0x01, 0x00, // Type: 1 = ICO
		0x01, 0x00, // Count: 1 image
	}

	// 图像目录条目
	imageSize := width * height * 4 // 32位 RGBA
	bmpHeaderSize := 40
	totalImageSize := bmpHeaderSize + imageSize

	entry := []byte{
		byte(width),        // Width
		byte(height),       // Height
		0x00,               // Color palette
		0x00,               // Reserved
		0x01, 0x00,         // Color planes
		0x20, 0x00,         // Bits per pixel (32)
		byte(totalImageSize),        // Size of image data (low byte)
		byte(totalImageSize >> 8),   // Size of image data
		byte(totalImageSize >> 16),  // Size of image data
		byte(totalImageSize >> 24),  // Size of image data (high byte)
		0x16, 0x00, 0x00, 0x00,      // Offset to image data (22 bytes)
	}

	// BITMAPINFOHEADER
	bmpHeader := []byte{
		0x28, 0x00, 0x00, 0x00, // Header size (40)
		byte(width), 0x00, 0x00, 0x00,  // Width
		byte(height * 2), 0x00, 0x00, 0x00, // Height (doubled for XOR + AND mask)
		0x01, 0x00,             // Planes
		0x20, 0x00,             // Bits per pixel (32)
		0x00, 0x00, 0x00, 0x00, // Compression
		0x00, 0x00, 0x00, 0x00, // Image size (can be 0 for uncompressed)
		0x00, 0x00, 0x00, 0x00, // X pixels per meter
		0x00, 0x00, 0x00, 0x00, // Y pixels per meter
		0x00, 0x00, 0x00, 0x00, // Colors used
		0x00, 0x00, 0x00, 0x00, // Important colors
	}

	// 像素数据 (BGRA格式，从下往上)
	pixels := make([]byte, imageSize)

	// 绘制一个简单的相机图标（蓝色背景 + 白色相机形状）
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			idx := ((height - 1 - y) * width + x) * 4 // ICO是从下往上的

			// 默认透明
			b, g, r, a := byte(0), byte(0), byte(0), byte(0)

			// 圆形背景
			cx, cy := float64(x)-7.5, float64(y)-7.5
			dist := cx*cx + cy*cy

			if dist < 56 { // 半径约7.5的圆
				// 蓝色背景 (#0078D4)
				b, g, r, a = 0xD4, 0x78, 0x00, 0xFF

				// 相机镜头（白色圆圈）
				if dist < 20 && dist > 8 {
					b, g, r = 0xFF, 0xFF, 0xFF
				}
				// 镜头中心
				if dist < 5 {
					b, g, r = 0xFF, 0xFF, 0xFF
				}

				// 相机顶部凸起
				if y >= 2 && y <= 4 && x >= 5 && x <= 10 {
					b, g, r = 0xD4, 0x78, 0x00
				}
			}

			pixels[idx+0] = b
			pixels[idx+1] = g
			pixels[idx+2] = r
			pixels[idx+3] = a
		}
	}

	// 组合所有部分
	result := make([]byte, 0, len(header)+len(entry)+len(bmpHeader)+len(pixels))
	result = append(result, header...)
	result = append(result, entry...)
	result = append(result, bmpHeader...)
	result = append(result, pixels...)

	return result
}
