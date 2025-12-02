/*
Project: Sandwich log_message.go
Created: 2024/01/01 by Assistant
*/

package log

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
