/*
Project: Sandwich log.go
Created: 2021/12/12 by Landers
*/

package main

import (
	"log"
)

const (
	PREFIX = "[Sandwich] "
)

func initLog() {
	log.SetPrefix(PREFIX)
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}
