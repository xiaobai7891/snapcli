//go:build windows

package notify

import (
	"github.com/go-toast/toast"
)

// WindowsNotifier Windows通知实现
type WindowsNotifier struct {
	appID string
}

// NewNotifier 创建通知器
func NewNotifier() Notifier {
	return &WindowsNotifier{
		appID: "SnapCLI",
	}
}

// Show 显示通知（异步，不阻塞主流程）
func (n *WindowsNotifier) Show(title, message string) error {
	go func() {
		notification := toast.Notification{
			AppID:   n.appID,
			Title:   title,
			Message: message,
		}
		notification.Push()
	}()
	return nil
}
