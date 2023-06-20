package mylog

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// D:\xxx\yyy\yourproject\pkg\log\log.go -> pkg\log\log.go:123
func getShortFileName(file string, lineInfo string) string {
	file = strings.Replace(file, "\\", "/", -1)

	// if 做一次快速简单的判断
	if strings.Contains(file, "/") {
		env, _ := os.Getwd()
		env = strings.Replace(env, "\\", "/", -1)
		if len(file) <= len(env) || !isChildDir(env, file) {
			return file + ":" + lineInfo
		}
		file = file[len(env):]
		if file[0] == '/' {
			file = file[1:]
		}
		file = file + ":" + lineInfo
	}
	return file
}

// 判断child是否是parent的子文件夹(为了性能只是简单的判断前缀，需要保证路径分隔符一致)
func isChildDir(parent, child string) bool {
	parent = strings.ToUpper(parent)
	child = strings.ToUpper(child)
	return strings.HasPrefix(child, parent)
}

// 去除颜色
func eliminateColor(line []byte) []byte {
	//"\033[31m 红色 \033[0m"
	if bytes.Contains(line, []byte("\x1b[0m")) {
		line = bytes.ReplaceAll(line, []byte("\x1b[0m"), []byte(""))

		index := bytes.Index(line, []byte("\x1b[")) //找到\x1b[的位置
		for index >= 0 && index+5 < len(line) {
			line = bytes.ReplaceAll(line, line[index:index+5], []byte("")) //删除\x1b[31m
			index = bytes.Index(line, []byte("\x1b["))
		}
	}
	return line
}

// panic,fatal,error,warn,info,debug,trace
// 默认info
func PraseLevel(level string) logrus.Level {
	level = strings.ToLower(level)
	switch level {
	case "trace":
		return logrus.TraceLevel
	case "debug":
		return logrus.DebugLevel
	case "info":
		return logrus.InfoLevel
	case "warn":
		return logrus.WarnLevel
	case "error":
		return logrus.ErrorLevel
	case "fatal":
		return logrus.FatalLevel
	case "panic":
		return logrus.PanicLevel
	default:
		return logrus.InfoLevel
	}
}

// 替换文件名中的非法字符为下划线
func makeFileNameLegal(s string) string {
	s = strings.ReplaceAll(s, "/", "_")
	s = strings.ReplaceAll(s, "\\", "_")
	s = strings.ReplaceAll(s, ":", "_")
	s = strings.ReplaceAll(s, "*", "_")
	s = strings.ReplaceAll(s, "?", "_")
	s = strings.ReplaceAll(s, "\"", "_")
	s = strings.ReplaceAll(s, "<", "_")
	s = strings.ReplaceAll(s, ">", "_")
	s = strings.ReplaceAll(s, "|", "_")
	return s
}

// 获取path路径下的文件夹名称
func getFolderNamesInPath(path string) ([]string, error) {
	if path == "" {
		path = "."
	}
	DirEntry, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	var dirs []string
	for _, v := range DirEntry {
		if v.IsDir() {
			dirs = append(dirs, v.Name())
		}
	}
	return dirs, nil
}

// 获取path下所有文件名称(含后缀)
func getFileNmaesInPath(path string) ([]string, error) {
	if path == "" {
		path = "."
	}
	DirEntry, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, v := range DirEntry {
		if !v.IsDir() {
			files = append(files, v.Name())
		}
	}
	return files, nil
}
func isEmptyDir(dir string) bool {
	DirEntry, err := os.ReadDir(dir)
	if err != nil {
		logrus.Errorf("dir empty check error:%v", err)
		return false
	}
	return len(DirEntry) == 0
}
func getFolderSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		size += info.Size()
		return nil
	})
	return size, err
}

// time1 > time2 return 1.
// time1 < time2 return -1.
// return 0 if error or equal.
func timeStringCompare(time1, time2, format string) int {
	t1, err := time.ParseInLocation(format, time1, time.Local)
	if err != nil {
		return 0
	}
	t2, err := time.ParseInLocation(format, time2, time.Local)
	if err != nil {
		return 0
	}
	if t1.After(t2) {
		return 1
	}
	if t1.Before(t2) {
		return -1
	}
	return 0
}

// 仅在有写入时才创建文件
type LazyFileWriter struct {
	filePath string
	file     *os.File
}

func (w *LazyFileWriter) Write(p []byte) (n int, err error) {
	if w.file == nil {
		w.file, err = os.OpenFile(w.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return 0, err
		}
	}
	return w.file.Write(p)
}

func (w *LazyFileWriter) Close() error {
	if w.file != nil {
		return w.file.Close()
	}
	return nil
}

func (w *LazyFileWriter) Seek(offset int64, whence int) (int64, error) {
	if w.file == nil {
		return 0, errors.New("file not created")
	}
	return w.file.Seek(offset, whence)
}

// Name returns the name of the file as presented to Open.
func (w *LazyFileWriter) Name() string {
	if w.file != nil {
		return w.file.Name()
	}
	return filepath.Base(w.filePath)
}

// 是否已经创建了文件
func (w *LazyFileWriter) IsCreated() bool {
	return w.file != nil
}

func NewLazyFileWriter(filePath string) *LazyFileWriter {
	return &LazyFileWriter{filePath: filePath}
}

func NewLazyFileWriterWithFile(filePath string, file *os.File) *LazyFileWriter {
	return &LazyFileWriter{filePath: filePath, file: file}
}
