package mylog

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/doraemonkeys/doraemon"
	myformatter "github.com/doraemonkeys/mylog/formatter"
	"github.com/sirupsen/logrus"
)

// Default log folder path.
//
// After enabling the maximum retention days for logs, if the log folder path is not set, it defaults to this path.
const DefaultSavePath = "./logs"

// logrus level
const (
	PanicLevel = "panic"
	FatalLevel = "fatal"
	ErrorLevel = "error"
	WarnLevel  = "warn"
	InfoLevel  = "info"
	DebugLevel = "debug"
	TraceLevel = "trace"
)

type LogConfig struct {
	// Path for log storage.
	LogDir string
	// Log file name suffix
	LogFileNameSuffix string
	// Default log file name (ignored if split by date or size)
	DefaultLogName string
	// Separate error logs (for Error level and above)
	ErrSeparate bool
	// Exclude error logs from normal log file (when errors are separated)
	ErrNotInNormal bool
	// Split logs by date (cannot be used with size split)
	DateSplit bool
	// Disable file output for logs
	LogFileDisable bool
	// Disable console output for logs
	NoConsole bool
	// Disable timestamp in logs
	NoTimestamp bool
	// Timestamp format, default is 2006-01-02 15:04:05.000
	TimestampFormat string
	// Show short file path in console output
	ShowShortFileInConsole bool
	// Show function name in console output
	ShowFuncInConsole bool
	// Disable caller information
	DisableCaller bool
	// Disable write buffer
	DisableWriterBuffer bool
	// Write buffer size, default is 4096 bytes
	WriterBufferSize int
	// Output in JSON format
	JSONFormat bool
	// Disable color output
	DisableColors bool
	// Disables the truncation of the level text to 4 characters.
	DisableLevelTruncation bool
	// PadLevelText Adds padding the level text so that all the levels
	// output at the same length PadLevelText is a superset of the DisableLevelTruncation option
	PadLevelText bool
	// Split logs by size in bytes (cannot be used with date split)
	MaxLogSize int64
	// Maximum retention days for logs. After enabling this, if the log folder path is not set, it defaults to DefaultSavePath.
	// Please do not place other files in the log folder, otherwise they may be deleted.
	MaxKeepDays int
	// Log file extension (default is .log)
	LogExt string
	// Log level (panic, fatal, error, warn, info, debug, trace)
	LogLevel string
	// Time zone
	TimeLocation *time.Location
	// Key for appending to each log entry
	key string
	// Value for appending to each log entry
	value interface{}
	// Suffix for log files that should not be deleted
	keepSuffix string
}

// SetKeyValue sets the key and value for appending to each log entry.
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

// InitGlobalLogger initializes the global logger.The global logger is the default logger of logrus.
func InitGlobalLogger(config LogConfig) error {
	return initlLog(logrus.StandardLogger(), config)
}

// NewLogger creates a new logger.
func NewLogger(config LogConfig) (*logrus.Logger, error) {
	logger := logrus.New()
	err := initlLog(logger, config)
	if err != nil {
		return nil, err
	}
	return logger, nil
}

var logDirsMap = make(map[string]bool)

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

	if logDirsMap[filepath.Clean(config.LogDir)] {
		return fmt.Errorf("logDir:%s has been used", config.LogDir)
	} else {
		logDirsMap[filepath.Clean(config.LogDir)] = true
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

// Deprecated: You don't need to call this function now.
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

// Delete n days old logs, n equals 0 to delete all logs.
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
