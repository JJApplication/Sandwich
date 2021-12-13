/*
Project: Sandwich balance.go
Created: 2021/12/12 by Landers
*/

package main

import (
	"github.com/gookit/goutil/mathutil"
)

// 负载均衡
// 基于随机轮询的算法

func pickOne(hosts []string) string {
	if len(hosts) == 1 {
		return hosts[0]
	}
	all := len(hosts)
	i := mathutil.RandomInt(0, all-1)
	return hosts[i]
}
