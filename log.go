/*
Project: Sandwich log.go
Created: 2021/12/12 by Landers
*/

package main

import (
	"fmt"
	"log"
)

var (
	PREFIX = fmt.Sprintf("[%s] ", Sandwich)
	DEBUG  = "[DEBUG] "
)

func InitLog() {
	log.SetPrefix(PREFIX)
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

func debugF(fmt string, v ...interface{}) {
	if Debug {
		log.Printf(DEBUG+fmt, v...)
	}
}

func debug(v ...interface{}) {
	if Debug {
		vv := append([]interface{}{"[DEBUG] "}, v...)
		log.Println(vv...)
	}
}
