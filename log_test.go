package mylog

import "testing"

func Test_timeStringCompare(t *testing.T) {
	type args struct {
		time1  string
		time2  string
		format string
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{"", args{"2020-01-01 12:00:00", "2019-01-01 12:00:00", "2006-01-02 15:04:05"}, 1},
		{"", args{"2020-01-01 12:10:00", "2020-01-01 12:00:00", "2006-01-02 15:04:05"}, 1},
		{"", args{"2010-01-01 12:00:00", "2019-01-01 12:00:00", "2006-01-02 15:04:05"}, -1},
		{"", args{"2010-01-01 12:00:00", "2010-01-01 12:00:09", "2006-01-02 15:04:05"}, -1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := timeStringCompare(tt.args.time1, tt.args.time2, tt.args.format); got != tt.want {
				t.Errorf("timeStringCompare() = %v, want %v", got, tt.want)
			}
		})
	}
}
