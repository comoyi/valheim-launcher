package config

import (
	"fmt"
	"github.com/comoyi/valheim-launcher/log"
	"github.com/spf13/viper"
	"os"
	"strings"
)

var Conf Config

type Config struct {
	DebugLevel string `toml:"debuglevel"`
	LogLevel   int
}

func LoadConfig() {
	var err error
	viper.SetConfigName("config")
	viper.SetConfigType("toml")
	viper.AddConfigPath(".")
	viper.AddConfigPath(fmt.Sprintf("%s%s%s", "$HOME", string(os.PathSeparator), ".valheim-launcher"))
	err = viper.ReadInConfig()
	if err != nil {
		log.Errorf("Read config failed, err: %v\n", err)
		return
	}

	debugLevel := strings.ToUpper(viper.GetString("debuglevel"))
	switch debugLevel {
	case "TRACE":
		viper.Set("LogLevel", log.Trace)
	case "DEBUG":
		viper.Set("LogLevel", log.Debug)
	case "INFO":
		viper.Set("LogLevel", log.Info)
	case "WARN":
		viper.Set("LogLevel", log.Warn)
	case "ERROR":
		viper.Set("LogLevel", log.Error)
	case "OFF":
		viper.Set("LogLevel", log.Off)
	default:
		viper.Set("LogLevel", log.Off)
	}

	err = viper.Unmarshal(&Conf)
	if err != nil {
		log.Errorf("Unmarshal config failed, err: %v\n", err)
		return
	}
	log.Debugf("config: %+v\n", Conf)
}
