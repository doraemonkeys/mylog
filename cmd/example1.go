package main

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/Doraemonkeys/mylog"
	"github.com/sirupsen/logrus"
)

func init() {
	config := mylog.LogConfig{
		LogPath:  `./test_log`,
		LogLevel: "debug",
		//LogFileNameSuffix: "test",
		//LogExt:            "log",
		//MaxLogSize: 1024 * 1024 * 10, //10M
		//MaxLogSize: 1024, //1K
		//ErrInNormal: true, //错误日志是否也输出到普通日志
		ErrSeparate: true, //错误日志是否单独输出到文件
		DateSplit:   true, //是否按日期分割日志
		//TimeLocation: time.Local, //时区
		//ShowShortFileInConsole: true, //控制台是否显示文件名和行号
		//ShowFuncInConsole: true, //控制台是否显示函数名
		//NoFile: true, //不输出到文件
		//NoConsole: true, //不输出到控制台
		//NoTimestamp: true, //不显示时间戳
		//DisableCaller: true, //关闭调用者信息
		MaxKeepDays: 1,
	}
	config.SetKeyValue("server", "[DEBUG]")
	err := mylog.InitGlobalLogger(config)
	if err != nil {
		panic(err)
	}
}

func main() {
	//.\*.exe --level=error
	//*.exe -level=info  //严重等级比info低的都不输出(比如debug)

	logrus.Trace("trace")
	logrus.Debug("debug")
	logrus.Info("info")
	logrus.Warn("warn")
	logrus.Error("error")

	var err error = errors.New("test error")
	//ERRO[2023-03-02 01:21:35] my msg  error="test error" server="[DEBUG]"
	logrus.WithError(err).Error("my msg")

	_, err = os.Open("not exist file")
	if err != nil {
		logrus.Error("open file error:", err)
	}

	fmt.Println()

	for i := 0; i < 100; i++ {
		fmt.Println("i:", i)
		logrus.Info("info")
		logrus.Trace("trace")
		logrus.Debug("debug")
		logrus.Info("info")
		logrus.Warn("warn")
		logrus.Error("error")
		fmt.Println("sleep 1s")
		time.Sleep(time.Second)
	}

}
