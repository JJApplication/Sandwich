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
)

func InitLog() {
	log.SetPrefix(PREFIX)
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}
