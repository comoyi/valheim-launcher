package config

import (
	"fmt"
	"github.com/comoyi/valheim-launcher/log"
	"github.com/comoyi/valheim-launcher/utils/fsutil"
	"github.com/spf13/viper"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var Conf Config

type Config struct {
	DebugLevel string `toml:"debuglevel"`
	LogLevel   int
	Protocol   string `toml:"protocol"`
	Host       string `toml:"host"`
	Port       int    `toml:"port"`
	Dir        string `toml:"dir"`
}

func initDefaultConfig() {
	viper.SetDefault("debuglevel", "OFF")
	viper.SetDefault("LogLevel", log.Off)
	viper.SetDefault("protocol", "http")
	viper.SetDefault("host", "127.0.0.1")
	viper.SetDefault("port", 8080)
	viper.SetDefault("dir", "")
}

func LoadConfig() {
	var err error

	viper.SetConfigName("config")
	viper.SetConfigType("toml")
	viper.AddConfigPath(".")
	//viper.AddConfigPath("config")
	viper.AddConfigPath(fmt.Sprintf("%s%s%s", "$HOME", string(os.PathSeparator), ".valheim-launcher"))

	initDefaultConfig()

	err = viper.ReadInConfig()
	if err != nil {
		log.Errorf("Read config failed, err: %v\n", err)
		//return
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

var saveMutex = &sync.Mutex{}

func SaveConfig() error {
	saveMutex.Lock()
	defer saveMutex.Unlock()

	err := viper.WriteConfig()
	if err == nil {
		return nil
	}

	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		log.Warnf("Get os.UserHomeDir failed, err: %v\n", err)
		return err
	}
	log.Debugf("userHomeDir: %s\n", userHomeDir)

	configPath := filepath.Join(userHomeDir, ".valheim-launcher")
	configFile := filepath.Join(configPath, "config.toml")
	log.Debugf("configFile: %s\n", configFile)

	exist, err := fsutil.Exists(configPath)
	if err != nil {
		log.Warnf("Check isPathExist failed, err: %v\n", err)
		return err
	}
	if !exist {
		err = os.MkdirAll(configPath, os.ModePerm)
		if err != nil {
			log.Warnf("Get os.MkdirAll failed, err: %v\n", err)
			return err
		}
	}

	err = viper.WriteConfigAs(configFile)
	if err != nil {
		log.Errorf("WriteConfigAs failed, err: %v\n", err)
		return err
	}
	return nil
}
