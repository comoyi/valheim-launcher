package client

import (
	"context"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/comoyi/valheim-launcher/config"
	"github.com/comoyi/valheim-launcher/log"
	"github.com/comoyi/valheim-launcher/theme"
	"github.com/comoyi/valheim-launcher/util/dialogutil"
	"github.com/comoyi/valheim-launcher/util/fsutil"
	"github.com/comoyi/valheim-launcher/util/timeutil"
	"github.com/spf13/viper"
	"os"
	"path/filepath"
	"time"
)

var w fyne.Window
var c *fyne.Container
var myApp fyne.App
var msgContainer = widget.NewLabel("")

func initUI() {
	initMainWindow()

	initMenu()
}

func initMainWindow() {
	windowTitle := fmt.Sprintf("%s-v%s", appName, versionText)

	myApp = app.NewWithID("com.comoyi.valheim-launcher")
	myApp.Settings().SetTheme(theme.CustomTheme)
	w = myApp.NewWindow(windowTitle)
	w.SetMaster()
	w.Resize(fyne.NewSize(800, 600))
	c = container.NewVBox()
	w.SetContent(c)

	pathLabel := widget.NewLabel("Valheim文件夹")
	pathInput := widget.NewLabel("")
	pathInput.SetText(config.Conf.Dir)

	var manualInputDialog dialog.Dialog
	inputBtnText := "手动输入文件夹地址"
	inputBtn := widget.NewButton(inputBtnText, func() {
		pathManualInput := widget.NewEntry()
		tipLabel := widget.NewLabel("")
		box := container.NewVBox(pathManualInput, tipLabel)
		manualInputDialog = dialog.NewCustomConfirm("请输入文件夹地址", "确定", "取消", box, func(b bool) {
			if b {
				if pathManualInput.Text == "" {
					tipLabel.SetText("请输入文件夹地址")
					manualInputDialog.Show()
					return
				}
				path := filepath.Clean(pathManualInput.Text)
				exists, err := fsutil.Exists(path)
				if err != nil {
					tipLabel.SetText("文件夹地址检测失败")
					manualInputDialog.Show()
					return
				}
				if !exists {
					tipLabel.SetText("该文件夹不存在")
					manualInputDialog.Show()
					return
				}
				f, err := os.Stat(path)
				if err != nil {
					tipLabel.SetText("文件夹地址检测失败[2]")
					manualInputDialog.Show()
					return
				}
				if !f.IsDir() {
					tipLabel.SetText("请输入正确的文件夹地址")
					manualInputDialog.Show()
					return
				}

				pathInput.SetText(path)
				saveDirConfig(path)
			}
		}, w)
		manualInputDialog.Resize(fyne.NewSize(700, 100))
		manualInputDialog.Show()
	})

	selectBtnText := "手动选择文件夹"
	selectBtn := widget.NewButton(selectBtnText, func() {
		dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil {
				log.Debugf("select folder failed, err: %v\n", err)
				return
			}
			if uri == nil {
				log.Debugf("select folder cancelled\n")
				return
			}
			path := uri.Path()
			path = filepath.Clean(path)
			dialog.NewCustomConfirm("提示", "确定", "取消", widget.NewLabel("选择这个文件夹吗？\n"+path), func(b bool) {
				if b {
					pathInput.SetText(path)
					saveDirConfig(path)
				}
			}, w).Show()
		}, w)
	})

	autoBtnText := "自动查找文件夹"
	autoBtn := widget.NewButton(autoBtnText, func() {
		isFound := false
		dirs := getPossibleDirs()
		for _, dir := range dirs {
			log.Debugf("check dir, %v\n", dir)
			exists, err := fsutil.Exists(dir)
			if err != nil {
				log.Debugf("skip this dir, dir: %v, err: %v\n", dir, err)
				continue
			}
			if exists {
				isFound = true
				pathInput.SetText(dir)
				saveDirConfig(dir)
				log.Debugf("found dir, %v\n", dir)
				addMsgWithTime("找到文件夹")
				break
			}
		}
		if !isFound {
			dialogutil.ShowInformation("", "未找到相关文件夹，请手动选择", w)
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
		baseDir = filepath.Clean(baseDir)

		if isUpdating {
			addMsgWithTime("取消更新")
			isUpdating = false
			updateBtn.SetText("更新")
			cancel()
			return
		}

		addMsgWithTime("开始更新")
		isUpdating = true
		updateBtn.SetText("取消更新")

		startTime := time.Now().Unix()

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
					err := update(ctx, baseDir, progressChan)
					if err != nil {
						dialogutil.ShowInformation("提示", "更新失败", w)
						addMsgWithTime("更新失败")
						log.Debugf("update failed, err: %v\n", err)
					} else {
						if isUpdating {
							endTime := time.Now().Unix()
							duration := endTime - startTime
							addMsgWithTime(fmt.Sprintf("更新完成，耗时：%s", timeutil.FormatDuration(duration)))
						}
					}

					// refresh progress bar
					refreshProgressbar(progressBar)

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
						refreshProgressbar(progressBar)
					}
				}
			}
		}(ctx)

	})

	c.Add(pathLabel)
	c2 := container.NewAdaptiveGrid(3)
	c2.Add(inputBtn)
	c2.Add(selectBtn)
	c2.Add(autoBtn)
	c.Add(c2)
	c3 := container.NewAdaptiveGrid(1)
	c3.Add(pathInput)
	c.Add(c3)
	c4 := container.NewAdaptiveGrid(1)
	c4.Add(updateBtn)
	c.Add(c4)
	c5 := container.NewAdaptiveGrid(1)
	c5.Add(progressBar)
	c.Add(c5)

	initAnnouncement(c)
	initMsgContainer(c)
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

func initAnnouncement(c *fyne.Container) {
	var announcementContainer = widget.NewLabel("")
	announcementBox := container.NewVBox()
	announcementLabel := widget.NewLabel("公告")
	announcementContainerScroll := container.NewScroll(announcementContainer)
	announcementContainerScroll.SetMinSize(fyne.NewSize(800, 100))
	announcementBox.Hide()
	announcementBox.Add(announcementLabel)
	announcementBox.Add(announcementContainerScroll)
	c.Add(announcementBox)

	go func() {
		refreshAnnouncement(announcementContainer, announcementBox)
		for {
			select {
			case <-time.After(10 * time.Second):
				refreshAnnouncement(announcementContainer, announcementBox)
			}
		}
	}()
}

func initMsgContainer(c *fyne.Container) {
	msgBox := container.NewVBox()
	msgContainerScroll := container.NewScroll(msgContainer)
	msgContainerScroll.SetMinSize(fyne.NewSize(800, 200))
	msgBox.Add(msgContainerScroll)
	c.Add(msgBox)
}

func refreshProgressbar(progressBar *widget.ProgressBar) {
	if progressBar.Value == float64(UpdateInf.Current) && progressBar.Max == float64(UpdateInf.Total) {
		return
	} else {
		progressBar.Value = float64(UpdateInf.Current)
		progressBar.Max = float64(UpdateInf.Total)
		progressBar.Refresh()
	}
}

func addMsgWithTime(msg string) {
	msg = fmt.Sprintf("%s %s", timeutil.GetCurrentDateTime(), msg)
	addMsg(msg)
}

func addMsg(msg string) {
	msgContainer.SetText(msg + "\n" + msgContainer.Text)
}

func saveDirConfig(path string) {
	viper.Set("dir", path)
	err := config.SaveConfig()
	if err != nil {
		log.Debugf("save config failed, err: %+v\n", err)
		return
	}
}
