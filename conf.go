package mylog

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// 日志配置,可以为空
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
	//在控制台输出shortfile
	ShowShortFileInConsole bool
	//在控制台输出func
	ShowFuncInConsole bool
	//按大小分割日志,单位byte。(不能和按日期分割同时使用)
	MaxLogSize int64
	// 日志最大保留天数(设置后请不要在日志文件夹中放置其他文件，否则可能被删除)
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
}

// 在每条log末尾添加key-value
func (c *LogConfig) SetKeyValue(key string, value interface{}) {
	c.key = key
	c.value = value
}

type logHook struct {
	ErrWriter   *os.File
	OtherWriter *os.File
	//修改Writer时加锁
	WriterLock *sync.RWMutex
	LogConfig  LogConfig
	// 2006_01_02
	FileDate string
	// byte,仅在SizeSplit>0时有效
	LogSize int64
	// 2006_01_02
	dateFmt string
	// 2006_01_02_030405(按大小分割时使用)
	dateFmt2 string
}

// 默认 --loglevel=info
func InitGlobalLogger(config LogConfig) error {
	return initlLog(logrus.StandardLogger(), config)
}

// 默认 --loglevel=info
func NewLogger(config LogConfig) (*logrus.Logger, error) {
	logger := logrus.New()
	err := initlLog(logger, config)
	if err != nil {
		return nil, err
	}
	return logger, nil
}

func initlLog(logger *logrus.Logger, config LogConfig) error {

	var level logrus.Level = PraseLevel(config.LogLevel)
	//fmt.Println("level:", level)

	logger.SetReportCaller(true) //开启调用者信息
	logger.SetLevel(level)       //设置最低的Level
	formatter := &TextFormatter{
		TimestampFormat: "2006-01-02 15:04:05", //时间格式
		FullTimestamp:   true,                  //开启时间戳
		ForceColors:     true,                  //开启颜色
		NoConsole:       config.NoConsole,
		// CallerPrettyfier: func(f *runtime.Frame) (string, string) {
		// 	//返回shortfile,funcname,linenum
		// 	//main.go:main:12
		// 	shortFile := f.File
		// 	if strings.Contains(f.File, "/") {
		// 		shortFile = f.File[strings.LastIndex(f.File, "/")+1:]
		// 	}
		// 	return "", fmt.Sprintf("%s:%s():%d:", shortFile, f.Function, f.Line)
		// },
		CallerPrettyfier: func(f *runtime.Frame) (string, string) {
			return "", ""
		},
	}

	if config.NoTimestamp {
		formatter.DisableTimestamp = true
	}
	logger.SetFormatter(formatter)

	if config.NoConsole {
		logger.SetOutput(io.Discard)
	}

	if config.LogExt == "" {
		config.LogExt = ".log"
	}
	if config.LogExt[0] != '.' {
		config.LogExt = "." + config.LogExt
	}
	if config.TimeLocation == nil {
		config.TimeLocation = time.Local
	}
	if config.DefaultLogName == "" {
		config.DefaultLogName = "DEFAULT"
	}

	hook := &logHook{}
	hook.dateFmt = "2006_01_02"
	hook.dateFmt2 = "2006_01_02_030405"
	hook.FileDate = time.Now().In(config.TimeLocation).Format(hook.dateFmt)
	hook.LogSize = 0
	hook.WriterLock = &sync.RWMutex{}
	hook.LogConfig = config

	//添加hook
	logger.AddHook(hook)

	err := hook.updateNewLogPathAndFile()
	if err != nil {
		return fmt.Errorf("updateNewLogPathAndFile err:%v", err)
	}
	if config.MaxKeepDays > 0 && !oldLogCheckerOnline {
		oldLogCheckerOnline = true
		go hook.deleteOldLog()
	}
	return nil
}
