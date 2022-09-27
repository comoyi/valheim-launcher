package timeutil

import (
	"fmt"
	"time"
)

// GetCurrentDateTime 获取当前时刻DateTime 例：1609527845 -> 2021-01-02 03:04:05
func GetCurrentDateTime() string {
	return TimeToDateTime(time.Now())
}

// TimestampToDateTime 10位时间戳转换为DateTime 例：1609527845 -> 2021-01-02 03:04:05
func TimestampToDateTime(timestamp int64) string {
	return TimeToDateTime(time.Unix(timestamp, 0))
}

// TimeToDateTime time.Time转换为DateTime 例：time.Time -> 2021-01-02 03:04:05
func TimeToDateTime(time time.Time) string {
	return time.Format("2006-01-02 15:04:05")
}

func FormatDuration(second int64) string {
	var d int64
	var h int64
	var m int64
	var s int64
	var str string
	var flag = false

	d = second / 86400
	second -= d * 86400
	h = second / 3600
	second -= h * 3600
	m = second / 60
	second -= m * 60
	s = second

	if d > 0 {
		flag = true
		str = fmt.Sprintf("%s%d天", str, d)
	}
	if flag || h > 0 {
		flag = true
		str = fmt.Sprintf("%s%d时", str, h)
	}
	if flag || m > 0 {
		flag = true
		str = fmt.Sprintf("%s%d分", str, m)
	}
	if flag || s > 0 {
		flag = true
		str = fmt.Sprintf("%s%d秒", str, s)
	}
	return str
}
