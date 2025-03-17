# mylog

A log library based on Logrus and Zap that implements various custom configurations.



## QuickStart

### Install

```
go get -u github.com/doraemonkeys/mylog
```

### zap

```go
package main

import (
	mylog "github.com/doraemonkeys/mylog/zap"
	"go.uber.org/zap"
)

func main() {
	logger := mylog.NewBuilder().Build()
	logger.Info("hello world", zap.String("name", "doraemon"))
	logger.Error("error")
	logger.Warn("warn")
	logger.Debug("debug")
}
```

### Logrus

```go
package main

import (
	"github.com/doraemonkeys/mylog"
	"github.com/sirupsen/logrus"
)

func init() {
	config := mylog.LogConfig{
		LogDir:             `./test_log`,
		LogLevel:            "trace",
		ErrSeparate:         true, // Whether the error log is output to a separate file.
		DateSplit:           true, // Whether to split the log by date.
		MaxKeepDays:         1,   // The maximum number of days to keep the log.
	}
	config.SetKeyValue("foo", "bar")
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



## Configuration Options

```go
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
```

