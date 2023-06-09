package mylog

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

func (hook *logHook) Fire(entry *logrus.Entry) error {
	if hook.LogConfig.key != "" {
		entry.Data[hook.LogConfig.key] = hook.LogConfig.value
	}
	file := entry.Caller.File
	file = getShortFileName(file, fmt.Sprint(entry.Caller.Line))
	entry.Data["FILE"] = file
	entry.Data["FUNC"] = entry.Caller.Function[strings.LastIndex(entry.Caller.Function, ".")+1:]

	if !hook.LogConfig.ShowShortFileInConsole {
		defer delete(entry.Data, "FILE")
	}
	if !hook.LogConfig.ShowFuncInConsole {
		defer delete(entry.Data, "FUNC")
	}
	// 为debug级别的日志添加颜色
	// if entry.Level == logrus.DebugLevel {
	// 	defer func() {
	// 		// \033[35m 紫色 \033[0m
	// 		entry.Message = "\x1b[35m" + entry.Message + "\x1b[0m"
	// 	}()
	// }

	//取消日志输出到文件
	if hook.LogConfig.LogFileDisable {
		return nil
	}

	//msg前添加固定前缀 DORAEMON
	//entry.Message = "DORAEMON " + entry.Message

	line, err := entry.Bytes()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to read entry, %v", err)
		return err
	}
	line = eliminateColor(line)

	hook.checkSplit()

	hook.WriterLock.RLock()
	defer hook.WriterLock.RUnlock()
	if hook.ErrWriter != nil && entry.Level <= logrus.ErrorLevel {
		hook.LogSize += int64(len(line))
		hook.ErrWriter.Write(line)

		if !hook.LogConfig.ErrInNormal {
			return nil
		}
	}

	if hook.OtherWriter != nil {
		hook.LogSize += int64(len(line))
		hook.OtherWriter.Write(line)
	}

	return nil
}

func (hook *logHook) Levels() []logrus.Level {
	//return []logrus.Level{logrus.ErrorLevel}

	//hook全部级别
	return logrus.AllLevels
}

// 检查是否需要分割日志
func (hook *logHook) checkSplit() {
	if hook.LogConfig.DateSplit {
		//按日期分割
		now := time.Now().In(hook.LogConfig.TimeLocation).Format(hook.dateFmt)
		if hook.FileDate != now {
			hook.WriterLock.Lock()
			if hook.FileDate == now {
				//已经分割过了
				hook.WriterLock.Unlock()
				return
			}
			hook.FileDate = now
			hook.split_date()
			hook.WriterLock.Unlock()
		}
		return
	}

	if hook.LogConfig.MaxLogSize > 0 {
		//按大小分割
		if hook.LogSize >= hook.LogConfig.MaxLogSize {
			//fmt.Println("日志大小超过限制，开始分割日志", hook.LogSize, hook.LogConfig.MaxLogSize)
			hook.WriterLock.Lock()
			if hook.LogSize < hook.LogConfig.MaxLogSize {
				//已经分割过了
				hook.WriterLock.Unlock()
				return
			}
			hook.LogSize = 0
			hook.split_size()
			hook.WriterLock.Unlock()
		}
		return
	}
}

// 按大小分割日志
func (hook *logHook) split_size() {
	if hook.ErrWriter != nil {
		hook.ErrWriter.Close()
	}
	if hook.OtherWriter != nil {
		hook.OtherWriter.Close()
	}
	err := hook.updateNewLogPathAndFile()
	if err != nil {
		panic(fmt.Sprintf("分割日志失败: %v", err))
	}
}

// 按日期分割日志
func (hook *logHook) split_date() {
	if hook.ErrWriter != nil {
		hook.ErrWriter.Close()
	}
	if hook.OtherWriter != nil {
		hook.OtherWriter.Close()
	}
	err := hook.updateNewLogPathAndFile()
	if err != nil {
		panic(fmt.Sprintf("分割日志失败: %v", err))
	}
}

func (hook *logHook) updateNewLogPathAndFile() error {
	if hook.LogConfig.LogFileDisable {
		return nil
	}

	// 检查日志目录是否存在
	if hook.LogConfig.LogPath != "" {
		if _, err := os.Stat(hook.LogConfig.LogPath); os.IsNotExist(err) {
			err = os.MkdirAll(hook.LogConfig.LogPath, 0755)
			if err != nil {
				return err
			}
		}
	}

	//更新日期(不多余，split_size也会用到)
	hook.FileDate = time.Now().In(hook.LogConfig.TimeLocation).Format(hook.dateFmt)

	var tempFileName string
	//默认情况
	if !hook.LogConfig.DateSplit && hook.LogConfig.MaxLogSize == 0 {
		tempFileName = hook.LogConfig.DefaultLogName
	}
	//按大小分割
	if hook.LogConfig.MaxLogSize > 0 {
		//按大小分割时，文件名格式为 2006_01_02_030405
		tempFileName = time.Now().In(hook.LogConfig.TimeLocation).Format(hook.dateFmt2)
	}
	//按日期分割
	if hook.LogConfig.DateSplit {
		tempFileName = hook.FileDate
	}

	if !hook.LogConfig.ErrSeparate {
		return hook.openLogFile(tempFileName)
	}
	return hook.openTwoLogFile(tempFileName)
}

