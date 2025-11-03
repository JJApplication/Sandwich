/*
Project: Sandwich queue_log_test.go
Created: 2024/01/01 by Assistant
*/

package main

import (
	"fmt"
	"time"

	"sandwich/log"
)

func main() {
	fmt.Println("=== 队列日志功能演示开始 ===")

	// 初始化日志系统（启用异步模式）
	log.InitLogWithColor(true)

	fmt.Printf("初始状态 - 异步模式: %t, 队列容量: %d\n", 
		log.IsAsyncEnabled(), log.GetQueueCapacity())

	// 测试基本日志功能
	fmt.Println("\n--- 测试基本日志功能 ---")
	log.Info("这是一条信息日志")
	log.Warn("这是一条警告日志")
	log.Error("这是一条错误日志")
	log.Debug("这是一条调试日志")

	// 测试格式化日志
	fmt.Println("\n--- 测试格式化日志 ---")
	log.InfoF("格式化信息日志: %s = %d", "计数", 42)
	log.WarnF("格式化警告日志: %.2f%%", 85.67)

	// 性能测试
	fmt.Println("\n--- 性能测试 (1000条日志) ---")
	start := time.Now()
	for i := 0; i < 1000; i++ {
		log.InfoF("性能测试日志 #%d", i)
	}
	elapsed := time.Since(start)
	fmt.Printf("写入1000条日志耗时: %v\n", elapsed)

	// 等待队列处理完成
	time.Sleep(100 * time.Millisecond)
	fmt.Printf("处理后队列大小: %d\n", log.GetQueueSize())

	// 测试异步开关
	fmt.Println("\n--- 测试异步开关 ---")
	log.SetAsyncEnabled(false)
	fmt.Printf("关闭异步后状态: %t\n", log.IsAsyncEnabled())
	log.Info("同步模式日志")

	log.SetAsyncEnabled(true)
	fmt.Printf("重新启用异步后状态: %t\n", log.IsAsyncEnabled())
	log.Info("异步模式日志")

	// 测试队列大小调整
	fmt.Println("\n--- 测试队列大小调整 ---")
	log.SetQueueSize(100)
	fmt.Printf("调整后队列容量: %d\n", log.GetQueueCapacity())

	fmt.Println("正在测试队列容量...")
	for i := 0; i < 150; i++ {
		log.InfoF("队列测试日志 #%d", i)
	}

	// 等待处理完成
	time.Sleep(200 * time.Millisecond)
	fmt.Printf("处理后队列大小: %d\n", log.GetQueueSize())

	// 关闭日志系统
	log.CloseLogger()

	fmt.Println("\n=== 队列日志功能测试完成 ===")
}