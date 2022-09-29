package client

import (
	"github.com/comoyi/valheim-launcher/log"
)

var appName = "Valheim Launcher"

func Start() {
	log.Debugf("Client start\n")

	initUI()

	w.ShowAndRun()
}
