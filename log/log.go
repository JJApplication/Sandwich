/*
Project: Sandwich log.go
Created: 2021/12/12 by Landers
*/

package log

import (
	"fmt"
	golog "log"
	"os"
	"sandwich/config"
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
		gl: golog.New(os.Stdout, PREFIX, golog.LstdFlags|golog.Lshortfile),
	}
}

type Log struct {
	gl *golog.Logger
}

// 直接使用的单例
var (
	logger *Log
)

func GetLogger() *Log {
	return logger
}

func (l *Log) do(t string, v ...interface{}) {
	vv := append([]interface{}{t}, v...)
	l.gl.Println(vv...)
}

func (l *Log) doF(t string, fmt string, v ...interface{}) {
	f := t + fmt
	l.gl.Printf(f, v...)
}

func (l *Log) Println(v ...interface{}) {
	l.gl.Println(v...)
}

func (l *Log) Printf(format string, args ...interface{}) {
	l.gl.Printf(format, args...)
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

func Println(v ...interface{}) {
	logger.Println(v...)
}

func Printf(format string, args ...interface{}) {
	logger.Printf(format, args...)
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
	if config.Get().Debug {
		logger.DebugF(fmt, v...)
	}
}

func Debug(v ...interface{}) {
	if config.Get().Debug {
		logger.Debug(v...)
	}
}
