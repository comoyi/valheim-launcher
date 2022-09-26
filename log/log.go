package log

import (
	"fmt"
	"github.com/spf13/viper"
	"io/fs"
	"os"
)

const (
	Trace int = 100
	Debug int = 200
	Info  int = 300
	Warn  int = 400
	Error int = 500
	Off   int = 900
)

func Tracef(format string, args ...interface{}) {
	if viper.GetInt("LogLevel") > Trace {
		return
	}
	s := fmt.Sprintf("[TRACE]"+format, args...)
	w(s)
}

func Debugf(format string, args ...interface{}) {
	if viper.GetInt("LogLevel") > Debug {
		return
	}
	s := fmt.Sprintf("[DEBUG]"+format, args...)
	w(s)
}

func Infof(format string, args ...interface{}) {
	if viper.GetInt("LogLevel") > Info {
		return
	}
	s := fmt.Sprintf("[INFO] "+format, args...)
	w(s)
}

func Warnf(format string, args ...interface{}) {
	if viper.GetInt("LogLevel") > Warn {
		return
	}
	s := fmt.Sprintf("[WARN] "+format, args...)
	w(s)
}

func Errorf(format string, args ...interface{}) {
	if viper.GetInt("LogLevel") > Error {
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
