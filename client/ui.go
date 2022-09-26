package client

import (
	"context"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/comoyi/valheim-launcher/theme"
	"github.com/comoyi/valheim-launcher/utils/dialogutil"
	"time"
)

var w fyne.Window
var c *fyne.Container
var myApp fyne.App

func initUI() {
	initMainWindow()
}

func initMainWindow() {
	windowTitle := fmt.Sprintf("%s-v%s", appName, versionText)

	myApp = app.New()
	myApp.Settings().SetTheme(theme.CustomTheme)
	w = myApp.NewWindow(windowTitle)
	w.SetMaster()
	w.Resize(fyne.NewSize(800, 400))
	c = container.NewVBox()
	w.SetContent(c)

	pathLabel := widget.NewLabel("文件夹")
	pathInput := widget.NewEntry()
	pathInput.Disable()

	selectBtnText := "手动选择文件夹"
	selectBtn := widget.NewButton(selectBtnText, func() {
		dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
			path := uri.Path()
			dialog.NewCustomConfirm("提示", "确定", "取消", widget.NewLabel("选择这个文件夹吗？\n"+path), func(b bool) {
				if b {
					pathInput.SetText(path)
				}
			}, w).Show()
		}, w)
	})

	autoBtnText := "自动查找文件夹"
	autoBtn := widget.NewButton(autoBtnText, func() {
		dialogutil.ShowInformation("提示", "开发中，敬请期待", w)
	})

	progressBar := widget.NewProgressBar()
	progressBar.Hide()
	progressBarFormatter := func() string {
		return fmt.Sprintf("%v / %v", progressBar.Value, progressBar.Max)
	}
	progressBar.TextFormatter = progressBarFormatter

	var updateBtn *widget.Button

	ctxParent := context.Background()
	var cancel context.CancelFunc
	isUpdating := false
	updateBtnText := "更新"
	updateBtn = widget.NewButton(updateBtnText, func() {
		baseDir := pathInput.Text
		if baseDir == "" {
			dialogutil.ShowInformation("提示", "请选择文件夹", w)
			return
		}
		if isUpdating {
			isUpdating = false
			updateBtn.SetText("更新")
			cancel()
			return
		}

		isUpdating = true
		updateBtn.SetText("取消更新")

		progressBar.SetValue(0)
		progressBar.Show()

		var ctx context.Context
		ctx, cancel = context.WithCancel(ctxParent)

		go func(ctx context.Context) {
			for {
				select {
				case <-ctx.Done():
					return
				default:
					update(ctx, baseDir)

					// delay for progress bar
					<-time.After(200 * time.Millisecond)

					isUpdating = false
					updateBtn.SetText("更新")
					cancel()
					return
				}
			}
		}(ctx)
		go func(ctx context.Context) {
			for {
				select {
				case <-time.After(100 * time.Millisecond):
					//progressBar.SetValue(UpdateInf.GetRatio())
					progressBar.Value = float64(UpdateInf.Current)
					progressBar.Max = float64(UpdateInf.Total)
					progressBar.Refresh()
				case <-ctx.Done():
					return
				}
			}
		}(ctx)

	})

	c.Add(pathLabel)
	c2 := container.NewAdaptiveGrid(1)
	c2.Add(pathInput)
	c.Add(c2)
	c3 := container.NewAdaptiveGrid(2)
	c3.Add(selectBtn)
	c3.Add(autoBtn)
	c.Add(c3)
	c4 := container.NewAdaptiveGrid(1)
	c4.Add(progressBar)
	c.Add(c4)
	c5 := container.NewAdaptiveGrid(1)
	c5.Add(updateBtn)
	c.Add(c5)
}
