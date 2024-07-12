package mylog

import (
	"bufio"
	"errors"
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

	if !hook.LogConfig.DisableCaller && entry.Caller != nil {
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
	}

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

	// ------------------- 加锁写入文件/缓冲 -------------------
	hook.WriterLock.RLock()
	defer func() {
		hook.WriterLock.RUnlock()
		if !hook.LogConfig.DisableWriterBuffer &&
			(entry.Level == logrus.PanicLevel || entry.Level == logrus.FatalLevel) {
			hook.WriterLock.Lock()
			_ = hook.OtherBufWriter.Flush()
			hook.WriterLock.Unlock()
		}
	}()

	if hook.ErrWriter != nil && entry.Level <= logrus.ErrorLevel {
		// 单独输出的错误日志也算在日志大小限制内
		hook.LogSize += int64(len(line))
		_, err := hook.ErrWriter.Write(line)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to write error to the file, %v", err)
			return err
		}
		if hook.LogConfig.ErrNotInNormal {
			return nil
		}
	}

	// safe check
	if hook.OtherWriter == nil && hook.OtherBufWriter == nil {
		fmt.Fprintf(os.Stderr, "Unexpected error, OtherWriter and OtherBufWriter are both nil")
		return errors.New("unexpected error, OtherWriter and OtherBufWriter are both nil")
	}
	if hook.LogConfig.DisableWriterBuffer && hook.OtherWriter == nil {
		fmt.Fprintf(os.Stderr, "Unexpected error, OtherWriter is nil when DisableWriterBuffer is true")
		return errors.New("unexpected error, OtherWriter is nil when DisableWriterBuffer is true")
	}
	if !hook.LogConfig.DisableWriterBuffer && hook.OtherBufWriter == nil {
		fmt.Fprintf(os.Stderr, "Unexpected error, OtherBufWriter is nil when DisableWriterBuffer is false")
		return errors.New("unexpected error, OtherBufWriter is nil when DisableWriterBuffer is false")
	}

	if hook.LogConfig.DisableWriterBuffer {
		_, err := hook.OtherWriter.Write(line)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to write log to the file, %v", err)
			return err
		}
	} else {
		hook.bufferQueue.Push(line)
	}

	hook.LogSize += int64(len(line))

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
			hook.split()
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
			hook.split()
			hook.WriterLock.Unlock()
		}
		return
	}
}

// 必须加锁调用
func (hook *logHook) split() {
	oldErrWriter := hook.ErrWriter
	oldOtherWriter := hook.OtherWriter
	oldOtherBufWriter := hook.OtherBufWriter
	if oldOtherBufWriter != nil {
		oldOtherBufWriter.Flush()
	}
	err := hook.updateNewLogPathAndFile()
	if err != nil {
		msg := fmt.Sprintf("ERROR!!!!!!!!!!!!!!!!!!!!!!!! split log file err:%v !!!!!!!!!!!!!!!!!!!!!!!!ERROR\n", err)
		fmt.Fprint(os.Stderr, msg)
		hook.ErrWriter = oldErrWriter
		hook.OtherWriter = oldOtherWriter
		hook.OtherBufWriter = oldOtherBufWriter
		if hook.ErrWriter != nil {
			hook.ErrWriter.Write([]byte(msg))
		} else if hook.OtherWriter != nil {
			hook.OtherWriter.Write([]byte(msg))
		} else if hook.OtherBufWriter != nil {
			hook.OtherBufWriter.Write([]byte(msg))
		}
		return
	}
	if oldErrWriter != nil {
		oldErrWriter.Close()
	}
	if oldOtherWriter != nil {
		oldOtherWriter.Close()
	}
}

