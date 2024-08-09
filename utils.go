package mylog

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// D:\xxx\yyy\yourproject\pkg\log\log.go -> log\log.go:123
func getShortFileName(file string, lineInfo string) string {
	file = strings.Replace(file, "\\", "/", -1)
	n := 0
	for i := len(file) - 1; i >= 0; i-- {
		if file[i] == '/' {
			n++
			if n >= 2 {
				file = file[i+1:]
				break
			}
		}
	}
	return file + ":" + lineInfo
}

// 去除颜色
func eliminateColor(line []byte) []byte {
	//"\033[31m 红色 \033[0m"
	if bytes.Contains(line, []byte("\x1b[0m")) {
		buf := make([]byte, 0, len(line))
		start := 0
		var end int
		for {
			index := bytes.Index(line[start:], []byte("\x1b[")) //找到\x1b[的位置
			if index < 0 {
				buf = append(buf, line[start:]...)
				break
			}
			end = start + index
			buf = append(buf, line[start:end]...)
			// end的位置是\x1b的位置，end + 3 与 end + 4 一个是\x1b[0m，一个是\x1b[31m，以此类推，
			// 如果 end + 4 <= line.len()或者end + 5 <= line.len() 都不成立，
			// 说明字符串含有\x1b，但是\x1b[0m或者\x1b[31m不完整，或许不是颜色字符串。
			tempIndex := end + 3
			for tempIndex < len(line) && tempIndex <= end+6 {
				if line[tempIndex] == 'm' {
					start = tempIndex + 1
					break
				}
				tempIndex++
			}
			if tempIndex == len(line) || tempIndex > end+6 {
				logrus.Warnf("'m' not found in line[%d..%d]\n", end+3, end+6)
				return line
			}
			if start == len(line) {
				break
			}
			if start > len(line) {
				logrus.Warnf("start: %d > line.len(): %d\n", start, len(line))
				return line
			}
		}
		return buf
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
