package client

import (
	"context"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/comoyi/valheim-launcher/log"
	"github.com/comoyi/valheim-launcher/theme"
	"github.com/comoyi/valheim-launcher/utils/dialogutil"
	"github.com/comoyi/valheim-launcher/utils/fileutil"
	"time"
)

var w fyne.Window
var c *fyne.Container
var myApp fyne.App

func initUI() {
	initMainWindow()

	initMenu()
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
		dirs := GetDirs()
		for _, dir := range dirs {
			log.Debugf("check dir, %v\n", dir)
			exists, err := fileutil.Exists(dir)
			if err != nil {
				log.Debugf("skip this dir, dir: %v, err: %v\n", dir, err)
				continue
			}
			if exists {
				log.Debugf("dir exist, %v\n", dir)
				pathInput.SetText(dir)
				break
			} else {
				log.Debugf("dir not exist, %v\n", dir)
			}
		}
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

		progressChan := make(chan struct{}, 10)

		var ctx context.Context
		ctx, cancel = context.WithCancel(ctxParent)

		go func(ctx context.Context) {
			for {
				select {
				case <-ctx.Done():
					return
				default:
					update(ctx, baseDir, progressChan)

					// delay for progress bar
					<-time.After(50 * time.Millisecond)

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
				case <-ctx.Done():
					return
				case <-progressChan:
					select {
					case <-ctx.Done():
						return
					default:
						if progressBar.Value == float64(UpdateInf.Current) && progressBar.Max == float64(UpdateInf.Total) {
							break
						} else {
							//progressBar.SetValue(UpdateInf.GetRatio())
							progressBar.Value = float64(UpdateInf.Current)
							progressBar.Max = float64(UpdateInf.Total)
							progressBar.Refresh()
						}
					}
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

func initMenu() {
	firstMenu := fyne.NewMenu("操作")
	helpMenuItem := fyne.NewMenuItem("关于", func() {
		content := container.NewVBox()
		appInfo := widget.NewLabel(appName)
		content.Add(appInfo)
		versionInfo := widget.NewLabel(fmt.Sprintf("Version %v", versionText))
		content.Add(versionInfo)

		h := container.NewHBox()

		authorInfo := widget.NewLabel("Copyright © 2022 清新池塘")
		h.Add(authorInfo)
		linkInfo := widget.NewHyperlink(" ", nil)
		_ = linkInfo.SetURLFromString("https://github.com/comoyi/valheim-launcher")
		h.Add(linkInfo)
		content.Add(h)
		dialog.NewCustom("关于", "关闭", content, w).Show()
	})
	helpMenu := fyne.NewMenu("帮助", helpMenuItem)
	mainMenu := fyne.NewMainMenu(firstMenu, helpMenu)
	w.SetMainMenu(mainMenu)
}
