/*
Project: Sandwich flow_control.go
Created: 2021/12/14 by Landers
*/

package main

import (
	"log"
)

// 流量控制

const (
	LIMIT_DURATION = 5
	// LIMIT 限制100/10s
	LIMIT = 100
	// ERROR_LIMIT 错误限制20/10s
	ERROR_LIMIT = 20
	// RESET 经过RESET * Duration次无请求后，从map中删除定时器
	RESET = 10
)

var limiter *ConnLimiter

func init() {
	limiter = NewConnLimiter(LIMIT)
}

type ConnLimiter struct {
	concurrentConn int
	bucket         chan int
}

func NewConnLimiter(c int) *ConnLimiter {
	return &ConnLimiter{
		concurrentConn: c,
		bucket:         make(chan int, c),
	}
}

// GetConn 获取桶令牌数量
func (cl *ConnLimiter) GetConn() bool {
	if len(cl.bucket) >= cl.concurrentConn {
		log.Println("reach limit")
		return false
	}
	cl.bucket <- 1
	return true
}

// ReleaseConn 释放所有桶
func (cl *ConnLimiter) ReleaseConn() {
	<-cl.bucket
	log.Println("new connection coming")
}
