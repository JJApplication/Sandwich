package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sandwich/config"
	"sandwich/utils"
	"time"
)

// 测试TraceID生成和传递功能
func main() {
	fmt.Println("=== Sandwich TraceID 功能测试 ===")
	fmt.Println()

	// 模拟配置
	mockConfig := &config.Config{
		ProxyHeader: config.ProxyHeader{
			TraceId: "X-Gateway-Trace-Id",
		},
	}

	// 设置全局配置（这里只是为了测试，实际使用中配置已经加载）
	_ = mockConfig

	fmt.Println("测试场景:")
	fmt.Println()

	// 测试场景1: 请求中没有TraceID，应该生成新的
	fmt.Println("场景1: 请求中没有TraceID")
	testNewTraceId()
	fmt.Println()

	// 测试场景2: 请求中已有TraceID，应该传递现有的
	fmt.Println("场景2: 请求中已有TraceID")
	testExistingTraceId()
	fmt.Println()

	// 测试场景3: 测试TraceID的唯一性
	fmt.Println("场景3: 测试TraceID唯一性")
	testTraceIdUniqueness()
	fmt.Println()

	fmt.Println("=== 测试完成 ===")
}

func testNewTraceId() {
	// 创建没有TraceID的请求
	req := httptest.NewRequest("GET", "/test", nil)
	
	// 创建响应
	w := httptest.NewRecorder()
	resp := &http.Response{
		Header:  make(http.Header),
		Request: req,
	}

	fmt.Printf("  请求前 - TraceID: %s\n", req.Header.Get("X-Gateway-Trace-Id"))
	
	// 调用AddTrace函数
	utils.AddTrace(resp)
	
	traceId := resp.Header.Get("X-Gateway-Trace-Id")
	fmt.Printf("  响应后 - TraceID: %s\n", traceId)
	
	if traceId != "" {
		fmt.Printf("  结果: ✓ 成功生成新的TraceID\n")
		fmt.Printf("  格式: %s (时间戳-随机数)\n", traceId)
	} else {
		fmt.Printf("  结果: ✗ 未能生成TraceID\n")
	}
}

func testExistingTraceId() {
	// 创建带有TraceID的请求
	req := httptest.NewRequest("GET", "/test", nil)
	existingTraceId := "existing-trace-12345"
	req.Header.Set("X-Gateway-Trace-Id", existingTraceId)
	
	// 创建响应
	resp := &http.Response{
		Header:  make(http.Header),
		Request: req,
	}

	fmt.Printf("  请求前 - TraceID: %s\n", existingTraceId)
	
	// 调用AddTrace函数
	utils.AddTrace(resp)
	
	responseTraceId := resp.Header.Get("X-Gateway-Trace-Id")
	fmt.Printf("  响应后 - TraceID: %s\n", responseTraceId)
	
	if responseTraceId == existingTraceId {
		fmt.Printf("  结果: ✓ 成功传递现有TraceID\n")
	} else {
		fmt.Printf("  结果: ✗ TraceID传递失败\n")
	}
}

func testTraceIdUniqueness() {
	traceIds := make(map[string]bool)
	duplicateCount := 0
	testCount := 100

	fmt.Printf("  生成 %d 个TraceID测试唯一性:\n", testCount)

	for i := 0; i < testCount; i++ {
		// 创建请求和响应
		req := httptest.NewRequest("GET", "/test", nil)
		resp := &http.Response{
			Header:  make(http.Header),
			Request: req,
		}

		// 生成TraceID
		utils.AddTrace(resp)
		traceId := resp.Header.Get("X-Gateway-Trace-Id")

		// 检查唯一性
		if traceIds[traceId] {
			duplicateCount++
			fmt.Printf("    发现重复: %s\n", traceId)
		} else {
			traceIds[traceId] = true
		}

		// 显示前几个生成的TraceID作为示例
		if i < 5 {
			fmt.Printf("    [%02d] %s\n", i+1, traceId)
		}

		// 添加小延迟以确保时间戳不同
		if i%10 == 0 {
			time.Sleep(1 * time.Millisecond)
		}
	}

	fmt.Printf("  结果统计:\n")
	fmt.Printf("    总数: %d\n", testCount)
	fmt.Printf("    唯一: %d\n", len(traceIds))
	fmt.Printf("    重复: %d\n", duplicateCount)
	
	if duplicateCount == 0 {
		fmt.Printf("  结果: ✓ 所有TraceID都是唯一的\n")
	} else {
		fmt.Printf("  结果: ⚠ 发现 %d 个重复的TraceID\n", duplicateCount)
	}
}