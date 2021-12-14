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

var flowPool map[string]*sync.Map
var errPool map[string]*sync.Map

func init() {
	flowPool = make(map[string]*sync.Map, 1)
	errPool = make(map[string]*sync.Map, 1)
}

type pool struct {
	lock *sync.Mutex
	ticker *time.Ticker
	domain string
	count int
	isLimit bool
	reset int
}

const (
	LIMIT_DURATION = 5
	// LIMIT 限制100/10s
	LIMIT = 50
	// ERROR_LIMIT 错误限制20/10s
	ERROR_LIMIT = 20
	// RESET 经过RESET * Duration次无请求后，从map中删除定时器
	RESET = 10
)

// 流量控制
// 基于令牌桶的限流
func limitFlow(domain string, client string) bool {
	v, ok := flowPool[client]
	if ok {
		dm, ok := v.Load(domain)
		if ok {
			go startFlowTicker(client, domain)
			cli := dm.(pool)
			if cli.isLimit {
				// reset
				cli.lock.Lock()
				cli.count = 0
				cli.lock.Unlock()
				cli.ticker.Reset(time.Second * LIMIT_DURATION)
				v.Store(domain, cli)
				return true
			}
			// set count + 1
			cli.lock.Lock()
			defer cli.lock.Unlock()
			cli.count += 1
			v.Store(domain, cli)
			return false
		}
		// init pool
		p := pool{
			lock:    &sync.Mutex{},
			ticker:  time.NewTicker(time.Second * LIMIT_DURATION),
			domain:  domain,
			count:   0,
			isLimit: false,
		}

		v.Store(domain, p)
		return false
	}

	tmp := &sync.Map{}
	flowPool[client] = tmp
	return false
}

// 启动定时器
// 当入出为0时 表明暂时无请求，删除定时器
func startFlowTicker(client, domain string) {
	v, ok := flowPool[client]
	var lastCount int
	if ok {
		pm, ok := v.Load(domain)
		if ok {
			p := pm.(pool)
			go func() {
				if p.ticker != nil {
					select {
					case <-p.ticker.C:
						dp, _ := v.Load(domain)
						dpp := dp.(pool)
						if dpp.count >= LIMIT {
							dpp.lock.Lock()
							dpp.isLimit = true
							dpp.lock.Unlock()
							v.Store(domain, dpp)
						} else {
							dpp.lock.Lock()
							dpp.isLimit = false
							dpp.lock.Unlock()
							v.Store(domain, dpp)
						}
						lastCount = dpp.count
						// 是否删除定时器
						if dpp.count == lastCount {
							if dpp.reset >= RESET {
								log.Printf("reach timeout, start to delete ticker on [%s]\n", domain)
								dpp.ticker.Stop()
								defer v.Delete(domain)
							} else {
								dpp.reset += 1
								v.Store(domain, dpp)
							}
						}
					}
				}
			}()
		}
	}
}

// 基于错误次数的限制
// 单位时间内的代理错误次数过多，会直接隔离服务不再进行跳转
func limitErr(domain string, client string) bool {
	return true
}

func addErrCount(domain string, client string) {

}