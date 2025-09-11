/*
Project: Sandwich log.go
Created: 2021/12/12 by Landers
*/

package log

import (
	"fmt"
	"github.com/fatih/color"
	golog "log"
	"os"
	"sandwich/config"
	"sandwich/constant"
)

var (
	PREFIX = fmt.Sprintf("[%s] ", constant.Sandwich)
	INFO   = fmt.Sprintf("[%-5s]", "INFO")
	ERROR  = fmt.Sprintf("[%-5s]", "ERROR")
	WARN   = fmt.Sprintf("[%-5s]", "WARN")
	DEBUG  = fmt.Sprintf("[%-5s]", "DEBUG")

	// 颜色日志级别
	colorInfo  = color.New(color.BgGreen, color.FgWhite).SprintFunc()  // 绿色背景白色文字
	colorError = color.New(color.BgRed, color.FgWhite).SprintFunc()    // 红色背景白色文字
	colorWarn  = color.New(color.BgYellow, color.FgBlack).SprintFunc() // 黄色背景黑色文字
	colorDebug = color.New(color.BgCyan, color.FgWhite).SprintFunc()   // 青色背景白色文字
)

func InitLog() {
	logger = &Log{
		gl:           golog.New(os.Stdout, PREFIX, golog.LstdFlags),
		ColorEnabled: true, // 默认启用颜色
	}
}

// InitLogWithColor 初始化日志并指定是否启用颜色
func InitLogWithColor(enableColor bool) {
	logger = &Log{
		gl:           golog.New(os.Stdout, PREFIX, golog.LstdFlags),
		ColorEnabled: enableColor,
	}
}

type Log struct {
	gl           *golog.Logger
	ColorEnabled bool // 是否启用颜色输出
	LogLevel     string
	LogFile      string
}

// 直接使用的单例
var (
	logger *Log
)

func GetLogger() *Log {
	return logger
}

// 获取带颜色的日志级别标签
func (l *Log) getColoredLevel(level string) string {
	if !l.ColorEnabled {
		return level
	}

	switch level {
	case INFO:
		return colorInfo(fmt.Sprintf("%-5s", "INFO"))
	case ERROR:
		return colorError(fmt.Sprintf("%-5s", "ERROR"))
	case WARN:
		return colorWarn(fmt.Sprintf("%-5s", "WARN"))
	case DEBUG:
		return colorDebug(fmt.Sprintf("%-5s", "DEBUG"))
	default:
		return level
	}
}

// SetColorEnabled 设置是否启用颜色输出
func (l *Log) SetColorEnabled(enabled bool) {
	l.ColorEnabled = enabled
}

func (l *Log) SetLevel(level string) {
	l.LogLevel = level
}

func (l *Log) do(t string, v ...interface{}) {
	coloredLevel := l.getColoredLevel(t)
	vv := append([]interface{}{coloredLevel}, v...)
	l.gl.Println(vv...)
}

func (l *Log) doF(t string, fmt string, v ...interface{}) {
	coloredLevel := l.getColoredLevel(t)
	f := coloredLevel + " " + fmt
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

func Reload(config config.LogConfig) {
	SetColorEnabled(config.Color)
	SetLogLevel(config.LogLevel)
}

// SetLogLevel 设置日志级别
func SetLogLevel(level string) {
	if logger != nil {
		logger.SetLevel(level)
	}
}

// SetColorEnabled 设置全局颜色输出
func SetColorEnabled(enabled bool) {
	if logger != nil {
		logger.SetColorEnabled(enabled)
	}
}

// IsColorEnabled 获取当前颜色设置状态
func IsColorEnabled() bool {
	if logger != nil {
		return logger.ColorEnabled
	}
	return false
}
