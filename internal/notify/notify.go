package notify

// Notifier 通知接口
type Notifier interface {
	Show(title, message string) error
}
