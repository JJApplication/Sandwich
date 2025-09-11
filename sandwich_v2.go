package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sandwich/app"
	"sandwich/breaker"
	"sandwich/cache"
	"sandwich/connector"
	"sandwich/data"
	"sandwich/flow"
	"sandwich/jobs"
	"syscall"
	"time"

	"sandwich/config"
	"sandwich/log"
)

// main 主函数
func main() {
	// 解析命令行参数
	var (
		configPath = flag.String("config", "", "配置文件路径")
		generate   = flag.Bool("gen", false, "生成配置文件")
		version    = flag.Bool("version", false, "显示版本信息")
		help       = flag.Bool("help", false, "显示帮助信息")
	)
	flag.Parse()

	// 显示版本信息
	if *version {
		fmt.Println("Sandwich Proxy v2.0.0")
		fmt.Println("支持多端口监听、HTTP/3 和 WebSocket 协议")
		return
	}

	// 显示帮助信息
	if *help {
		flag.Usage()
		fmt.Println("\n示例:")
		fmt.Println("  sandwich --config=config.yaml")
		fmt.Println("  sandwich --version")
		return
	}

	if *generate {
		if err := config.CreateConfig(); err != nil {
			fmt.Println(err)
		}
		return
	}
	
	log.InitLog()

	// 如果指定了配置文件，检查文件是否存在
	if *configPath != "" {
		if _, err := os.Stat(*configPath); os.IsNotExist(err) {
			log.Printf("配置文件不存在: %s", *configPath)
			return
		}
		// 转换为绝对路径
		absPath, err := filepath.Abs(*configPath)
		if err != nil {
			log.Printf("获取配置文件绝对路径失败: %v", err)
			return
		}
		*configPath = absPath
	}

	// 创建应用程序实例
	sandwichApp := app.NewApplication(log.GetLogger())

	// 初始化应用程序
	if err := sandwichApp.Initialize(*configPath); err != nil {
		log.Printf("初始化应用程序失败: %v", err)
		return
	}

	// 初始化应用
	data.InitMongo()
	data.InitInflux()
	data.InitPool()
	// load domain map
	cache.InitNoEngineDomainMap()
	// load helios config
	connector.InitHeliosConfig()

	// init gzip cache for static pages
	cache.InitGzipCache()

	// start sync jobs
	jobs.InitSyncJobs()

	// init worker
	breaker.InitBreaker()
	flow.InitLimiter()

	// 设置信号处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 启动应用程序
	if err := sandwichApp.Start(); err != nil {
		log.ErrorF("启动应用程序失败: %v", err)
		return
	}

	// 等待信号
	sig := <-sigChan
	log.Printf("收到信号: %v，正在优雅关闭...", sig)

	// 创建关闭超时上下文
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// 在 goroutine 中执行关闭操作
	done := make(chan error, 1)
	go func() {
		done <- sandwichApp.Stop()
	}()

	// 等待关闭完成或超时
	select {
	case err := <-done:
		if err != nil {
			log.Printf("关闭应用程序时发生错误: %v", err)
			os.Exit(1)
		}
		log.Println("应用程序已优雅关闭")
	case <-shutdownCtx.Done():
		log.Println("关闭超时，强制退出")
		os.Exit(1)
	}
}
