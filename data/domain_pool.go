/*
Project: Sandwich domain_pool.go
Created: 2021/12/12 by Landers
*/

package data

import (
	"fmt"
	"sandwich/log"
	"sandwich/structure"
)

// 域名端口映射表
// 更加安全的端口映射表
var domainPool = structure.NewMap[[]int]()

func InitPool() {
	GetDataFromMongo()
}

func getDomainPort(host string) []int {
	if d, ok := domainPool.Get(host); ok {
		return d
	}
	return nil
}

// DomainReflect 将端口转换为ip地址 单机的ip都是127.0.0.1
func DomainReflect(host string) []string {
	group := getDomainPort(host)
	if len(group) == 0 {
		return nil
	}
	var dGroup []string
	for _, v := range group {
		dGroup = append(dGroup, fmt.Sprintf("127.0.0.1:%d", v))
	}

	return dGroup
}

func GetDataFromMongo() {
	data := getAppFromMongo()
	for _, v := range data {
		log.InfoF("find app [%s] from mongo, domain: [%s], ports: [%+v]\n",
			v.Meta.Name, v.Meta.Meta.Domain, v.Meta.RunData.Ports)
	}

	// 托管随机端口服务和固定端口服务
	for _, d := range data {
		log.InfoF("load [%s] to pool\n", d.Meta.Name)
		if d.Meta.Meta.Domain != "" && d.Meta.RunData.RandomPort {
			domainPool.Put(d.Meta.Meta.Domain, d.Meta.RunData.Ports)
		} else if d.Meta.Meta.Domain != "" && len(d.Meta.RunData.Ports) > 0 && !d.Meta.RunData.RandomPort {
			domainPool.Put(d.Meta.Meta.Domain, d.Meta.RunData.Ports)
		}
	}

	log.Info("domainPool is:")
	domainPool.Range(func(key string, value []int) bool {
		log.InfoF("[%s]: %#v\n", key, value)
		return true
	})
}
