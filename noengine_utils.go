/*
   Create: 2024/1/22
   Project: Sandwich
   Github: https://github.com/landers1037
   Copyright Renj
*/

package main

import "os"

const (
	refreshTime = 1 * 60 * 60
)

func getContent(file string) ([]byte, error) {
	return os.ReadFile(file)
}
