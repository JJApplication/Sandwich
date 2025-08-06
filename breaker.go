/*
Create: 2022/9/7
Project: Sandwich
Github: https://github.com/landers1037
Copyright Renj
*/

// Package main
package main

import (
	"log"
	"sync"
	"time"
)

// 熔断控制器
// 在需要转发的微服务返回大量失败时，直接熔断当前的连接请求禁止客户端访问

var breaker *Breaker

var (
	BreakerLimit int // 限制内部错误的次数
	BreakerReset int // 默认的重置时间当服务down时 等待60s后重试
)

func InitBreaker() {
	breaker = NewBreaker()
	go breaker.Reset()
}

type BreakerBucket struct {
	errorConn int
	bucket    chan int
}

type Breaker struct {
	mux           sync.Mutex
	serviceBucket map[string]*BreakerBucket
}

func NewBreaker() *Breaker {
	return &Breaker{serviceBucket: make(map[string]*BreakerBucket, 10), mux: sync.Mutex{}}
}

func (b *Breaker) Get(domain string) bool {
	sb, ok := b.serviceBucket[domain]
	if !ok {
		b.add(domain)
		return true
	}
	if len(sb.bucket) >= sb.errorConn {
		log.Printf("[%s] breaker now is broken\n", domain)
		return false
	}
	return true
}

func (b *Breaker) Set(domain string) bool {
	sb, ok := b.serviceBucket[domain]
	if !ok {
		return true
	}
	if len(sb.bucket) < sb.errorConn {
		sb.bucket <- 1
		return true
	}

	return false
}

func (b *Breaker) add(domain string) {
	b.mux.Lock()
	b.serviceBucket[domain] = &BreakerBucket{
		errorConn: BreakerLimit,
		bucket:    make(chan int, BreakerLimit),
	}
	b.mux.Unlock()
}

// Reset 自定重置
func (b *Breaker) Reset() {
	ticker := time.Tick(time.Duration(BreakerReset) * time.Second)
	for range ticker {
		for domain, s := range b.serviceBucket {
			b.mux.Lock()
			s.bucket = make(chan int, BreakerLimit)
			log.Printf("[%s] breaker now is reset\n", domain)
			b.mux.Unlock()
		}
	}
}
