package client

import (
	"context"
	"errors"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	theme2 "fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/comoyi/valheim-launcher/config"
	"github.com/comoyi/valheim-launcher/log"
	"github.com/comoyi/valheim-launcher/theme"
	"github.com/comoyi/valheim-launcher/util/dialogutil"
	"github.com/comoyi/valheim-launcher/util/fsutil"
	"github.com/comoyi/valheim-launcher/util/timeutil"
	"github.com/spf13/viper"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

var w fyne.Window
var c *fyne.Container
var myApp fyne.App
var msgContainer = widget.NewLabel("")

func initUI() {
	initMainWindow()

	initMenu()

	go func() {
		autoCleanMsg()
	}()
}

func initMainWindow() {
	windowTitle := fmt.Sprintf("%s-v%s", appName, versionText)

	myApp = app.NewWithID("com.comoyi.valheim-launcher")
	myApp.Settings().SetTheme(theme.CustomTheme)
	w = myApp.NewWindow(windowTitle)
	w.SetMaster()
	w.Resize(fyne.NewSize(800, 800))
	c = container.NewVBox()
	w.SetContent(c)

	useStepLabel := widget.NewLabel("【使用步骤】第一步：选文件夹，第二步：更新MOD，第三步：启动英灵神殿\n【注意】更新MOD前请先关闭英灵神殿")
	pathLabel := widget.NewLabel("英灵神殿所在文件夹，以下3种方式任选一种，推荐自动查找")
	pathInput := widget.NewLabel("")
	pathInput.SetText(config.Conf.Dir)

	selectBtnText := "选择文件夹"
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
	selectBtn.SetIcon(theme2.FolderIcon())

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
	autoBtn.SetIcon(theme2.SearchIcon())

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
	updateBtnText := "更新MOD"
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
			updateBtn.SetText(updateBtnText)
			cancel()
			return
		}

		addMsgWithTime("开始更新")
		isUpdating = true
		updateBtn.SetText("取消更新")

		startTime := time.Now().Unix()

		progressBar.SetValue(0)
		progressBar.Show()

		progressChan := make(chan struct{}, 100)

		var ctx context.Context
		ctx, cancel = context.WithCancel(ctxParent)

		go func(ctx context.Context) {
			var err error
			maxTimes := 3
			triedTimes := 0
		bf:
			for {
				select {
				case <-ctx.Done():
					return
				default:
					if triedTimes >= maxTimes {
						log.Debugf("reach max retry times, triedTimes: %d, maxTimes: %d\n", triedTimes, maxTimes)
						break bf
					}
					triedTimes++
					err = update(ctx, baseDir, progressChan)
					if err != nil {
						if !errors.Is(err, errServerScanning) {
							log.Debugf("not errServerScanning\n")
							break bf
						} else {
							addMsgWithTime("服务器正在刷新文件列表，等待重试...")
							select {
							case <-ctx.Done():
								return
							case <-time.After(3 * time.Second):

							}
						}
					} else {
						break bf
					}
				}
			}
			if err != nil {
				if isUpdating {
					dialogutil.ShowInformation("提示", "更新失败", w)
					addMsgWithTime("更新失败")
					log.Debugf("update failed, err: %v\n", err)
				}
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
			updateBtn.SetText(updateBtnText)
			cancel()
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
	updateBtn.SetIcon(theme2.ViewRefreshIcon())

	c.Add(useStepLabel)
	c.Add(pathLabel)
	c2 := container.NewAdaptiveGrid(3)
	initManualInputBtn(c2, pathInput)
	c2.Add(selectBtn)
	c2.Add(autoBtn)
	c.Add(c2)
	c3 := container.NewAdaptiveGrid(1)
	c3.Add(pathInput)
	c.Add(c3)
	startBtn := initStartBtn()
	c4 := container.NewAdaptiveGrid(2)
	c4.Add(updateBtn)
	c4.Add(startBtn)
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

func initManualInputBtn(c *fyne.Container, pathInput *widget.Label) {
	var manualInputDialog dialog.Dialog
	inputBtnText := "手动输入文件夹地址"
	inputBtn := widget.NewButton(inputBtnText, func() {
		manualPathInput := widget.NewEntry()
		tipLabel := widget.NewLabel("")
		box := container.NewVBox(manualPathInput, tipLabel)
		manualInputDialog = dialog.NewCustomConfirm("请输入文件夹地址", "确定", "取消", box, func(b bool) {
			if b {
				if manualPathInput.Text == "" {
					tipLabel.SetText("请输入文件夹地址")
					manualInputDialog.Show()
					return
				}
				path := filepath.Clean(manualPathInput.Text)
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
	inputBtn.SetIcon(theme2.DocumentCreateIcon())

	c.Add(inputBtn)
}

func initStartBtn() *widget.Button {
	var btn *widget.Button
	btn = widget.NewButton("启动英灵神殿", func() {
		if runtime.GOOS != "windows" {
			dialogutil.ShowInformation("", "当前只支持Windows", w)
			return
		}
		cmd := exec.Command("cmd", "/C", "start", "/B", "steam://rungameid/892970")
		err := cmd.Start()
		if err != nil {
			log.Infof("Start failed, err: %v\n", err)
			addMsgWithTime("启动失败，请通过其他方式启动")
			return
		}
	})
	btn.SetIcon(theme2.MediaPlayIcon())
	return btn
}

func initAnnouncement(c *fyne.Container) {
	var announcementContainer = widget.NewLabel("")
	announcementBox := container.NewVBox()
	announcementLabel := widget.NewLabel("公告")
	announcementContainerScroll := container.NewScroll(announcementContainer)
	announcementContainerScroll.SetMinSize(fyne.NewSize(800, 150))
	announcementBox.Hide()
	announcementBox.Add(announcementLabel)
	announcementBox.Add(announcementContainerScroll)
	c.Add(announcementBox)

	go func() {
		refreshAnnouncement(announcementContainer, announcementBox, c)
		interval := config.Conf.AnnouncementRefreshInterval
		if interval > 0 {
			for {
				select {
				case <-time.After(time.Duration(interval) * time.Second):
					refreshAnnouncement(announcementContainer, announcementBox, c)
				}
			}
		}
	}()
}

func initMsgContainer(c *fyne.Container) {
	msgBox := container.NewVBox()
	msgContainerScroll := container.NewScroll(msgContainer)
	msgContainerScroll.SetMinSize(fyne.NewSize(800, 150))
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

func autoCleanMsg() {
	max := 10000
	for {
		<-time.After(300 * time.Second)
		if len([]rune(msgContainer.Text)) > max {
			msgContainer.Text = string([]rune(msgContainer.Text)[:max])
			msgContainer.Refresh()
		}
	}
}
