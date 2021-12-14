/*
Project: Sandwich domain_pool.go
Created: 2021/12/12 by Landers
*/

package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/kamva/mgm/v3"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// 域名端口映射表
var domainPool map[string][]int

const (
	DBName   = "DirichletMongo"
	SyncTime = time.Second * 60
)

type DaoAPP struct {
	mgm.DefaultModel `bson:",inline"`
	App
}

type App struct {
	Name    string  `json:"name" validate:"required" bson:"name"`
	ID      string  `json:"id" validate:"required" bson:"id"`
	Meta    Meta    `json:"meta" bson:"meta"`
	RunData RunData `json:"run_data" bson:"run_data"`
}

type Meta struct {
	Author string `json:"author" bson:"author"`
	Domain string `json:"domain" bson:"domain"`
}

// RunData 运行时依赖
type RunData struct {
	Envs       []string `json:"envs" bson:"envs"` // just like `Name=Diri`
	Ports      []int    `json:"ports" bson:"ports"`
	RandomPort bool     `json:"random_port" bson:"random_port"` // if using random port
	Host       string   `json:"host" bson:"host"`               // always must be localhost
}

func (a *DaoAPP) CollectionName() string {
	return "app"
}

func init() {
	domainPool = make(map[string][]int, 1)
	mongo := flag.String("mongo", "mongodb://127.0.0.1:27017", "mongo db url")
	flag.Parse()

	err := mgm.SetDefaultConfig(&mgm.Config{CtxTimeout: 1 * time.Second}, DBName, options.Client().ApplyURI(*mongo))
	if err != nil {
		log.Printf("failed to connect to mongo: %s\n", err.Error())
		return
	}
	getDataFromMongo()
	go syncJob()
}

func getDomainPort(host string) []int {
	if d, ok := domainPool[host]; ok {
		return d
	}
	return []int{}
}

// 将端口转换为ip地址 单机的ip都是127.0.0.1
func domainReflect(host string) []string {
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

func getDataFromMongo() {
	var data []DaoAPP
	err := mgm.Coll(&DaoAPP{}).SimpleFind(&data, bson.M{})
	for _, v := range data {
		log.Printf("find app [%s] from mongo, domain: [%s], ports: [%+v]\n",
			v.Name, v.Meta.Domain, v.RunData.Ports)
	}

	if err != nil {
		log.Printf("get data from mongo failed: %s\n", err.Error())
		return
	}
	for _, d := range data {
		log.Printf("load [%s] to pool\n", d.Name)
		if d.Meta.Domain != "" && d.RunData.RandomPort {
			domainPool[d.Meta.Domain] = d.RunData.Ports
		}
	}

	log.Println("domainPool is:")
	for k, v := range domainPool {
		log.Printf("[%s]: %+v\n", k, v)
	}
}

// 异步从数据库同步端口数据
func syncJob() {
	tick := time.NewTicker(SyncTime)
	select {
	case <-tick.C:
		log.Println("sync job active")
		getDataFromMongo()
	}
}
