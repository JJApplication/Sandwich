/*
Project: Sandwich log.go
Created: 2021/12/12 by Landers
*/

package main

import (
	"log"
)

const (
	PREFIX = "[SANDWICH]"
)

func init() {
	log.SetPrefix(PREFIX)
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}
