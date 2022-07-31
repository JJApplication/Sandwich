/*
Project: Sandwich sandwich.go
Created: 2021/12/12 by Landers
*/

package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
)

func main() {
	port := flag.String("port", "8888", "port")
	flag.Parse()
	initLog()

	p := Proxy()
	log.Println("proxy server start")
	err := http.ListenAndServe(fmt.Sprintf(":%s", *port), p)
	if err != nil {
		log.Printf("proxy server err: %s\n", err.Error())
	}
}
