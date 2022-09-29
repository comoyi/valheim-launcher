package client

import (
	"github.com/comoyi/valheim-launcher/log"
)

var appName = "Valheim Launcher"
var versionText = "1.0.1"

func Start() {
	log.Debugf("Client start\n")

	initUI()

	w.ShowAndRun()
}
