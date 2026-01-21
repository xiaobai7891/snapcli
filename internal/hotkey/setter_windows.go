//go:build windows

package hotkey

import (
	"fmt"
	"os/exec"
	"strings"
)

type HotkeyResult struct {
	Modifiers []string
	Key       string
	Cancelled bool
}

// ShowHotkeySetter 显示快捷键设置对话框
// 使用 PowerShell InputBox，避免 Windows GUI 线程问题
func ShowHotkeySetter(currentHotkey string) *HotkeyResult {
	fmt.Println("正在打开快捷键设置对话框...")

	// 使用 PowerShell 显示输入框
	// 注意：PowerShell 中用 `n 表示换行
	script := fmt.Sprintf(`
Add-Type -AssemblyName Microsoft.VisualBasic
$msg = "请输入新的快捷键组合" + [char]10 + [char]10 + "格式: 修饰键+主键" + [char]10 + "示例: alt+1, ctrl+shift+s, alt+a" + [char]10 + [char]10 + "支持的修饰键: ctrl, alt, shift, win" + [char]10 + "支持的主键: a-z, 0-9, f1-f12"
$result = [Microsoft.VisualBasic.Interaction]::InputBox($msg, "设置快捷键", "%s")
Write-Output $result
`, currentHotkey)

	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script)
	output, err := cmd.Output()
	if err != nil {
		fmt.Println("PowerShell 执行失败:", err)
		return nil
	}

	result := strings.TrimSpace(string(output))
	if result == "" {
		fmt.Println("用户取消或输入为空")
		return nil
	}

	fmt.Println("用户输入:", result)

	// 解析快捷键字符串
	return parseHotkeyString(result)
}

// parseHotkeyString 解析快捷键字符串，如 "ctrl+alt+s"
func parseHotkeyString(s string) *HotkeyResult {
	s = strings.ToLower(strings.TrimSpace(s))
	parts := strings.Split(s, "+")

	if len(parts) < 2 {
		fmt.Println("快捷键格式无效，需要至少一个修饰键和一个主键")
		return nil
	}

	result := &HotkeyResult{
		Modifiers: []string{},
		Key:       "",
	}

	for i, part := range parts {
		part = strings.TrimSpace(part)
		if i == len(parts)-1 {
			// 最后一个是主键
			result.Key = part
		} else {
			// 前面的是修饰键
			switch part {
			case "ctrl", "control":
				result.Modifiers = append(result.Modifiers, "ctrl")
			case "alt", "option":
				result.Modifiers = append(result.Modifiers, "alt")
			case "shift":
				result.Modifiers = append(result.Modifiers, "shift")
			case "win", "cmd", "command", "super":
				result.Modifiers = append(result.Modifiers, "win")
			default:
				fmt.Printf("未知的修饰键: %s\n", part)
				return nil
			}
		}
	}

	if len(result.Modifiers) == 0 {
		fmt.Println("需要至少一个修饰键")
		return nil
	}

	if result.Key == "" {
		fmt.Println("需要一个主键")
		return nil
	}

	// 验证主键
	key := strings.ToUpper(result.Key)
	validKey := false

	// 检查字母
	if len(key) == 1 && key[0] >= 'A' && key[0] <= 'Z' {
		validKey = true
	}
	// 检查数字
	if len(key) == 1 && key[0] >= '0' && key[0] <= '9' {
		validKey = true
	}
	// 检查功能键
	if strings.HasPrefix(key, "F") && len(key) <= 3 {
		validKey = true
	}

	if !validKey {
		fmt.Printf("无效的主键: %s (支持 a-z, 0-9, f1-f12)\n", result.Key)
		return nil
	}

	return result
}

// ValidateHotkey 验证快捷键是否有效
func ValidateHotkey(mods []string, key string) error {
	if len(mods) == 0 {
		return fmt.Errorf("需要至少一个修饰键 (Ctrl/Alt/Shift/Win)")
	}
	if key == "" {
		return fmt.Errorf("需要一个主键 (A-Z, 0-9, F1-F12)")
	}
	return nil
}
