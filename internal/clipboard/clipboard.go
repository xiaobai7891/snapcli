package clipboard

// Clipboard 剪贴板接口
type Clipboard interface {
	SetText(text string) error
	GetText() (string, error)
}
