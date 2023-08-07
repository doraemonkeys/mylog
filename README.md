# mylog

一个基于logrus的日志库，实现了各种定制化配置。



## 配置预览

```go
type LogConfig struct {
	//日志路径(可以为空)
	LogPath string
	//日志文件名后缀
	LogFileNameSuffix string
	//默认日志文件名(若按日期或大小分割日志，此项无效)
	DefaultLogName string
	//是否分离错误日志(Error级别以上)
	ErrSeparate bool
	//如果分离错误日志，普通日志文件是否仍然包含错误日志
	ErrInNormal bool
	//按日期分割日志(不能和按大小分割同时使用)
	DateSplit bool
	//取消日志输出到文件
	LogFileDisable bool
	//取消日志输出到控制台
	NoConsole bool
	//取消时间戳Timestamp
	NoTimestamp bool
	// 时间戳格式，默认 2006-01-02 15:04:05.000
	TimestampFormat string
	//在控制台输出shortfile
	ShowShortFileInConsole bool
	//在控制台输出func
	ShowFuncInConsole bool
	// 关闭调用者信息
	DisableCaller bool
	// 禁用写缓冲
	DisableWriterBuffer bool
	//按大小分割日志,单位byte。(不能和按日期分割同时使用)
	MaxLogSize int64
	// 日志最大保留天数，设置后请不要在日志文件夹中放置其他文件，否则可能被删除。
	// 开启此功能后，如果没有设置日志文件夹路径，则默认为DefaultSavePath。
	MaxKeepDays int
	//日志扩展名(默认.log)
	LogExt string
	//panic,fatal,error,warn,info,debug,trace
	LogLevel string
	//时区
	TimeLocation *time.Location
	//在每条log末尾添加key-value
	key string
	//在每条log末尾添加key-value
	value interface{}
	// 标记不被删除的日志文件名需要含有的后缀
	keepSuffix string
}
```



## QuickStart

```
go get -u github.com/doraemonkeys/mylog
```



```go
package main

import (
	"github.com/doraemonkeys/mylog"
	"github.com/sirupsen/logrus"
)

func init() {
	config := mylog.LogConfig{
		LogPath:     `./test_log`,
		LogLevel:    "trace",
		ErrSeparate: true, //错误日志是否单独输出到文件
		DateSplit:   true, //是否按日期分割日志
		MaxKeepDays: 1,
	}
	config.SetKeyValue("server", "[DEBUG]")
	err := mylog.InitGlobalLogger(config)
	if err != nil {
		panic(err)
	}
}

func main() {
	logrus.Trace("trace")
	logrus.Debug("debug")
	logrus.Info("info")
	logrus.Warn("warn")
	logrus.Error("error")
}
```



