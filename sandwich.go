/*
Project: Sandwich sandwich.go
Created: 2021/12/12 by Landers
*/

package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	InitConfigFromEnvs()
	InitLog()
	InitMongo()
	InitInflux()
	InitPool()
	// load noengine map
	InitNoEngineDomainMap()
	// load helios config
	InitHeliosConfig()

	// start sync jobs
	InitSyncJobs()
	log.Println("proxy server start")
	err := http.ListenAndServe(fmt.Sprintf("%s:%s", Host, Port), Proxy())
	if err != nil {
		log.Printf("proxy server err: %s\n", err.Error())
	}
}
