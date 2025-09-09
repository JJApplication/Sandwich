/*
Project: Sandwich sandwich.go
Created: 2021/12/12 by Landers
*/

package main

import (
	"fmt"
	"net/http"
	"sandwich/constant"
	"sandwich/log"
)

func main() {
	constant.InitConfigFromEnvs()
	log.InitLog()
	InitMongo()
	InitInflux()
	InitPool()
	// load noengine map
	InitNoEngineDomainMap()
	// load helios config
	InitHeliosConfig()

	// init gzip cache for static pages
	initGzipCache()

	// start sync jobs
	InitSyncJobs()

	// init worker
	InitBreaker()
	InitLimiter()
	log.InfoF("proxy server start on: %s:%s", constant.Host, constant.Port)
	err := http.ListenAndServe(fmt.Sprintf("%s:%s", constant.Host, constant.Port), Proxy())
	if err != nil {
		log.ErrorF("proxy server err: %s\n", err.Error())
	}
}
