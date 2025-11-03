package main

import (
	"fmt"
	"time"

	"sandwich/log"
)

func main() {
	fmt.Println("=== Sandwich 队列日志系统演示 ===\n")

	// 1. 初始化日志系统
	fmt.Println("1. 初始化日志系统...")
	log.InitLogWithColor(true)
	defer log.CloseLogger()

	fmt.Printf("   异步模式: %t\n", log.IsAsyncEnabled())
	fmt.Printf("   队列容量: %d\n", log.GetQueueCapacity())
	fmt.Printf("   当前队列大小: %d\n\n", log.GetQueueSize())

	// 2. 基本日志测试
	fmt.Println("2. 测试基本日志功能...")
	log.Info("信息日志")
	log.Warn("警告日志")
	log.Error("错误日志")
	log.InfoF("格式化日志: %s = %d", "测试", 123)
	
	time.Sleep(50 * time.Millisecond) // 等待处理
	fmt.Printf("   处理后队列大小: %d\n\n", log.GetQueueSize())

	// 3. 性能测试
	fmt.Println("3. 性能测试 (100条日志)...")
	start := time.Now()
	for i := 0; i < 100; i++ {
		log.InfoF("性能测试 #%d", i)
	}
	elapsed := time.Since(start)
	fmt.Printf("   写入耗时: %v\n", elapsed)
	
	time.Sleep(100 * time.Millisecond) // 等待处理
	fmt.Printf("   处理后队列大小: %d\n\n", log.GetQueueSize())

	// 4. 测试异步开关
	fmt.Println("4. 测试异步开关...")
	log.SetAsyncEnabled(false)
	fmt.Printf("   关闭异步: %t\n", log.IsAsyncEnabled())
	log.Info("同步模式日志")
	
	log.SetAsyncEnabled(true)
	fmt.Printf("   重新启用异步: %t\n", log.IsAsyncEnabled())
	log.Info("异步模式日志")
	
	time.Sleep(50 * time.Millisecond)
	fmt.Printf("   队列大小: %d\n\n", log.GetQueueSize())

	// 5. 测试队列大小调整
	fmt.Println("5. 测试队列大小调整...")
	log.SetQueueSize(50)
	fmt.Printf("   新队列容量: %d\n", log.GetQueueCapacity())
	
	// 测试队列容量
	for i := 0; i < 20; i++ {
		log.InfoF("容量测试 #%d", i)
	}
	
	time.Sleep(100 * time.Millisecond)
	fmt.Printf("   处理后队列大小: %d\n\n", log.GetQueueSize())

	fmt.Println("=== 演示完成 ===")
}