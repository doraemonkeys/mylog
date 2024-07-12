package mylog

import (
	"reflect"
	"testing"
)

func Test_eliminateColor(t *testing.T) {
	tests := []struct {
		name string
		line []byte
		want []byte
	}{
		{"test1", []byte("\x1b[31m 红色 \x1b[0m"), []byte(" 红色 ")},
		{"test2", []byte("\x1b[31m 红色 \x1b[0m\x1b[31m 红色 \x1b[0m"), []byte(" 红色  红色 ")},
		{"test3", []byte("\x1b[31m 红色 \x1b[0m\x1b[31m 红色 \x1b[0m\x1b[31m 红色 \x1b[0m"), []byte(" 红色  红色  红色 ")},
		{"test4", []byte("你好\x1b[31m 红色 \x1b[0m"), []byte("你好 红色 ")},
		{"test5", []byte("你好\x1b[2m 不知道啥色 \x1b[0m"), []byte("你好 不知道啥色 ")},
		{"test6", []byte("你好\x1b[2m 不知道啥色 \x1b[0m 世界！！！"), []byte("你好 不知道啥色  世界！！！")},
		{"test7", []byte("你好\x1b[101m 不知道啥色 \x1b[0m 世界！！！"), []byte("你好 不知道啥色  世界！！！")},
		{"test8", []byte("你好\x1b[1001m 不知道啥色 \x1b[0m 世界！！！"), []byte("你好 不知道啥色  世界！！！")},
		{"test9", []byte("你好 红色，hello world"), []byte("你好 红色，hello world")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := eliminateColor(tt.line); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("eliminateColor() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getShortFileName(t *testing.T) {
	tests := []struct {
		file     string
		lineInfo string
		expected string
	}{
		{"C:/Users/Example/project/main.go", "10", "project/main.go:10"},
		{"C:\\Users\\Example\\project\\main.go", "20", "project/main.go:20"},
		{"/home/user/project/main.go", "30", "project/main.go:30"},
		{"project/main.go", "40", "project/main.go:40"},
		{"main.go", "50", "main.go:50"},
		{"main", "50", "main:50"},
		{"", "50", ":50"},
	}

	for _, test := range tests {
		result := getShortFileName(test.file, test.lineInfo)
		if result != test.expected {
			t.Errorf("For file %s and lineInfo %s, expected %s but got %s", test.file, test.lineInfo, test.expected, result)
		}
	}
}
