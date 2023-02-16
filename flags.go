/*
Create: 2022/7/29
Project: Sandwich
Github: https://github.com/landers1037
Copyright Renj
*/

// Package main
package main

import (
	"flag"
	"os"
	"time"
)

// 读取flags参数
var (
	Port         = "8888"
	MongoUrl     = "mongodb://127.0.0.1:27017"
	DBName       = "ApolloMongo"
	SyncTime     = time.Second * 60
	InfluxUrl    = "http://127.0.0.1:8086"
	InfluxToken  = ""
	InfluxOrg    = "renj"
	InfluxBucket = "sandwich"
	InfluxPwd    = os.Getenv("InfluxPwd")
	EnableInflux *bool
	CacheSize    = 10
)

func parseFlags() {
	port := flag.String("port", "", "port")
	mongourl := flag.String("mongo", "", "mongo db url")
	mongodb := flag.String("db", "", "mongo db name")
	syncTime := flag.Int("t", 60, "auto sync time")
	influxUrl := flag.String("influx", "", "influx db url")
	influxToken := flag.String("token", "", "influx db token")
	EnableInflux = flag.Bool("enable", false, "enable influx")
	cache := flag.Int("size", CacheSize, "cache size[mb]")
	flag.Parse()

	if *port != "" {
		Port = *port
	}

	if *mongourl != "" {
		MongoUrl = *mongourl
	}
	// 从环境变量读取数据库
	name := os.Getenv("mongo")
	if name != "" {
		DBName = name
	}

	if *mongodb != "" {
		DBName = *mongodb
	}

	if *syncTime > 0 {
		SyncTime = time.Duration(*syncTime) * time.Second
	}

	if *influxUrl != "" {
		InfluxUrl = *influxUrl
	}

	if *influxToken != "" {
		InfluxToken = *influxToken
	}

	if *cache != 0 {
		CacheSize = *cache
	}
}
