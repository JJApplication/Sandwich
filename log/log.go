/*
Project: Sandwich log.go
Created: 2021/12/12 by Landers
*/

package log

import (
	"fmt"
	golog "log"
	"os"
	"sandwich/constant"
)

var (
	PREFIX = fmt.Sprintf("[%s] ", constant.Sandwich)
	INFO   = "[INFO] "
	ERROR  = "[ERROR] "
	WARN   = "[WARN] "
	DEBUG  = "[DEBUG] "
)

func InitLog() {
	logger = &Log{
		l: golog.New(os.Stdout, PREFIX, golog.LstdFlags|golog.Lshortfile),
	}
}

type Log struct {
	l *golog.Logger
}

// 直接使用的单例
var (
	logger *Log
)

func (l *Log) do(t string, v ...interface{}) {
	vv := append([]interface{}{t}, v...)
	l.l.Println(vv...)
}

func (l *Log) doF(t string, fmt string, v ...interface{}) {
	f := t + fmt
	l.l.Printf(f, v...)
}

func (l *Log) Info(v ...interface{}) {
	l.do(INFO, v...)
}

func (l *Log) InfoF(format string, args ...interface{}) {
	l.doF(INFO, format, args...)
}

func (l *Log) Error(v ...interface{}) {
	l.do(ERROR, v...)
}

func (l *Log) ErrorF(format string, args ...interface{}) {
	l.doF(ERROR, format, args...)
}

func (l *Log) Warn(v ...interface{}) {
	l.do(WARN, v...)
}

func (l *Log) WarnF(format string, args ...interface{}) {
	l.doF(WARN, format, args...)
}

func (l *Log) Debug(v ...interface{}) {
	l.do(DEBUG, v...)
}

func (l *Log) DebugF(format string, args ...interface{}) {
	l.doF(DEBUG, format, args...)
}

func Info(v ...interface{}) {
	logger.Info(v...)
}

func InfoF(format string, args ...interface{}) {
	logger.InfoF(format, args...)
}

func Error(v ...interface{}) {
	logger.Error(v...)
}

func ErrorF(format string, args ...interface{}) {
	logger.ErrorF(format, args...)
}

func DebugF(fmt string, v ...interface{}) {
	if constant.Debug {
		logger.DebugF(fmt, v...)
	}
}

func Debug(v ...interface{}) {
	if constant.Debug {
		logger.Debug(v...)
	}
}
