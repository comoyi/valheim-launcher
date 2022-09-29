package log

import (
	"fmt"
	"github.com/spf13/viper"
	"io/fs"
	"os"
)

const (
	Trace string = "TRACE"
	Debug string = "DEBUG"
	Info  string = "INFO"
	Warn  string = "WARN"
	Error string = "ERROR"
	Off   string = "OFF"
)

var logLevelMap = map[string]int{
	Trace: 100,
	Debug: 200,
	Info:  300,
	Warn:  400,
	Error: 500,
	Off:   900,
}

func Tracef(format string, args ...interface{}) {
	if logLevelMap[viper.GetString("log_level")] > logLevelMap[Trace] {
		return
	}
	s := fmt.Sprintf("[TRACE]"+format, args...)
	w(s)
}

func Debugf(format string, args ...interface{}) {
	if logLevelMap[viper.GetString("log_level")] > logLevelMap[Debug] {
		return
	}
	s := fmt.Sprintf("[DEBUG]"+format, args...)
	w(s)
}

func Infof(format string, args ...interface{}) {
	if logLevelMap[viper.GetString("log_level")] > logLevelMap[Info] {
		return
	}
	s := fmt.Sprintf("[INFO] "+format, args...)
	w(s)
}

func Warnf(format string, args ...interface{}) {
	if logLevelMap[viper.GetString("log_level")] > logLevelMap[Warn] {
		return
	}
	s := fmt.Sprintf("[WARN] "+format, args...)
	w(s)
}

func Errorf(format string, args ...interface{}) {
	if logLevelMap[viper.GetString("log_level")] > logLevelMap[Error] {
		return
	}
	s := fmt.Sprintf("[ERROR]"+format, args...)
	w(s)
}

func w(s string) {
	file, err := os.OpenFile("log.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, fs.ModePerm)
	if err != nil {
		return
	}
	defer file.Close()
	_, err = file.WriteString(s)
	if err != nil {
		return
	}
}
