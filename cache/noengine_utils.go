/*
   Create: 2024/1/22
   Project: Sandwich
   Github: https://github.com/landers1037
   Copyright Renj
*/

package cache

import "os"

func getContent(file string) ([]byte, error) {
	return os.ReadFile(file)
}
