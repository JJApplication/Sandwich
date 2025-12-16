//go:build v1

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
	"sandwich/structure"
	"strings"
	"sync"
)

const (
	InfoLevel = iota
	WarnLevel
	ErrorLevel
	DebugLevel
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
		asyncEnabled: true, // 默认启用异步日志
		queueSize:    1000, // 默认队列大小
		stopChan:     make(chan struct{}),
	}
}

// InitLogWithColor 初始化日志并指定是否启用颜色
func InitLogWithColor(enableColor bool) {
	logger = &Log{
		gl:           golog.New(os.Stdout, PREFIX, golog.LstdFlags),
		ColorEnabled: enableColor,
		asyncEnabled: true, // 默认启用异步日志
		queueSize:    1000, // 默认队列大小
		stopChan:     make(chan struct{}),
	}
}

type Log struct {
	gl           *golog.Logger
	ColorEnabled bool // 是否启用颜色输出
	LogLevel     string
	LogFile      string
	// 队列相关字段
	queue        *structure.Queue // 日志消息队列
	asyncEnabled bool             // 是否启用异步日志
	queueSize    int              // 队列大小
	stopChan     chan struct{}    // 停止信号
	wg           sync.WaitGroup   // 等待组，用于优雅关闭
}

// 直接使用的单例
var (
	logger *Log
)

func GetLogger() *Log {
	return logger
}

// 获取带颜色的日志级别标签
func (l *Log) getColoredLevel(level int) string {
	if !l.ColorEnabled {
		switch level {
		case InfoLevel:
			return INFO
		case WarnLevel:
			return WARN
		case ErrorLevel:
			return ERROR
		case DebugLevel:
			return DEBUG
		default:
			return INFO
		}
	}

	switch level {
	case InfoLevel:
		return colorInfo(fmt.Sprintf("%-5s", "INFO"))
	case ErrorLevel:
		return colorError(fmt.Sprintf("%-5s", "ERROR"))
	case WarnLevel:
		return colorWarn(fmt.Sprintf("%-5s", "WARN"))
	case DebugLevel:
		return colorDebug(fmt.Sprintf("%-5s", "DEBUG"))
	default:
		return colorInfo(fmt.Sprintf("%-5s", "INFO"))
	}
}

// SetColorEnabled 设置是否启用颜色输出
func (l *Log) SetColorEnabled(enabled bool) {
	l.ColorEnabled = enabled
}

func (l *Log) SetLevel(level string) {
	l.LogLevel = level
}

func (l *Log) do(level int, v ...interface{}) {
	if !l.shouldLog(level) {
		return
	}

	coloredLevel := l.getColoredLevel(level)
	vv := append([]interface{}{coloredLevel}, v...)
	l.gl.Println(vv...)
}

func (l *Log) doF(level int, fmt string, v ...interface{}) {
	if !l.shouldLog(level) {
		return
	}
	coloredLevel := l.getColoredLevel(level)
	format := coloredLevel + " " + l.betterFmt(fmt)
	l.gl.Printf(format, v...)
}

func (l *Log) Println(v ...interface{}) {
	l.gl.Println(v...)
}

func (l *Log) Printf(format string, args ...interface{}) {
	l.gl.Printf(format, args...)
}

func (l *Log) Info(v ...interface{}) {
	l.do(InfoLevel, v...)
}

func (l *Log) InfoF(format string, args ...interface{}) {
	l.doF(InfoLevel, format, args...)
}

func (l *Log) Error(v ...interface{}) {
	l.do(ErrorLevel, v...)
}

func (l *Log) ErrorF(format string, args ...interface{}) {
	l.doF(ErrorLevel, format, args...)
}

func (l *Log) Warn(v ...interface{}) {
	l.do(WarnLevel, v...)
}

func (l *Log) WarnF(format string, args ...interface{}) {
	l.doF(WarnLevel, format, args...)
}

func (l *Log) Debug(v ...interface{}) {
	l.do(DebugLevel, v...)
}

func (l *Log) DebugF(format string, args ...interface{}) {
	l.doF(DebugLevel, format, args...)
}

func (l *Log) shouldLog(level int) bool {
	var myLevel int
	switch l.LogLevel {
	case "debug":
		myLevel = DebugLevel
	case "info":
		myLevel = InfoLevel
	case "warn":
		myLevel = WarnLevel
	case "error":
		myLevel = ErrorLevel
	default:
		myLevel = InfoLevel
	}
	return myLevel >= level
}

func (l *Log) betterFmt(fmt string) string {
	if strings.HasSuffix(fmt, "\n") {
		return fmt
	}
	return fmt + "\n"
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

func Warn(v ...interface{}) {
	logger.Warn(v...)
}

func WarnF(format string, args ...interface{}) {
	logger.WarnF(format, args...)
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
