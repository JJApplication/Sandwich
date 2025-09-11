/*
Create: 2022/9/7
Project: Sandwich
Github: https://github.com/landers1037
Copyright Renj
*/

// Package breaker
package breaker

import (
	"sandwich/config"
	"sandwich/log"
	"sync"
	"time"
)

// 熔断控制器
// 在需要转发的微服务返回大量失败时，直接熔断当前的连接请求禁止客户端访问

var breaker *Breaker

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
	cf            *config.MiddlewareConfig
}

func NewBreaker() *Breaker {
	return &Breaker{
		serviceBucket: make(map[string]*BreakerBucket, 10),
		mux:           sync.Mutex{},
		cf:            config.Get().GetMiddle("breaker"),
	}
}

func (b *Breaker) Get(domain string) bool {
	sb, ok := b.serviceBucket[domain]
	if !ok {
		b.add(domain)
		return true
	}
	if len(sb.bucket) >= sb.errorConn {
		log.InfoF("[%s] breaker now is broken\n", domain)
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
		errorConn: b.cf.GetInt("limit"),
		bucket:    make(chan int, b.cf.GetInt("limit")),
	}
	b.mux.Unlock()
}

// Reset 自定重置
func (b *Breaker) Reset() {
	ticker := time.Tick(time.Duration(b.cf.GetInt("reset")) * time.Second)
	for range ticker {
		for domain, s := range b.serviceBucket {
			b.mux.Lock()
			s.bucket = make(chan int, b.cf.GetInt("limit"))
			log.InfoF("[%s] breaker now is reset\n", domain)
			b.mux.Unlock()
		}
	}
}

func (b *Breaker) Check() bool {
	return b.cf.Enabled
}

func Get(domain string) bool {
	return breaker.Get(domain)
}

func Set(domain string) bool {
	return breaker.Set(domain)
}
