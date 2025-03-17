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
