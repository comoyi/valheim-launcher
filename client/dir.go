package client

import (
	"fmt"
	"github.com/comoyi/valheim-launcher/log"
	"os"
	"runtime"
)

func getPossibleDirs() []string {
	dirs := make([]string, 0)

	sysType := runtime.GOOS
	if sysType == "windows" {
		dirs = getWindowsPossibleDirs()
	} else {
		userHomeDir, err := os.UserHomeDir()
		if err != nil {
			log.Warnf("Get os.UserHomeDir failed, err: %v\n", err)
		} else {
			dirs = append(dirs, fmt.Sprintf("%s%s%s", userHomeDir, string(os.PathSeparator), "valheim"))
			dirs = append(dirs, fmt.Sprintf("%s%s%s", userHomeDir, string(os.PathSeparator), "Valheim"))
		}
	}

	return dirs
}

func getWindowsPossibleDirs() []string {
	dirs := make([]string, 0)
	ds := []string{
		`C:\Program Files (x86)\Steam\steamapps\common\Valheim`,
		`D:\Program Files (x86)\Steam\steamapps\common\Valheim`,
		`E:\Program Files (x86)\Steam\steamapps\common\Valheim`,
		`F:\Program Files (x86)\Steam\steamapps\common\Valheim`,
		`G:\Program Files (x86)\Steam\steamapps\common\Valheim`,
		`H:\Program Files (x86)\Steam\steamapps\common\Valheim`,
		`C:\Program Files\Steam\steamapps\common\Valheim`,
		`D:\Program Files\Steam\steamapps\common\Valheim`,
		`E:\Program Files\Steam\steamapps\common\Valheim`,
		`F:\Program Files\Steam\steamapps\common\Valheim`,
		`G:\Program Files\Steam\steamapps\common\Valheim`,
		`H:\Program Files\Steam\steamapps\common\Valheim`,
		`C:\SteamLibrary\steamapps\common\Valheim`,
		`D:\SteamLibrary\steamapps\common\Valheim`,
		`E:\SteamLibrary\steamapps\common\Valheim`,
		`F:\SteamLibrary\steamapps\common\Valheim`,
		`G:\SteamLibrary\steamapps\common\Valheim`,
		`H:\SteamLibrary\steamapps\common\Valheim`,
		`C:\Valheim`,
		`D:\Valheim`,
		`E:\Valheim`,
		`F:\Valheim`,
		`G:\Valheim`,
		`H:\Valheim`,
	}
	dirs = append(dirs, ds...)
	return dirs
}
