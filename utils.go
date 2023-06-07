package mylog

import (
	"bytes"
	"os"
	"strings"

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
