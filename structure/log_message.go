/*
Project: Sandwich log_message.go
Created: 2024/01/01 by Assistant
*/

package structure

import (
	"time"
)

// LogMessage 日志消息结构体，用于在队列中传递日志信息
type LogMessage struct {
	Level     int           // 日志级别
	Message   string        // 格式化后的消息内容
	Args      []interface{} // 原始参数
	Format    string        // 格式化字符串（如果是格式化日志）
	Timestamp time.Time     // 日志时间戳
	IsFormat  bool          // 是否为格式化日志
}

// NewLogMessage 创建新的日志消息
func NewLogMessage(level int, message string, args []interface{}) *LogMessage {
	return &LogMessage{
		Level:     level,
		Message:   message,
		Args:      args,
		Timestamp: time.Now(),
		IsFormat:  false,
	}
}

// NewLogMessageF 创建新的格式化日志消息
func NewLogMessageF(level int, format string, args []interface{}) *LogMessage {
	return &LogMessage{
		Level:     level,
		Format:    format,
		Args:      args,
		Timestamp: time.Now(),
		IsFormat:  true,
	}
}

// LogLevel 日志级别常量
const (
	LogLevelInfo = iota
	LogLevelWarn
	LogLevelError
	LogLevelDebug
)

// LogLevelNames 日志级别名称映射
var LogLevelNames = map[int]string{
	LogLevelInfo:  "INFO",
	LogLevelWarn:  "WARN",
	LogLevelError: "ERROR",
	LogLevelDebug: "DEBUG",
}