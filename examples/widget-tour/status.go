package main

import "github.com/egoist/mygo/ui/widget"

// statusTour shows the status components: badges, an info bar with an
// action and a close button, and an empty state with an action.
func statusTour() widget.Control {
	result := widget.Label("Click the action or close the bar").Muted()
	return widget.VBox(12).Add(
		widget.FlowPanel(8).Add(
			widget.Badge("Live").SetStatus(widget.StatusSuccess),
			widget.Badge("Đang chạy").SetStatus(widget.StatusRunning).SetDetail("Bước 2/5"),
			widget.Badge("Cần xác minh").SetStatus(widget.StatusWarning),
			widget.Badge("Sai mật khẩu").SetStatus(widget.StatusDanger),
		),
		widget.InfoBar(widget.StatusWarning, "Cần xác minh", "Mở email đã gửi để xác minh tài khoản.").
			SetAction("Xác minh", func() { result.SetText("Verify clicked") }).
			SetClosable(true).OnClose(func() { result.SetText("Info bar closed") }),
		widget.EmptyState("Chưa có tài khoản", "Thêm một tài khoản để bắt đầu.").
			SetAction("Thêm", func() { result.SetText("Add clicked") }).SetSize(0, 220),
		result,
	)
}
