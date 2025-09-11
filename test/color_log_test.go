package main

import (
	"sandwich/log"
	"time"
)

// 测试颜色日志功能
func main() {
	println("=== Sandwich 颜色日志功能测试 ===")
	println()

	// 测试场景1: 启用颜色的日志输出
	println("场景1: 启用颜色的日志输出")
	log.InitLogWithColor(true)
	
	testLogOutput("启用颜色")
	println()

	// 等待一下，让输出更清晰
	time.Sleep(100 * time.Millisecond)

	// 测试场景2: 禁用颜色的日志输出
	println("场景2: 禁用颜色的日志输出")
	log.SetColorEnabled(false)
	
	testLogOutput("禁用颜色")
	println()

	// 测试场景3: 重新启用颜色
	println("场景3: 重新启用颜色")
	log.SetColorEnabled(true)
	
	testLogOutput("重新启用颜色")
	println()

	// 测试场景4: 使用Logger实例
	println("场景4: 使用Logger实例")
	logger := log.GetLogger()
	if logger != nil {
		logger.Info("使用Logger实例的Info日志")
		logger.ErrorF("使用Logger实例的ErrorF日志，当前时间：%s", time.Now().Format("15:04:05"))
		logger.Warn("使用Logger实例的Warn日志")
		logger.DebugF("使用Logger实例的DebugF日志，测试参数：%d", 12345)
	}
	println()

	// 测试场景5: 检查颜色状态
	println("场景5: 检查颜色状态")
	if log.IsColorEnabled() {
		log.Info("当前颜色功能已启用")
	} else {
		log.Info("当前颜色功能已禁用")
	}

	println("=== 测试完成 ===")
}

func testLogOutput(scenario string) {
	log.Info("这是INFO级别日志 -", scenario)
	log.Error("这是ERROR级别日志 -", scenario)
	log.Warn("这是WARN级别日志 -", scenario)
	log.Debug("这是DEBUG级别日志 -", scenario)
	
	log.InfoF("这是InfoF格式化日志 - %s，时间：%s", scenario, time.Now().Format("15:04:05"))
	log.ErrorF("这是ErrorF格式化日志 - %s，数字：%d", scenario, 404)
	log.Printf("这是普通Printf日志 - %s", scenario)
}