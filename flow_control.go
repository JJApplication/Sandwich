/*
Project: Sandwich flow_control.go
Created: 2021/12/14 by Landers
*/

package main

import (
	"log"
	"sync"
	"time"
)

// 流量控制
// 在访问的请求数超出限制时 禁止当前客户端请求
// 无法识别客户端所以是针对全局的请求限制

const (
	// LIMIT 限制100/10s
	LIMIT = 100
	// RESET 经过RESET * Duration次无请求后，从map中删除定时器
	RESET = 10
)

var limiter *ConnLimiter

func init() {
	limiter = NewConnLimiter(LIMIT)
	go limiter.AutoRelease()
}

type ConnLimiter struct {
	concurrentConn int
	bucket         chan int
	mux            sync.Mutex
}

func NewConnLimiter(c int) *ConnLimiter {
	return &ConnLimiter{
		concurrentConn: c,
		bucket:         make(chan int, c),
		mux:            sync.Mutex{},
	}
}

// GetConn 获取桶令牌数量
func (cl *ConnLimiter) GetConn() bool {
	if len(cl.bucket) >= cl.concurrentConn {
		log.Println("limiter reach limit")
		return false
	}
	cl.bucket <- 1
	return true
}

// ReleaseConn 释放所有桶
func (cl *ConnLimiter) ReleaseConn() {
	<-cl.bucket
	log.Println("limiter new connection coming")
}

// AutoRelease 每5秒释放一次桶
func (cl *ConnLimiter) AutoRelease() {
	ticker := time.Tick(RESET * time.Second)
	for range ticker {
		if len(cl.bucket) >= cl.concurrentConn {
			cl.mux.Lock()
			cl.bucket = make(chan int, LIMIT)
			cl.mux.Unlock()
			log.Println("limiter connection auto released")
		}
	}
}
