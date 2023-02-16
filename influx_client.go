/*
Create: 2022/9/7
Project: Sandwich
Github: https://github.com/landers1037
Copyright Renj
*/

// Package main
package main

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
)

// influx db client

// 默认的时序数据库名称

const (
	DefaultPwd      = "12345678"
	SandwichMeasure = "sandwich-request"
	// 标签信息 即每个度量携带的标签
	FieldDomain = "domain"
	// 字段信息 即需要记录的字段
	TagDomain        = "domain"
	TagClient        = "client"
	TagUrl           = "url"
	TagRefer         = "refer"
	TagMethod        = "method"
	TagContendLength = "content-length"
	TagStat          = "stat"
)

const (
	StatPass     = "pass"
	StatAbort    = "abort"
	StatBreak    = "break"
	StatNotFound = "not-found"
)

var influxC influxdb2.Client
var writeApi api.WriteAPI

func initInflux() {
	if !preCheck() {
		return
	}
	log.Println("init influxdb")
	if InfluxToken != "" {
		influxC = influxdb2.NewClient(InfluxUrl, InfluxToken)
	} else {
		influxC = influxdb2.NewClient(InfluxUrl, InfluxToken)
		if InfluxPwd == "" {
			InfluxPwd = DefaultPwd
		}
		res, err := influxC.Setup(context.Background(), InfluxOrg, InfluxPwd, InfluxOrg, InfluxBucket, 0)
		if err != nil {
			log.Printf("setup Error: %s\n", err.Error())
		} else {
			log.Printf("setup finished,authtoken: %s\n", *res.Auth.Token)
		}
	}

	writeApi = influxC.WriteAPI(InfluxOrg, InfluxBucket)
	go autoFlush()
}

// 写入数据
// stat 放行pass 禁止block 熔断break
func addInfluxData(req *http.Request, stat string) {
	if !preCheck() {
		return
	}
	p := influxdb2.NewPoint(
		SandwichMeasure,
		map[string]string{
			TagDomain:        req.Host,
			TagClient:        req.RemoteAddr,
			TagMethod:        req.Method,
			TagRefer:         req.Referer(),
			TagContendLength: strconv.FormatInt(req.ContentLength, 10),
			TagUrl:           req.RequestURI,
			TagStat:          stat,
		},
		map[string]interface{}{
			FieldDomain: req.Host,
		},
		time.Now())
	writeApi.WritePoint(p)
}

func autoFlush() {
	if !preCheck() {
		return
	}
	ticker := time.Tick(1 * time.Second)
	for range ticker {
		writeApi.Flush()
	}
}

func preCheck() bool {
	return *EnableInflux
}

// 查询数据 无需聚合运算
// `from(bucket: "sandwich")|>range(start: -1h)|>filter(fn: (r)=>r._measurement == "sandwich")`
func getInfluxData(query string) []map[string]interface{} {
	var res []map[string]interface{}
	queryApi := influxC.QueryAPI(InfluxOrg)
	result, err := queryApi.Query(context.Background(), query)
	if err != nil {
		log.Printf("query influx error: %s\n", err.Error())
		return nil
	}
	for result.Next() {
		// log.Printf("%+v", result.Record().Values())
		res = append(res, result.Record().Values())
	}
	return res
}
