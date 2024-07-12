package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/doraemonkeys/mylog"
	"github.com/sirupsen/logrus"
)

var logger *logrus.Logger

func init() {
	config := mylog.LogConfig{
		LogDir:    `./test_log`,
		LogLevel:  "trace",
		DateSplit: true,
	}
	config.SetKeyValue("foo", "bar")
	l, err := mylog.NewLogger(config)
	if err != nil {
		panic(err)
	}
	logger = l
}

func main() {

	logger.Trace("trace")
	logger.Debug("debug")
	logger.Info("info")
	logger.Warn("warn")
	logger.Error("error")

	logger.WithFields(logrus.Fields{
		"animal": "walrus",
	}).Info("A walrus appears")

	var err error = errors.New("test error")
	//ERRO[2023-03-02 01:21:35] my msg  error="test error" server="[DEBUG]"
	logger.WithError(err).Error("my msg")

	_, err = os.Open("not exist file")
	if err != nil {
		logger.Error("open file error:", err)
	}

	fmt.Println()

	for i := 0; i < 200; i++ {
		fmt.Println("i:", i)
		logger.Info("info")
		logger.Trace("trace")
		logger.Debug("debug")
		logger.Info("info")
		logger.Warn("warn")
		logger.Error("error")
		// err := mylog.FlushBuf(logrus.StandardLogger())
		// if err != nil {
		// 	panic(err)
		// }
		fmt.Println("sleep 1s")
		// time.Sleep(time.Second)
	}

}
