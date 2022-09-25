package client

import "github.com/comoyi/valheim-launcher/log"

var appName = "Launcher"
var versionText = "0.0.1"

func Start() {
	log.Debugf("Client start\n")

	initUI()

	w.ShowAndRun()
}
