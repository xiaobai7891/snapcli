# SnapCLI

[English](#english) | [中文](#中文)

---

## English

A lightweight screenshot tool designed for CLI-based AI assistants (Claude Code, Cursor, Aider, etc.).

**One hotkey to capture, file path automatically copied to clipboard.**

### Why SnapCLI?

When using CLI AI tools like Claude Code, sharing screenshots is often needed. However, the typical workflow is cumbersome:

1. Take a screenshot with another tool
2. Save the image file manually
3. Navigate to find the file
4. Copy the file path
5. Paste into the terminal

**SnapCLI simplifies this to: Press hotkey → Select region → Paste path**

### Features

- **One-key workflow**: Hotkey → Select area → Path in clipboard
- **Smart selection**: Click to capture detected window, or drag to select custom region
- **Visual feedback**: Dark overlay with highlighted selection area
- **Custom crosshair cursor**: Visible on any background (white outline + red center)
- **System tray**: Runs quietly in background
- **Configurable hotkey**: Default `Alt+1`, fully customizable
- **Optimized performance**: Dirty region rendering, O(n) image processing

### Installation

#### Windows

Download `snapcli.exe` from [Releases](../../releases) and run it.

#### Build from Source

```bash
# Windows
go build -ldflags="-H windowsgui" -o snapcli.exe ./cmd/snapcli

# Or use the build script
build.bat
```

### Usage

1. Run the program (icon appears in system tray)
2. Press `Alt+1` to start capture
3. **Click** on a window to capture it, or **drag** to select a custom region
4. Screenshot saved, path copied to clipboard
5. Paste the path in your terminal with `Ctrl+V`

#### Keyboard Shortcuts

| Action | Key |
|--------|-----|
| Capture | `Alt+1` (default) |
| Cancel | `ESC` or Right-click |
| Full screen | `Enter` |

#### Customize Hotkey

```bash
snapcli --set-hotkey ctrl+shift+s
snapcli --config  # Show config file path
```

#### Configuration

Config file location:
- Windows: `%APPDATA%\snapcli\config.json`
- macOS: `~/.config/snapcli/config.json`

```json
{
    "hotkey": {
        "modifiers": ["alt"],
        "key": "1"
    },
    "storage": {
        "directory": "./screenshots",
        "format": "png",
        "quality": 90
    },
    "behavior": {
        "showNotification": true
    }
}
```

---

## 中文

专为 CLI AI 工具（Claude Code、Cursor、Aider 等）设计的轻量级截图助手。

**一键截图，路径自动复制到剪贴板。**

### 为什么做这个工具？

在使用 Claude Code 等 CLI AI 工具时，经常需要分享截图给 AI。但传统流程很繁琐：

1. 用其他截图工具截图
2. 手动保存图片文件
3. 找到保存的文件
4. 复制文件路径
5. 粘贴到终端

**SnapCLI 把这个流程简化为：按快捷键 → 选择区域 → 粘贴路径**

### 功能特点

- **一键操作**：快捷键 → 选区 → 路径自动进剪贴板
- **智能选区**：点击自动识别窗口，拖动自定义区域
- **视觉反馈**：暗色遮罩 + 高亮选区
- **自定义光标**：白色轮廓 + 红色中心，任何背景下都清晰可见
- **系统托盘**：后台静默运行
- **自定义快捷键**：默认 `Alt+1`，可自由配置
- **性能优化**：脏区域渲染、O(n) 图像处理

### 安装

#### Windows

从 [Releases](../../releases) 下载 `snapcli.exe`，双击运行即可。

#### 从源码构建

```bash
# Windows
go build -ldflags="-H windowsgui" -o snapcli.exe ./cmd/snapcli

# 或使用构建脚本
build.bat
```

### 使用方法

1. 运行程序，图标出现在系统托盘
2. 按 `Alt+1` 开始截图
3. **单击**窗口直接截取该窗口，或**拖动**选择自定义区域
4. 截图完成，路径已复制到剪贴板
5. 在终端中 `Ctrl+V` 粘贴路径

#### 快捷键

| 操作 | 按键 |
|------|------|
| 截图 | `Alt+1`（默认） |
| 取消 | `ESC` 或 鼠标右键 |
| 全屏截图 | `Enter` |

#### 自定义快捷键

```bash
snapcli --set-hotkey ctrl+shift+s
snapcli --config  # 查看配置文件位置
```

#### 配置文件

配置文件位置：
- Windows: `%APPDATA%\snapcli\config.json`
- macOS: `~/.config/snapcli/config.json`

```json
{
    "hotkey": {
        "modifiers": ["alt"],
        "key": "1"
    },
    "storage": {
        "directory": "./screenshots",
        "format": "png",
        "quality": 90
    },
    "behavior": {
        "showNotification": true
    }
}
```

---

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Contributing

Issues and Pull Requests are welcome!
