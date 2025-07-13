package utils

import (
	"math/rand"
	"time"
	"unicode/utf8"
)

// RandString 生成指定长度随机字符串
func RandString(length int) string {
	str := "0123456789abcdefghijklmnopqrstuvwxyz"
	data := []byte(str)
	var result []byte
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < length; i++ {
		result = append(result, data[r.Intn(len(data))])
	}
	return string(result)
}

func TruncateString(s string, length int) string {
	if utf8.RuneCountInString(s) <= length {
		return s
	}

	runes := []rune(s)
	if length < 1 {
		return ""
	}
	return string(runes[:length])
}
