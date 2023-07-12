package util

import (
	"fmt"
	"strings"
)

func GetHash(magnet string) string {
	splitList := strings.Split(magnet, ":")
	fmt.Println(splitList, "分割出来的路径", "hash: ", splitList[len(splitList)-1])
	return splitList[len(splitList)-1]
}