func (hook *logHook) openTwoLogFile(tempFileName string) error {
	var errorFileName string
	var commonFileName string
	if hook.LogConfig.LogFileNameSuffix == "" {
		errorFileName = tempFileName + "_" + "ERROR" + hook.LogConfig.LogExt
		commonFileName = tempFileName + hook.LogConfig.LogExt
	} else {
		errorFileName = tempFileName + "_" + "ERROR" + "_" + hook.LogConfig.LogFileNameSuffix + hook.LogConfig.LogExt
		commonFileName = tempFileName + "_" + hook.LogConfig.LogFileNameSuffix + hook.LogConfig.LogExt
	}
	errorFileName = makeFileNameLegal(errorFileName)
	commonFileName = makeFileNameLegal(commonFileName)

	newPath := filepath.Join(hook.LogConfig.LogPath, hook.FileDate)
	errorFileName = filepath.Join(newPath, errorFileName)
	commonFileName = filepath.Join(newPath, commonFileName)
	err := os.MkdirAll(newPath, 0777)
	if err != nil {
		return err
	}

	file, err := os.OpenFile(errorFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return err
	}
	hook.ErrWriter = file
	hook.LogSize, _ = file.Seek(0, io.SeekEnd)

	file2, err := os.OpenFile(commonFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return err
	}
	hook.OtherWriter = file2
	tempSize, _ := file2.Seek(0, io.SeekEnd)
	hook.LogSize += tempSize
	return nil
}

func (hook *logHook) openLogFile(tempFileName string) error {
	var newFileName string
	if hook.LogConfig.LogFileNameSuffix == "" {
		newFileName = tempFileName + hook.LogConfig.LogExt
	} else {
		newFileName = tempFileName + "_" + hook.LogConfig.LogFileNameSuffix + hook.LogConfig.LogExt
	}
	newFileName = makeFileNameLegal(newFileName)
	newFileName = filepath.Join(hook.LogConfig.LogPath, newFileName)

	file, err := os.OpenFile(newFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return err
	}
	hook.OtherWriter = file

	//更新日志大小(文件为空时，返回0)
	hook.LogSize, _ = file.Seek(0, io.SeekEnd)
	return nil
}

var oldLogCheckerOnline = false

func (hook *logHook) deleteOldLog() {
	for {
		hook.deleteOldLogOnce()
		time.Sleep(time.Hour * 24)
	}
}

// 删除过期日志
func (hook *logHook) deleteOldLogOnce() {
	if hook.LogConfig.MaxKeepDays == 0 {
		return
	}
	hook.deleteOldLogDirOnce(hook.LogConfig.LogPath)
	hook.deleteOldLogFileOnce(hook.LogConfig.LogPath)
}

func (hook *logHook) deleteOldLogDirOnce(dir string) {
	if hook.LogConfig.MaxKeepDays == 0 {
		return
	}
	dirs, err := getFolderNamesInPath(hook.LogConfig.LogPath)
	if err != nil {
		logrus.Errorf("deleteOldLog getDirs err:%v", err)
		return
	}
	for _, dir := range dirs {
		date, err := time.Parse(hook.dateFmt, dir)
		if err != nil {
			logrus.Errorf("deleteOldLog time.Parse err:%v", err)
			continue
		}
		if time.Since(date).Hours() > float64(hook.LogConfig.MaxKeepDays*24) {
			hook.deleteOldLogFileOnce(filepath.Join(hook.LogConfig.LogPath, dir))
			//if dir is empty
			if isEmptyDir(dir) {
				err := os.Remove(dir)
				if err != nil {
					logrus.Errorf("deleteOldLogDir os.Remove err:%v", err)
				}
			}
		}
	}
}

func (hook *logHook) deleteOldLogFileOnce(dir string) {
	if hook.LogConfig.MaxKeepDays == 0 {
		return
	}
	files, err := getFileNmaesInPath(dir)
	if err != nil {
		logrus.Errorf("deleteOldLog getFiles err:%v", err)
		return
	}
	for _, fileName := range files {
		fileName = strings.ToLower(fileName)
		if hook.ErrWriter != nil && fileName == strings.ToLower(hook.ErrWriter.Name()) {
			continue
		}
		if hook.OtherWriter != nil && fileName == strings.ToLower(hook.OtherWriter.Name()) {
			continue
		}
		// 最后修改时间
		fileInfo, err := os.Stat(filepath.Join(dir, fileName))
		if err != nil {
			logrus.Errorf("deleteOldLog os.Stat err:%v", err)
			continue
		}
		if time.Since(fileInfo.ModTime()).Hours() > float64(hook.LogConfig.MaxKeepDays*24) {
			err := os.Remove(filepath.Join(dir, fileName))
			if err != nil {
				logrus.Errorf("deleteOldLog os.Remove err:%v", err)
			}
		}
	}
}
