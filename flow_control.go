/*
Project: Sandwich flow_control.go
Created: 2021/12/14 by Landers
*/

package main

import (
	"sandwich/constant"
	"sandwich/log"
	"sync"
	"time"
)

// 流量控制
// 在访问的请求数超出限制时 禁止当前客户端请求
// 无法识别客户端所以是针对全局的请求限制

var limiter *ConnLimiter

func InitLimiter() {
	limiter = NewConnLimiter(constant.LIMIT)
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
		log.Info("limiter reach limit")
		return false
	}
	cl.bucket <- 1
	return true
}

// ReleaseConn 释放所有桶
func (cl *ConnLimiter) ReleaseConn() {
	<-cl.bucket
	log.Info("limiter new connection coming")
}

// AutoRelease 每5秒释放一次桶
func (cl *ConnLimiter) AutoRelease() {
	ticker := time.Tick(time.Duration(constant.RESET) * time.Second)
	for range ticker {
		if len(cl.bucket) >= cl.concurrentConn {
			cl.mux.Lock()
			cl.bucket = make(chan int, constant.LIMIT)
			cl.mux.Unlock()
			log.Info("limiter connection auto released")
		}
	}
}