func (hook *logHook) updateNewLogPathAndFile() error {
	if hook.LogConfig.LogFileDisable {
		return nil
	}

	// 检查日志目录是否存在
	if hook.LogConfig.LogDir != "" {
		if !DirIsExist(hook.LogConfig.LogDir) {
			err := os.MkdirAll(hook.LogConfig.LogDir, 0755)
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
		//按大小分割时，文件名格式为 2006_01_02_150405
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
		errorFileName = tempFileName + "_" + "error" + hook.LogConfig.LogExt
		commonFileName = tempFileName + hook.LogConfig.LogExt
	} else {
		errorFileName = tempFileName + "_" + "error" + "_" + hook.LogConfig.LogFileNameSuffix + hook.LogConfig.LogExt
		commonFileName = tempFileName + "_" + hook.LogConfig.LogFileNameSuffix + hook.LogConfig.LogExt
	}
	errorFileName = makeFileNameLegal(errorFileName)
	commonFileName = makeFileNameLegal(commonFileName)

	newPath := filepath.Join(hook.LogConfig.LogDir, hook.FileDate)
	errorFileName = filepath.Join(newPath, errorFileName)
	commonFileName = filepath.Join(newPath, commonFileName)

	var (
		lazyFile *lazyFileWriter
		file2    *os.File
		ok       bool
		err      error
	)
	lazyFile, file2, ok, err = hook.tryOpenTwoOldLogFile(errorFileName, commonFileName)
	if err != nil {
		return err
	}
	if !ok {
		err := os.MkdirAll(newPath, 0777)
		if err != nil {
			return err
		}
		lazyFile = newLazyFileWriter(errorFileName)
		file2, err = os.OpenFile(commonFileName, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
		if err != nil {
			return err
		}
	}

	hook.ErrWriter = lazyFile
	if lazyFile.IsCreated() {
		hook.LogSize, _ = lazyFile.Seek(0, io.SeekEnd)
	} else {
		hook.LogSize = 0
	}
	if hook.LogConfig.DisableWriterBuffer {
		hook.OtherWriter = file2
	} else {
		hook.OtherBufWriter = bufio.NewWriterSize(file2, hook.WriterBufferSize)
	}
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
	newFileName = filepath.Join(hook.LogConfig.LogDir, newFileName)

	file, err := hook.tryOpenOldLogFile(newFileName)
	if err != nil {
		return err
	}

	if hook.LogConfig.DisableWriterBuffer {
		hook.OtherWriter = file
	} else {
		hook.OtherBufWriter = bufio.NewWriterSize(file, hook.WriterBufferSize)
	}

	//更新日志大小(文件为空时，返回0)
	hook.LogSize, _ = file.Seek(0, io.SeekEnd)
	return nil
}

func (hook *logHook) tryOpenOldLogFile(newFileName string) (*os.File, error) {
	if hook.LogConfig.DateSplit && hook.LogConfig.MaxLogSize > 0 {
		return nil, errors.New("按日期分割和按大小分割不能同时开启")
	}
	if hook.LogConfig.DateSplit {
		return os.OpenFile(newFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	}
	// default
	if !hook.LogConfig.DateSplit && hook.LogConfig.MaxLogSize == 0 {
		return os.OpenFile(newFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	}

	//按大小分割
	oldLogFiles, err := getFileNmaesInPath(hook.LogConfig.LogDir)
	if err != nil {
		return nil, err
	}
	var latestLogFile string = hook.dateFmt2

	for _, file := range oldLogFiles {
		if len(file) != len(filepath.Base(newFileName)) {
			continue
		}
		fileNameTime := file[0:len(hook.dateFmt2)]
		latestLogFileTime := latestLogFile[0:len(hook.dateFmt2)]
		if timeStringCompare(fileNameTime, latestLogFileTime, hook.dateFmt2) == 1 {
			latestLogFile = file
		}
	}
	if latestLogFile == hook.dateFmt2 {
		return os.OpenFile(newFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	}
	// 检查文件大小
	fileStat, err := os.Stat(filepath.Join(hook.LogConfig.LogDir, latestLogFile))
	if err != nil {
		return nil, err
	}
	if fileStat.Size() < hook.LogConfig.MaxLogSize {
		return os.OpenFile(filepath.Join(hook.LogConfig.LogDir, latestLogFile), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	}
	// 文件大小超过限制，新建文件
	return os.OpenFile(newFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
}

func (hook *logHook) tryOpenTwoOldLogFile(errorFileName, commonFileName string) (*lazyFileWriter, *os.File, bool, error) {
	if hook.LogConfig.MaxLogSize == 0 {
		return nil, nil, false, nil
	}
	dirs, err := getFolderNamesInPath(hook.LogConfig.LogDir)
	if err != nil {
		return nil, nil, false, err
	}
	var latestFolder string = hook.dateFmt
	for _, dir := range dirs {
		if len(dir) != len(hook.dateFmt) {
			continue
		}
		if timeStringCompare(dir, latestFolder, hook.dateFmt) == 1 {
			latestFolder = dir
		}
	}
	if latestFolder == hook.dateFmt {
		return nil, nil, false, nil
	}
	// 检查文件夹大小
	folderSize, err := getFolderSize(filepath.Join(hook.LogConfig.LogDir, latestFolder))
	if err != nil {
		return nil, nil, false, err
	}
	// 直接在文件夹中创建新文件
	if folderSize < hook.LogConfig.MaxLogSize {
		errorFileName = filepath.Join(hook.LogConfig.LogDir, latestFolder, filepath.Base(errorFileName))
		commonFileName = filepath.Join(hook.LogConfig.LogDir, latestFolder, filepath.Base(commonFileName))
	}
	file := newLazyFileWriter(errorFileName)
	file2, err := os.OpenFile(commonFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, nil, false, err
	}
	return file, file2, true, nil
}

func (hook *logHook) deleteOldLogTimer() {
	for {
		hook.deleteOldLogOnce(hook.LogConfig.MaxKeepDays)
		time.Sleep(time.Hour * 24)
	}
}

// 删除过期日志(n<=0时删除所有)
func (hook *logHook) deleteOldLogOnce(n int) {
	if hook.LogConfig.LogDir == "" {
		// 仅支持删除文件夹中的日志
		return
	}
	if n <= 0 {
		// return
		hook.WriterLock.Lock()
		if hook.OtherBufWriter != nil {
			hook.OtherBufWriter.Flush()
		}
		if hook.ErrWriter != nil && hook.ErrWriter.IsCreated() {
			path := hook.ErrWriter.Name()
			hook.ErrWriter.Close()
			err := os.Remove(path)
			if err != nil {
				fmt.Fprintf(os.Stderr, "deleteOldLog os.Remove err:%v", err)
			}
		}
		if hook.OtherWriter != nil {
			path := hook.OtherWriter.Name()
			hook.OtherWriter.Close()
			err := os.Remove(path)
			if err != nil {
				fmt.Fprintf(os.Stderr, "deleteOldLog os.Remove err:%v", err)
			}
		}
		hook.updateNewLogPathAndFile()
		hook.WriterLock.Unlock()
	}
	hook.deleteOldLogDirOnce(hook.LogConfig.LogDir, n)
	hook.deleteOldLogFileOnce(hook.LogConfig.LogDir, n)
}

// 删除文件夹中n天前的日志文件夹。
// 由于调用了logrus.Errorf，所以不要对此方法加锁，否则会死锁。
func (hook *logHook) deleteOldLogDirOnce(dir string, n int) {
	// if n <= 0 {
	// 	return
	// }
	dirs, err := getFolderNamesInPath(dir)
	if err != nil {
		logrus.Errorf("deleteOldLog getDirs err:%v", err)
		return
	}
	for _, dir := range dirs {
		if strings.HasSuffix(dir, hook.LogConfig.keepSuffix) {
			continue
		}
		var dirNameTime = dir
		if len(dir) > len(hook.dateFmt) {
			dirNameTime = dir[0:len(hook.dateFmt)]
		}
		date, err := time.Parse(hook.dateFmt, dirNameTime)
		if err != nil {
			// logrus.Errorf("deleteOldLog time.Parse err:%v", err)
			// can not parse dir to time, ignore
			continue
		}
		if time.Since(date).Hours() > float64(n*24) {
			dirPath := filepath.Join(hook.LogConfig.LogDir, dir)
			hook.deleteOldLogFileOnce(dirPath, n)
			//if dir is empty
			if isEmptyDir(dirPath) {
				err := os.Remove(dirPath)
				if err != nil {
					logrus.Errorf("deleteOldLogDir os.Remove err:%v", err)
				}
			}
		}
	}
}

// 删除文件夹中n天前的日志文件。
// 由于调用了logrus.Errorf，所以不要对此方法加锁，否则会死锁。
func (hook *logHook) deleteOldLogFileOnce(dir string, n int) {
	// if n <= 0 {
	// 	return
	// }
	files, err := getFileNmaesInPath(dir)
	if err != nil {
		logrus.Errorf("deleteOldLog getFiles err:%v", err)
		return
	}
	// var ErrWriterFilePath string
	// var OthWriterFilePath string
	// if hook.ErrWriter != nil {
	// 	ErrWriterFilePath, _ = filepath.Abs(hook.ErrWriter.Name())
	// }
	// if hook.OtherWriter != nil {
	// 	OthWriterFilePath, _ = filepath.Abs(hook.OtherWriter.Name())
	// }
	for _, fileName := range files {
		if strings.HasSuffix(fileName, hook.LogConfig.keepSuffix) {
			continue
		}
		// fileAbsPath, _ := filepath.Abs(filepath.Join(dir, fileName))
		// fileAbsPath = strings.ToLower(fileAbsPath)
		tempFileName := strings.ToLower(fileName)
		if hook.ErrWriter != nil && tempFileName == strings.ToLower(filepath.Base(hook.ErrWriter.Name())) {
			continue
		}
		if hook.OtherWriter != nil && tempFileName == strings.ToLower(filepath.Base(hook.OtherWriter.Name())) {
			continue
		}
		// 最后修改时间
		fileInfo, err := os.Stat(filepath.Join(dir, fileName))
		if err != nil {
			logrus.Errorf("deleteOldLog os.Stat err:%v", err)
			continue
		}
		if time.Since(fileInfo.ModTime()).Hours() > float64(n*24) {
			err := os.Remove(filepath.Join(dir, fileName))
			if err != nil {
				logrus.Errorf("deleteOldLog os.Remove err:%v", err)
			}
		}
	}
}
