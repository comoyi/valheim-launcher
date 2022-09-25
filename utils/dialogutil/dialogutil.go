package dialogutil

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

func ShowInformation(title, message string, parent fyne.Window) {
	NewInformation(title, message, parent).Show()
}

func NewInformation(title, message string, parent fyne.Window) dialog.Dialog {
	content := container.NewVBox()
	messageLabel := widget.NewLabel(message)
	content.Add(messageLabel)
	return dialog.NewCustom(title, "OK", content, parent)
}
