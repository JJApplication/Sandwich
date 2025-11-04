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
	"time"
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

	LevelMap = map[int]string{
		InfoLevel:  INFO,
		WarnLevel:  WARN,
		ErrorLevel: ERROR,
		DebugLevel: DEBUG,
	}
)

func InitLog() {
	logger = &Log{
		gl:           golog.New(os.Stdout, PREFIX, golog.LstdFlags),
		ColorEnabled: true, // 默认启用颜色
		asyncEnabled: true, // 默认启用异步日志
		queueSize:    1000, // 默认队列大小
		stopChan:     make(chan struct{}),
	}
	logger.initQueue()
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
	logger.initQueue()
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
		return LevelMap[level]
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
		return LevelMap[level]
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

	// 创建日志消息并加入队列
	msg := NewLogMessage(level, "", v)
	l.enqueueLogMessage(msg)
}

func (l *Log) doF(level int, fmt string, v ...interface{}) {
	if !l.shouldLog(level) {
		return
	}

	// 创建格式化日志消息并加入队列
	msg := NewLogMessageF(level, fmt, v)
	l.enqueueLogMessage(msg)
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

// initQueue 初始化日志队列和异步处理goroutine
func (l *Log) initQueue() {
	if l.asyncEnabled {
		l.queue = structure.NewQueue(l.queueSize)
		l.wg.Add(1)
		go l.processLogQueue()
	}
}

// processLogQueue 异步处理日志队列中的消息
func (l *Log) processLogQueue() {
	defer l.wg.Done()

	for {
		select {
		case <-l.stopChan:
			// 处理剩余的日志消息
			l.flushQueue()
			return
		default:
			// 检查队列是否已关闭
			if l.queue.IsClosed() {
				// 处理剩余的日志消息
				l.flushQueue()
				return
			}

			// 从队列中取出日志消息并处理
			if msg := l.queue.DequeueBlocking(100 * time.Millisecond); msg != nil {
				if logMsg, ok := msg.(*LogMessage); ok {
					l.processLogMessage(logMsg)
				}
			}
		}
	}
}

// processLogMessage 处理单个日志消息
func (l *Log) processLogMessage(msg *LogMessage) {
	if !l.shouldLog(msg.Level) {
		return
	}

	coloredLevel := l.getColoredLevel(msg.Level)

	if msg.IsFormat {
		// 格式化日志
		format := coloredLevel + " " + l.betterFmt(msg.Format)
		l.gl.Printf(format, msg.Args...)
	} else {
		// 普通日志
		vv := append([]interface{}{coloredLevel}, msg.Args...)
		l.gl.Println(vv...)
	}
}

// flushQueue 刷新队列中剩余的日志消息
func (l *Log) flushQueue() {
	if l.queue == nil {
		return
	}

	for !l.queue.IsEmpty() {
		if msg := l.queue.Dequeue(); msg != nil {
			if logMsg, ok := msg.(*LogMessage); ok {
				l.processLogMessage(logMsg)
			}
		}
	}
}

// enqueueLogMessage 将日志消息加入队列
func (l *Log) enqueueLogMessage(msg *LogMessage) {
	if !l.asyncEnabled || l.queue == nil {
		// 如果异步未启用，直接同步处理
		l.processLogMessage(msg)
		return
	}

	// 尝试加入队列，如果队列满了则直接输出（防止阻塞）
	if !l.queue.Enqueue(msg) {
		// 队列满了，直接同步输出
		l.processLogMessage(msg)
	}
}

// SetAsyncEnabled 设置是否启用异步日志
func (l *Log) SetAsyncEnabled(enabled bool) {
	if l.asyncEnabled == enabled {
		return
	}

	if l.asyncEnabled && !enabled {
		// 关闭异步模式 - 先关闭队列，然后等待goroutine结束
		if l.queue != nil {
			l.queue.Close()
		}
		if l.stopChan != nil {
			close(l.stopChan)
		}
		l.wg.Wait()
	}

	l.asyncEnabled = enabled
	if enabled {
		l.stopChan = make(chan struct{})
		l.initQueue()
	}
}

// SetQueueSize 设置队列大小（需要重新初始化）
func (l *Log) SetQueueSize(size int) {
	if l.queueSize == size {
		return
	}

	wasAsync := l.asyncEnabled
	if wasAsync {
		// 优雅关闭异步模式 - 先关闭队列，然后等待goroutine结束
		if l.queue != nil {
			l.queue.Close()
		}
		if l.stopChan != nil {
			close(l.stopChan)
		}
		l.wg.Wait()
	}

	l.queueSize = size
	if wasAsync {
		l.asyncEnabled = true
		l.stopChan = make(chan struct{})
		l.initQueue()
	}
}

// Close 关闭日志系统
func (l *Log) Close() {
	if l.asyncEnabled && l.queue != nil {
		// 先关闭队列，唤醒等待的goroutine
		l.queue.Close()
		// 然后关闭停止通道
		close(l.stopChan)
		// 等待goroutine结束
		l.wg.Wait()
		l.asyncEnabled = false
	}
}

// GetQueueSize 获取当前队列大小
func (l *Log) GetQueueSize() int {
	if l.queue != nil {
		return l.queue.Size()
	}
	return 0
}

// GetQueueCapacity 获取队列容量
func (l *Log) GetQueueCapacity() int {
	return l.queueSize
}

// IsAsyncEnabled 检查是否启用异步日志
func (l *Log) IsAsyncEnabled() bool {
	return l.asyncEnabled
}

// 全局异步日志管理函数

// SetAsyncEnabled 设置全局异步日志开关
func SetAsyncEnabled(enabled bool) {
	if logger != nil {
		logger.SetAsyncEnabled(enabled)
	}
}

// SetQueueSize 设置全局队列大小
func SetQueueSize(size int) {
	if logger != nil {
		logger.SetQueueSize(size)
	}
}

// CloseLogger 优雅关闭全局日志系统
func CloseLogger() {
	if logger != nil {
		logger.Close()
	}
}

// GetQueueSize 获取全局队列当前大小
func GetQueueSize() int {
	if logger != nil {
		return logger.GetQueueSize()
	}
	return 0
}

// GetQueueCapacity 获取全局队列容量
func GetQueueCapacity() int {
	if logger != nil {
		return logger.GetQueueCapacity()
	}
	return 0
}

// IsAsyncEnabled 检查全局是否启用异步日志
func IsAsyncEnabled() bool {
	if logger != nil {
		return logger.IsAsyncEnabled()
	}
	return false
}
