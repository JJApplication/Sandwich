/*
   Create: 2024/1/22
   Project: Sandwich
   Github: https://github.com/landers1037
   Copyright Renj
*/

package main

import "os"

const (
	// 域名列表刷新时长
	refreshTime = 1 * 60 * 60
	// 域名端口映射刷新时间
	refreshTime2 = 10 * 60
)

func getContent(file string) ([]byte, error) {
	return os.ReadFile(file)
}
