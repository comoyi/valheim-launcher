package config

import (
	"github.com/comoyi/valheim-launcher/log"
	"github.com/comoyi/valheim-launcher/util/fsutil"
	"github.com/spf13/viper"
	"os"
	"path/filepath"
	"sync"
)

var Conf Config

type Config struct {
	LogLevel                    string            `toml:"log_level" mapstructure:"log_level"`
	Protocol                    string            `toml:"protocol" mapstructure:"protocol"`
	Host                        string            `toml:"host" mapstructure:"host"`
	Port                        int               `toml:"port" mapstructure:"port"`
	Dir                         string            `toml:"dir" mapstructure:"dir"`
	AnnouncementRefreshInterval int64             `toml:"announcement_refresh_interval" mapstructure:"announcement_refresh_interval"`
	IsUseCache                  bool              `toml:"is_use_cache" mapstructure:"is_use_cache"`
	CacheDir                    string            `toml:"cache_dir" mapstructure:"cache_dir"`
	DownloadServers             []*DownloadServer `toml:"download_server" mapstructure:"download_server"`
}

type DownloadServer struct {
	Protocol   string `toml:"protocol" mapstructure:"protocol"`
	Host       string `toml:"host" mapstructure:"host"`
	Port       int    `toml:"port" mapstructure:"port"`
	PrefixPath string `toml:"prefix_path" mapstructure:"prefix_path"`
	Type       int    `toml:"type" mapstructure:"type"`
}

func initDefaultConfig() {
	viper.SetDefault("log_level", log.Off)
	viper.SetDefault("protocol", "http")
	viper.SetDefault("host", "127.0.0.1")
	viper.SetDefault("port", 8080)
	viper.SetDefault("dir", "")
	viper.SetDefault("announcement_refresh_interval", 60)
	viper.SetDefault("is_use_cache", true)
	viper.SetDefault("cache_dir", ".cache")
}

func LoadConfig() {
	var err error

	viper.SetConfigName("config")
	viper.SetConfigType("toml")
	viper.AddConfigPath(".")

	configDirPath, err := getConfigDirPath()
	if err != nil {
		log.Warnf("Get configDirPath failed, err: %v\n", err)
		return
	}
	viper.AddConfigPath(configDirPath)

	initDefaultConfig()

	err = viper.ReadInConfig()
	if err != nil {
		log.Errorf("Read config failed, err: %v\n", err)
		//return
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

	configDirPath, err := getConfigDirPath()
	if err != nil {
		log.Warnf("Get configDirPath failed, err: %v\n", err)
		return err
	}

	configFile := filepath.Join(configDirPath, "config.toml")
	log.Debugf("configFile: %s\n", configFile)

	exist, err := fsutil.Exists(configDirPath)
	if err != nil {
		log.Warnf("Check isPathExist failed, err: %v\n", err)
		return err
	}
	if !exist {
		err = os.MkdirAll(configDirPath, os.FileMode(0o755))
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

func getConfigDirPath() (string, error) {
	configRootPath, err := getConfigRootPath()
	if err != nil {
		return "", err
	}

	configPath := filepath.Join(configRootPath, ".valheim-launcher")
	return configPath, nil
}

func getConfigRootPath() (string, error) {
	var err error
	configRootPath := ""
	configRootPath, err = os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return configRootPath, nil
}
