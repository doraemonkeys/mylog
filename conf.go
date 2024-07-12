package mylog

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/doraemonkeys/doraemon"
	myformatter "github.com/doraemonkeys/mylog/formatter"
	"github.com/sirupsen/logrus"
)

// 默认日志文件夹路径。
//
// 开启日志最大保留天数后，如果没有设置日志文件夹路径，则默认为此路径。
const DefaultSavePath = "./logs"

// 日志级别
const (
	PanicLevel = "panic"
	FatalLevel = "fatal"
	ErrorLevel = "error"
	WarnLevel  = "warn"
	InfoLevel  = "info"
	DebugLevel = "debug"
	TraceLevel = "trace"
)

// 日志配置,可以为空
type LogConfig struct {
	//日志保存文件夹路径(可以为空)
	LogDir string
	//日志文件名后缀
	LogFileNameSuffix string
	//默认日志文件名(若按日期或大小分割日志，此项无效)
	DefaultLogName string
	//是否分离错误日志(Error级别以上)
	ErrSeparate bool
	//使普通日志文件不包含错误日志(分离错误日志时有效)
	ErrNotInNormal bool
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
	// 写缓冲大小，默认4096字节
	WriterBufferSize int
	// 以json格式输出
	JSONFormat bool
	// 禁用颜色
	DisableColors bool
	// Disables the truncation of the level text to 4 characters.
	DisableLevelTruncation bool
	// PadLevelText Adds padding the level text so that all the levels
	// output at the same length PadLevelText is a superset of the DisableLevelTruncation option
	PadLevelText bool
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

// 在每条log末尾添加key-value
func (c *LogConfig) SetKeyValue(key string, value interface{}) {
	c.key = key
	c.value = value
}

type logHook struct {
	// 写入文件的操作是线程安全的
	ErrWriter *lazyFileWriter
	// 写入文件的操作是线程安全的
	OtherWriter *os.File
	// bufio 并发不安全，只在一个goroutine中写入
	OtherBufWriter *bufio.Writer
	// 默认4096
	WriterBufferSize int
	// 写入ErrWriter、OtherWriter、OtherBufWriter 加读锁防止被修改为nil或close(因为暂且认为写入文件的操作是线程安全的)。
	WriterLock *sync.RWMutex

	// LastBufferWroteTime time.Time

	bufferQueue *doraemon.SimpleMQ[[]byte]
	LogConfig   LogConfig
	// 2006_01_02
	FileDate string
	// byte,仅在SizeSplit>0时有效
	LogSize int64
	// 2006_01_02
	dateFmt string
	// 2006_01_02_150405(按大小分割时使用)
	dateFmt2 string
}

// InitGlobalLogger初始化全局logger。
//
// 如果存在多个logger，调用者应该保证不同logger的日志文件路径不同。
// 默认的loglevel为info。
func InitGlobalLogger(config LogConfig) error {
	return initlLog(logrus.StandardLogger(), config)
}

// NewLogger返回一个新的logger。
//
// 如果存在多个logger，调用者应该保证不同logger的日志文件路径不同。
// 默认的loglevel为info。
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

	if !config.DisableCaller {
		logger.SetReportCaller(true) //开启调用者信息
	}

	logger.SetLevel(level) //设置最低的Level

	if config.TimestampFormat == "" {
		config.TimestampFormat = "2006-01-02 15:04:05.000"
	}

	// if config.NoTimestamp {
	// 	formatter.DisableTimestamp = true
	// }

	var formatter logrus.Formatter
	if config.JSONFormat {
		formatter = &logrus.JSONFormatter{
			TimestampFormat:  config.TimestampFormat, //时间戳格式
			DisableTimestamp: config.NoTimestamp,     //开启时间戳
			CallerPrettyfier: func(f *runtime.Frame) (string, string) {
				return "", ""
			},
		}
	} else {
		formatter = &myformatter.TextFormatter{
			TimestampFormat:        config.TimestampFormat, //时间戳格式
			FullTimestamp:          true,
			DisableTimestamp:       config.NoTimestamp,    //开启时间戳
			ForceColors:            !config.DisableColors, //开启颜色
			DisableLevelTruncation: config.DisableLevelTruncation,
			PadLevelText:           config.PadLevelText,
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
		config.DefaultLogName = "default"
	}
	if config.MaxKeepDays > 0 && config.LogDir == "" {
		config.LogDir = DefaultSavePath
	}
	config.keepSuffix = "keep"

	hook := &logHook{}
	hook.bufferQueue = doraemon.NewSimpleMQ[[]byte](10)
	hook.dateFmt = "2006_01_02"
	hook.dateFmt2 = "2006_01_02_150405"
	hook.FileDate = time.Now().In(config.TimeLocation).Format(hook.dateFmt)
	hook.LogSize = 0
	hook.WriterLock = &sync.RWMutex{}
	hook.LogConfig = config
	hook.WriterBufferSize = config.WriterBufferSize
	if hook.WriterBufferSize <= 0 {
		hook.WriterBufferSize = 4096
	}

	//添加hook
	logger.AddHook(hook)

	err := hook.updateNewLogPathAndFile()
	if err != nil {
		return fmt.Errorf("updateNewLogPathAndFile err:%v", err)
	}
	if config.MaxKeepDays > 0 {
		go hook.deleteOldLogTimer()
	}
	if !config.DisableWriterBuffer && !config.LogFileDisable {
		go hook.bufferFlusher()
	}
	return nil
}

// 刷新缓冲区
func (hook *logHook) bufferFlusher() {
	for {
		lines := hook.bufferQueue.WaitPopAll()
		hook.WriterLock.RLock()
		for i := 0; i < len(*lines); i++ {
			_, err := hook.OtherBufWriter.Write((*lines)[i])
			if err != nil {
				fmt.Fprintln(os.Stderr, "bufferFlusher Write err:", err)
			}
		}
		if hook.bufferQueue.IsEmptyNoLock() {
			err := hook.OtherBufWriter.Flush()
			if err != nil {
				fmt.Fprintln(os.Stderr, "flushBuffer err:", err)
			}
		}
		hook.WriterLock.RUnlock()
		hook.bufferQueue.RecycleBuffer(lines)
	}
}

func FlushBuf(logger *logrus.Logger) error {
	if logger == nil {
		return nil
	}
	var hooksMap logrus.LevelHooks = logger.Hooks
	if hooksMap == nil {
		return nil
	}
	for _, hooks := range hooksMap {
		if hooks == nil {
			continue
		}
		for _, hook := range hooks {
			if hook == nil {
				continue
			}
			if logHook, ok := hook.(*logHook); ok {
				if logHook == nil {
					continue
				}
				if logHook.OtherBufWriter != nil {
					logHook.WriterLock.Lock()
					if logHook.OtherBufWriter != nil && logHook.OtherBufWriter.Buffered() > 0 {
						err := logHook.OtherBufWriter.Flush()
						if err != nil {
							logHook.WriterLock.Unlock()
							return err
						}
					}
					logHook.WriterLock.Unlock()
				}
			}
		}
	}
	return nil
}

// 删除n天前的日志，n等于0时删除所有日志。
func DeleteOldLog(logger *logrus.Logger, n int) error {
	if logger == nil {
		return nil
	}
	var hooksMap logrus.LevelHooks = logger.Hooks
	if hooksMap == nil {
		return nil
	}
	for _, hooks := range hooksMap {
		if hooks == nil {
			continue
		}
		for _, hook := range hooks {
			if hook == nil {
				continue
			}
			if logHook, ok := hook.(*logHook); ok {
				if logHook == nil {
					continue
				}
				logHook.deleteOldLogOnce(n)
			}
		}
	}
	return nil
}
