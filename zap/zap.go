package zap

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

type ZapBuilder struct {
	// Path for log storage.
	logDir string
	// Default log file name
	defaultLogName string
	// Disable separate error logs (for Error level and above)
	noErrSeparate bool
	// Disable file output for logs
	logFileDisable bool
	// Disable console output for logs
	noConsole bool
	// Disable timestamp in logs
	noTimestamp bool
	// Timestamp format, default is ISO8601TimeEncoder
	timestampFormat string
	// Disable caller information
	disableCaller bool

	// Disable color output
	disableColors bool
	// Output in JSON format for file
	jsonFormatFile bool
	// Output in JSON format for console
	jsonFormatConsole bool
	// Log file extension (default is .log)
	logFileExt string
	// maxLogSizeMB is the maximum size in megabytes of the log file before it gets rotated. It defaults to 100 megabytes.
	maxLogSizeMB int
	// Maximum retention days for logs.
	maxKeepDays int
	// Log level (panic, fatal, error, warn, info, debug, trace)
	logLevel zapcore.Level
	// Time zone
	timeLocation *time.Location
	// Stacktrace level
	stacktraceLevel zapcore.Level
	// callerSkip
	callerSkip int

	// enable standard error output
	// enableStdErr bool
}

// ReplaceGlobals replaces the global Logger and SugaredLogger, and returns a function to restore the original values. It's safe for concurrent use.
func ReplaceGlobals(logger *zap.Logger) func() {
	return zap.ReplaceGlobals(logger)
}

func NewBuilder() *ZapBuilder {
	return &ZapBuilder{
		logDir:          "./logs",
		logFileExt:      ".log",
		defaultLogName:  "default",
		timeLocation:    time.Local,
		logLevel:        zapcore.InfoLevel,
		stacktraceLevel: zapcore.FatalLevel,
		maxLogSizeMB:    100, // lumberjack.Logger default max size
	}
}

func (b *ZapBuilder) LogDir(logDir string) *ZapBuilder {
	b.logDir = logDir
	return b
}

func (b *ZapBuilder) DefaultLogName(defaultLogName string) *ZapBuilder {
	b.defaultLogName = defaultLogName
	return b
}

func (b *ZapBuilder) LogFileExt(logExt string) *ZapBuilder {
	b.logFileExt = logExt
	return b
}

func (b *ZapBuilder) NoErrSeparate() *ZapBuilder {
	b.noErrSeparate = true
	return b
}

func (b *ZapBuilder) NoLogFile() *ZapBuilder {
	b.logFileDisable = true
	return b
}

func (b *ZapBuilder) NoConsole() *ZapBuilder {
	b.noConsole = true
	return b
}

func (b *ZapBuilder) NoTimestamp() *ZapBuilder {
	b.noTimestamp = true
	return b
}

func (b *ZapBuilder) NoCaller() *ZapBuilder {
	b.disableCaller = true
	return b
}

func (b *ZapBuilder) NoColors() *ZapBuilder {
	b.disableColors = true
	return b
}

func (b *ZapBuilder) JSONFormatFile() *ZapBuilder {
	b.jsonFormatFile = true
	return b
}

func (b *ZapBuilder) JSONFormatConsole() *ZapBuilder {
	b.jsonFormatConsole = true
	return b
}

// MaxLogSize sets the maximum size in megabytes of the log file before it gets rotated. It defaults to 100 megabytes.
func (b *ZapBuilder) MaxLogSize(maxLogSize int) *ZapBuilder {
	b.maxLogSizeMB = maxLogSize
	return b
}

// MaxKeepDays sets the maximum retention days for logs.
func (b *ZapBuilder) MaxKeepDays(maxKeepDays int) *ZapBuilder {
	b.maxKeepDays = maxKeepDays
	return b
}

// Level sets the log level.
func (b *ZapBuilder) Level(logLevel zapcore.Level) *ZapBuilder {
	b.logLevel = logLevel
	return b
}

// StacktraceLevel sets the stacktrace level.
func (b *ZapBuilder) StacktraceLevel(stacktraceLevel zapcore.Level) *ZapBuilder {
	b.stacktraceLevel = stacktraceLevel
	return b
}

// TimeLocation sets the time zone.
func (b *ZapBuilder) TimeLocation(timeLocation *time.Location) *ZapBuilder {
	b.timeLocation = timeLocation
	return b
}

// CallerSkip sets the caller skip.
//
// CallerSkip increases the number of callers skipped by caller annotation (as enabled by the AddCaller option).
// When building wrappers around the Logger and SugaredLogger,
// supplying this Option prevents zap from always reporting the wrapper code as the caller.
func (b *ZapBuilder) CallerSkip(callerSkip int) *ZapBuilder {
	b.callerSkip = callerSkip
	return b
}

func (b *ZapBuilder) Build() *zap.Logger {
	if b.logFileDisable && b.noConsole {
		return zap.NewNop()
	}
	if b.logFileDisable {
		return b.buildOnlyFile()
	}
	if b.noConsole {
		return b.buildOnlyConsole()
	}
	return b.build()
}

func (b *ZapBuilder) build() *zap.Logger {
	core := zapcore.NewTee(b.buildFileCore(), b.buildConsoleCore())
	opts := []zap.Option{
		zap.AddStacktrace(b.stacktraceLevel),
	}
	if !b.disableCaller {
		opts = append(opts, zap.AddCaller())
		if b.callerSkip > 0 {
			opts = append(opts, zap.AddCallerSkip(b.callerSkip))
		}
	}
	return zap.New(core, opts...)
}

