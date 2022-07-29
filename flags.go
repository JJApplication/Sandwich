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
	MongoUrl = "mongodb://127.0.0.1:27017"
	DBName   = "ApolloMongo"
	SyncTime = time.Second * 60
)

func parseFlags() {
	mongourl := flag.String("mongo", "", "mongo db url")
	mongodb := flag.String("db", "", "mongo db name")
	syncTime := flag.Int("t", 60, "auto sync time")

	flag.Parse()

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
}
