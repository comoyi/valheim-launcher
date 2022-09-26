package client

import (
	"fmt"
	"github.com/comoyi/valheim-launcher/log"
	"os"
	"runtime"
)

var appName = "Valheim Launcher"
var versionText = "0.0.1"

func Start() {
	log.Debugf("Client start\n")

	initUI()

	w.ShowAndRun()
}

func GetDirs() []string {
	dirs := make([]string, 0)

	sysType := runtime.GOOS
	if sysType == "windows" {
		dirs = append(dirs, "C:\\Program Files (x86)\\Steam\\steamapps\\common\\Valheim")
		dirs = append(dirs, "D:\\Program Files (x86)\\Steam\\steamapps\\common\\Valheim")
		dirs = append(dirs, "E:\\Program Files (x86)\\Steam\\steamapps\\common\\Valheim")
		dirs = append(dirs, "F:\\Program Files (x86)\\Steam\\steamapps\\common\\Valheim")
		dirs = append(dirs, "C:\\Program Files\\Steam\\steamapps\\common\\Valheim")
		dirs = append(dirs, "D:\\Program Files\\Steam\\steamapps\\common\\Valheim")
		dirs = append(dirs, "E:\\Program Files\\Steam\\steamapps\\common\\Valheim")
		dirs = append(dirs, "F:\\Program Files\\Steam\\steamapps\\common\\Valheim")
		dirs = append(dirs, "C:\\Valheim")
		dirs = append(dirs, "D:\\Valheim")
	} else {
		userHomeDir, err := os.UserHomeDir()
		if err != nil {
			log.Warnf("Get os.UserHomeDir failed, err: %v\n", err)
		} else {
			log.Debugf("userHomeDir: %s\n", userHomeDir)
			path := fmt.Sprintf("%s%s%s", userHomeDir, string(os.PathSeparator), "valheim")
			dirs = append(dirs, path)
		}
	}

	return dirs
}
