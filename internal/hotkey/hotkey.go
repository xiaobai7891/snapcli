package hotkey

import (
	"fmt"
	"strings"

	"golang.design/x/hotkey"
	"golang.design/x/hotkey/mainthread"
)

// Manager 热键管理器
type Manager struct {
	hk       *hotkey.Hotkey
	callback func()
}

// NewManager 创建热键管理器
func NewManager() *Manager {
	return &Manager{}
}

// parseModifiers 解析修饰键
func parseModifiers(mods []string) []hotkey.Modifier {
	var result []hotkey.Modifier
	for _, mod := range mods {
		switch strings.ToLower(mod) {
		case "ctrl", "control":
			result = append(result, hotkey.ModCtrl)
		case "alt", "option":
			result = append(result, hotkey.ModAlt)
		case "shift":
			result = append(result, hotkey.ModShift)
		case "win", "cmd", "command", "super":
			result = append(result, hotkey.ModWin)
		}
	}
	return result
}

// parseKey 解析主键
func parseKey(key string) hotkey.Key {
	key = strings.ToUpper(key)

	// 字母键
	if len(key) == 1 && key[0] >= 'A' && key[0] <= 'Z' {
		return hotkey.Key(key[0])
	}

	// 数字键
	if len(key) == 1 && key[0] >= '0' && key[0] <= '9' {
		return hotkey.Key(key[0])
	}

	// 功能键
	switch key {
	case "F1":
		return hotkey.KeyF1
	case "F2":
		return hotkey.KeyF2
	case "F3":
		return hotkey.KeyF3
	case "F4":
		return hotkey.KeyF4
	case "F5":
		return hotkey.KeyF5
	case "F6":
		return hotkey.KeyF6
	case "F7":
		return hotkey.KeyF7
	case "F8":
		return hotkey.KeyF8
	case "F9":
		return hotkey.KeyF9
	case "F10":
		return hotkey.KeyF10
	case "F11":
		return hotkey.KeyF11
	case "F12":
		return hotkey.KeyF12
	case "SPACE":
		return hotkey.KeySpace
	case "RETURN", "ENTER":
		return hotkey.KeyReturn
	case "ESCAPE", "ESC":
		return hotkey.KeyEscape
	case "TAB":
		return hotkey.KeyTab
	case "CAPSLOCK":
		return hotkey.Key(0x14) // VK_CAPITAL
	case "DELETE", "DEL":
		return hotkey.KeyDelete
	case "UP":
		return hotkey.KeyUp
	case "DOWN":
		return hotkey.KeyDown
	case "LEFT":
		return hotkey.KeyLeft
	case "RIGHT":
		return hotkey.KeyRight
	}

	// 默认返回 S
	return hotkey.KeyS
}

// Register 注册热键
func (m *Manager) Register(modifiers []string, key string, callback func()) error {
	mods := parseModifiers(modifiers)
	k := parseKey(key)

	fmt.Printf("注册热键: modifiers=%v, key=%s, keyCode=0x%X\n", mods, key, k)

	m.hk = hotkey.New(mods, k)
	m.callback = callback

	if err := m.hk.Register(); err != nil {
		return fmt.Errorf("无法注册热键: %v", err)
	}

	return nil
}

// Unregister 注销热键
func (m *Manager) Unregister() error {
	if m.hk != nil {
		return m.hk.Unregister()
	}
	return nil
}

// Listen 开始监听热键（阻塞）
func (m *Manager) Listen() {
	for range m.hk.Keydown() {
		if m.callback != nil {
			m.callback()
		}
	}
}

// ListenAsync 异步监听热键
func (m *Manager) ListenAsync() {
	go m.Listen()
}

// Run 在主线程中运行（某些平台需要）
func Run(fn func()) {
	mainthread.Init(fn)
}

// GetSupportedModifiers 获取支持的修饰键列表
func GetSupportedModifiers() []string {
	return []string{"ctrl", "alt", "shift", "win"}
}

// GetSupportedKeys 获取支持的主键列表
func GetSupportedKeys() []string {
	keys := []string{}

	// 字母
	for c := 'a'; c <= 'z'; c++ {
		keys = append(keys, string(c))
	}

	// 数字
	for c := '0'; c <= '9'; c++ {
		keys = append(keys, string(c))
	}

	// 功能键
	for i := 1; i <= 12; i++ {
		keys = append(keys, fmt.Sprintf("f%d", i))
	}

	return keys
}
