package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Hotkey 快捷键配置
type Hotkey struct {
	Modifiers []string `json:"modifiers"` // ctrl, alt, shift, win(windows)/cmd(mac)
	Key       string   `json:"key"`       // 主键，如 s, a, 1, f1 等
}

// Storage 存储配置
type Storage struct {
	Directory string `json:"directory"` // 保存目录
	Format    string `json:"format"`    // 图片格式: png, jpg
	Quality   int    `json:"quality"`   // jpg质量 1-100
}

// Behavior 行为配置
type Behavior struct {
	ShowNotification bool `json:"showNotification"` // 显示通知
	PlaySound        bool `json:"playSound"`        // 播放声音
	AutoStart        bool `json:"autoStart"`        // 开机启动
}

// Config 主配置结构
type Config struct {
	Hotkey   Hotkey   `json:"hotkey"`
	Storage  Storage  `json:"storage"`
	Behavior Behavior `json:"behavior"`
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	// 获取 exe 所在目录
	exePath, _ := os.Executable()
	exeDir := filepath.Dir(exePath)

	return &Config{
		Hotkey: Hotkey{
			Modifiers: []string{"alt"},
			Key:       "1",
		},
		Storage: Storage{
			Directory: exeDir,
			Format:    "png",
			Quality:   90,
		},
		Behavior: Behavior{
			ShowNotification: true,
			PlaySound:        false,
			AutoStart:        false,
		},
	}
}

// GetConfigPath 获取配置文件路径
func GetConfigPath() string {
	var configDir string

	if runtime.GOOS == "windows" {
		configDir = os.Getenv("APPDATA")
		if configDir == "" {
			homeDir, _ := os.UserHomeDir()
			configDir = filepath.Join(homeDir, "AppData", "Roaming")
		}
	} else {
		homeDir, _ := os.UserHomeDir()
		configDir = filepath.Join(homeDir, ".config")
	}

	return filepath.Join(configDir, "snapcli", "config.json")
}

// Load 加载配置
func Load() (*Config, error) {
	configPath := GetConfigPath()

	// 如果配置文件不存在，返回默认配置
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		cfg := DefaultConfig()
		// 保存默认配置
		_ = cfg.Save()
		return cfg, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return DefaultConfig(), err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return DefaultConfig(), err
	}

	// 验证并修正配置
	cfg.Validate()

	return &cfg, nil
}

// Validate 验证并修正配置值
func (c *Config) Validate() {
	defaults := DefaultConfig()

	// 验证图片质量 (1-100)
	if c.Storage.Quality < 1 || c.Storage.Quality > 100 {
		c.Storage.Quality = defaults.Storage.Quality
	}

	// 验证图片格式
	format := strings.ToLower(c.Storage.Format)
	if format != "png" && format != "jpg" && format != "jpeg" {
		c.Storage.Format = defaults.Storage.Format
	} else {
		c.Storage.Format = format
	}

	// 防止路径遍历攻击
	if strings.Contains(c.Storage.Directory, "..") {
		c.Storage.Directory = defaults.Storage.Directory
	}

	// 验证快捷键
	if c.Hotkey.Key == "" {
		c.Hotkey = defaults.Hotkey
	}

	// 验证修饰键
	validMods := map[string]bool{"ctrl": true, "alt": true, "shift": true, "win": true, "cmd": true, "control": true, "option": true, "super": true, "command": true}
	validatedMods := []string{}
	for _, mod := range c.Hotkey.Modifiers {
		if validMods[strings.ToLower(mod)] {
			validatedMods = append(validatedMods, strings.ToLower(mod))
		}
	}
	if len(validatedMods) == 0 {
		c.Hotkey.Modifiers = defaults.Hotkey.Modifiers
	} else {
		c.Hotkey.Modifiers = validatedMods
	}
}

// Save 保存配置
func (c *Config) Save() error {
	configPath := GetConfigPath()

	// 确保目录存在
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "    ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}

// SetHotkey 设置快捷键
func (c *Config) SetHotkey(modifiers []string, key string) error {
	c.Hotkey.Modifiers = modifiers
	c.Hotkey.Key = key
	return c.Save()
}

// GetHotkeyString 获取快捷键的字符串表示
func (c *Config) GetHotkeyString() string {
	result := ""
	for i, mod := range c.Hotkey.Modifiers {
		if i > 0 {
			result += "+"
		}
		result += mod
	}
	if len(c.Hotkey.Modifiers) > 0 {
		result += "+"
	}
	result += c.Hotkey.Key
	return result
}

// EnsureStorageDir 确保存储目录存在
func (c *Config) EnsureStorageDir() error {
	// 展开 ~
	dir := c.Storage.Directory
	if len(dir) > 0 && dir[0] == '~' {
		homeDir, _ := os.UserHomeDir()
		dir = filepath.Join(homeDir, dir[1:])
	}
	c.Storage.Directory = dir

	return os.MkdirAll(dir, 0755)
}
