//go:build v1

/*
Project: Sandwich sandwich.go
Created: 2021/12/12 by Landers
*/

package main

import (
	"fmt"
	"net/http"
	"sandwich/breaker"
	"sandwich/cache"
	"sandwich/constant"
	"sandwich/data"
	"sandwich/flow"
	"sandwich/jobs"
	"sandwich/log"
)

func v1() {
	constant.InitConfigFromEnvs()
	log.InitLog()
	data.InitMongo()
	data.InitInflux()
	data.InitPool()
	// load noengine map
	cache.InitNoEngineDomainMap()
	// load helios config
	InitHeliosConfig()

	// init gzip cache for static pages
	cache.initGzipCache()

	// start sync jobs
	jobs.InitSyncJobs()

	// init worker
	breaker.InitBreaker()
	flow.InitLimiter()
	log.InfoF("proxy server start on: %s:%s", constant.Host, constant.Port)
	err := http.ListenAndServe(fmt.Sprintf("%s:%s", constant.Host, constant.Port), Proxy())
	if err != nil {
		log.ErrorF("proxy server err: %s\n", err.Error())
	}
}
