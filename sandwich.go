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

const (
	Sandwich = "Sandwich"
)

func main() {
	parseFlags()
	initLog()
	initMongo()
	initInflux()
	initPool()
	// load noengine map
	InitNoEngineDomainMap()
	// load helios config
	InitHeliosConfig()

	// start sync jobs
	go syncDomainMap()
	log.Println("proxy server start")
	err := http.ListenAndServe(fmt.Sprintf(":%s", Port), Proxy())
	if err != nil {
		log.Printf("proxy server err: %s\n", err.Error())
	}
}