func (b *ZapBuilder) buildOnlyConsole() *zap.Logger {
	core := b.buildConsoleCore()
	opts := []zap.Option{
		zap.AddStacktrace(b.stacktraceLevel),
	}
	if !b.disableCaller {
		opts = append(opts, zap.AddCaller())
		if b.callerSkip > 0 {
			opts = append(opts, zap.AddCallerSkip(b.callerSkip))
		}
	}
	return zap.New(core, opts...)
}

func (b *ZapBuilder) buildOnlyFile() *zap.Logger {
	core := b.buildFileCore()
	opts := []zap.Option{
		zap.AddStacktrace(b.stacktraceLevel),
	}
	if !b.disableCaller {
		opts = append(opts, zap.AddCaller())
		if b.callerSkip > 0 {
			opts = append(opts, zap.AddCallerSkip(b.callerSkip))
		}
	}
	return zap.New(core, opts...)
}

func (b *ZapBuilder) buildFileCore() zapcore.Core {
	// If file logging is disabled, return an empty core
	if b.logFileDisable {
		return zapcore.NewNopCore()
	}

	// Configure encoder
	encoderConfig := zapcore.EncoderConfig{
		MessageKey:     "msg",
		LevelKey:       "level",
		TimeKey:        "time",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     b.getTimeEncoder(),
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// Encoder
	var encoder zapcore.Encoder
	if b.jsonFormatFile {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	// Create normal log file
	normalLogPath := filepath.Join(b.logDir, b.defaultLogName+b.logFileExt)
	normalLogWriter := &lumberjack.Logger{
		Filename:   normalLogPath,
		MaxSize:    b.maxLogSizeMB, // MB
		MaxBackups: 0,              // 不限制备份数量
		MaxAge:     b.maxKeepDays,  // 保留天数
		Compress:   true,           // 压缩旧日志
		LocalTime:  true,           // 备份文件名使用本地时间
	}
	normalWriteSyncer := zapcore.AddSync(normalLogWriter)

	// If error logging is not separated, return the normal log core directly
	if b.noErrSeparate {
		return zapcore.NewCore(encoder, normalWriteSyncer, b.logLevel)
	}

	// Create error log file
	errorLogPath := filepath.Join(b.logDir, b.defaultLogName+".error"+b.logFileExt)
	errorLogWriter := &lumberjack.Logger{
		Filename:   errorLogPath,
		MaxSize:    b.maxLogSizeMB, // MB
		MaxBackups: 0,
		MaxAge:     b.maxKeepDays,
		Compress:   true,
		LocalTime:  true,
	}
	errorWriteSyncer := zapcore.AddSync(errorLogWriter)

	// Create error log core, only record errors and above
	errorCore := zapcore.NewCore(encoder, errorWriteSyncer, zap.ErrorLevel)

	// Error log also records in normal log
	normalCore := zapcore.NewCore(encoder, normalWriteSyncer, b.logLevel)
	return zapcore.NewTee(normalCore, errorCore)
}

func (b *ZapBuilder) buildConsoleCore() zapcore.Core {
	// If console logging is disabled, return an empty core
	if b.noConsole {
		return zapcore.NewNopCore()
	}

	// Configure encoder
	encoderConfig := zapcore.EncoderConfig{
		MessageKey:     "msg",
		LevelKey:       "level",
		TimeKey:        "time",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeCaller:   zapcore.ShortCallerEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeTime:     b.getTimeEncoder(),
	}

	// Set level encoder based on whether colors are disabled
	if b.disableColors {
		encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	} else {
		encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	// Select encoder based on configuration
	var encoder zapcore.Encoder
	if b.jsonFormatConsole {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	// Create console output
	consoleWriteSyncer := zapcore.Lock(os.Stdout)
	return zapcore.NewCore(encoder, consoleWriteSyncer, b.logLevel)
}

func ParseLevel(levelStr string) zapcore.Level {
	switch strings.ToLower(levelStr) {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn", "warning":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	case "dpanic":
		return zapcore.DPanicLevel
	case "panic":
		return zapcore.PanicLevel
	case "fatal":
		return zapcore.FatalLevel
	default:
		return zapcore.InfoLevel
	}
}

func (b *ZapBuilder) getTimeEncoder() zapcore.TimeEncoder {
	if b.noTimestamp {
		return func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {}
	}

	timeFormat := b.timestampFormat
	if timeFormat == "" {
		return zapcore.ISO8601TimeEncoder
	}

	return func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		if b.timeLocation != nil {
			enc.AppendString(t.In(b.timeLocation).Format(timeFormat))
			return
		}
		// Reference zapcore.encodeTimeLayout
		type appendTimeEncoder interface {
			AppendTimeLayout(time.Time, string)
		}
		if enc, ok := enc.(appendTimeEncoder); ok {
			enc.AppendTimeLayout(t, timeFormat)
			return
		}
		enc.AppendString(t.Format(timeFormat))
	}
}

// 自定义核心：用于过滤特定级别的日志
// type levelFilterCore struct {
// 	zapcore.Core
// 	maxLevel zapcore.Level
// 	minLevel zapcore.Level
// }

// func (c *levelFilterCore) Enabled(lvl zapcore.Level) bool {
// 	return lvl <= c.maxLevel && lvl >= c.minLevel && c.Core.Enabled(lvl)
// }

// func (c *levelFilterCore) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
// 	if ent.Level <= c.maxLevel && ent.Level >= c.minLevel {
// 		return c.Core.Check(ent, ce)
// 	}
// 	return ce
// }
