/*
Create: 2022/9/7
Project: Sandwich
Github: https://github.com/landers1037
Copyright Renj
*/

// Package data
package data

import (
	"context"
	"net/http"
	"sandwich/config"
	"sandwich/log"
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

func InitInflux() {
	cf := config.Get()
	if !preCheck() {
		return
	}
	log.Info("init influxdb")
	if cf.Database.Influx.Token != "" {
		influxC = influxdb2.NewClient(cf.Database.Influx.URL, cf.Database.Influx.Token)
	} else {
		influxC = influxdb2.NewClient(cf.Database.Influx.URL, cf.Database.Influx.Token)
		if cf.Database.Influx.Password == "" {
			cf.Database.Influx.Password = DefaultPwd
		}
		res, err := influxC.Setup(context.Background(), cf.Database.Influx.Org, cf.Database.Influx.Password, cf.Database.Influx.Org, cf.Database.Influx.Bucket, 0)
		if err != nil {
			log.GetLogger().Error().Err(err).Msg("setup Error")
		} else {
			log.GetLogger().Info().Str("token", *res.Auth.Token).Msg("setup finished")
		}
	}

	writeApi = influxC.WriteAPI(cf.Database.Influx.Org, cf.Database.Influx.Bucket)
	go autoFlush()
}

// AddInfluxData 写入数据
// stat 放行pass 禁止block 熔断break
func AddInfluxData(req *http.Request, stat string) {
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
	cf := config.Get()
	return cf.Database.Influx.Enabled
}

// 查询数据 无需聚合运算
// `from(bucket: "sandwich")|>range(start: -1h)|>filter(fn: (r)=>r._measurement == "sandwich")`
func getInfluxData(query string) []map[string]interface{} {
	cf := config.Get()
	var res []map[string]interface{}
	queryApi := influxC.QueryAPI(cf.Database.Influx.Org)
	result, err := queryApi.Query(context.Background(), query)
	if err != nil {
		log.GetLogger().Error().Err(err).Msg("query influx error")
		return nil
	}
	for result.Next() {
		res = append(res, result.Record().Values())
	}
	return res
}
