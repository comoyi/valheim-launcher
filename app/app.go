package app

import (
	"github.com/comoyi/valheim-launcher/client"
	"github.com/comoyi/valheim-launcher/config"
)

func Start() {
	config.LoadConfig()
	client.Start()
}
