package utils

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"errors"
	"log/slog"
	"math/rand"
	"strings"
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

func AesEncrypt(orig string, key string) (string, error) {
	orig = strings.TrimSpace(orig)
	defer func() {
		if err := recover(); err != nil {
			slog.Error("AES加密错误", "err_msg", err)
		}
	}()
	if len(orig) == 0 {
		return "", nil
	}
	if len(key) != 32 {
		return "", errors.New("加密需要的key长度错误")
	}
	// 转成字节数组
	origData := []byte(orig)
	k := []byte(key)
	// 分组秘钥
	block, err := aes.NewCipher(k)
	if err != nil {
		return "", err
	}
	// 获取秘钥块的长度
	blockSize := block.BlockSize()
	//补码
	PKCS7Padding := func(ciphertext []byte, blockSize int) []byte {
		padding := blockSize - len(ciphertext)%blockSize
		padText := bytes.Repeat([]byte{byte(padding)}, padding)
		return append(ciphertext, padText...)
	}
	origData = PKCS7Padding(origData, blockSize)
	// 加密模式
	blockMode := cipher.NewCBCEncrypter(block, k[:blockSize])
	// 创建数组
	crypt := make([]byte, len(origData))
	// 加密
	blockMode.CryptBlocks(crypt, origData)
	return base64.StdEncoding.EncodeToString(crypt), nil
}

func AesDecrypt(crypt string, key string) (string, error) {
	crypt = strings.TrimSpace(crypt)
	defer func() {
		if err := recover(); err != nil {
			slog.Error("AES解密错误", "err_msg", err)
		}
	}()
	if len(crypt) == 0 {
		return "", nil
	}
	if len(crypt) < 1 || len(key) != 32 {
		return "", errors.New("解密需要的key长度错误")
	}
	// 转成字节数组
	cryptByte, _ := base64.StdEncoding.DecodeString(crypt)
	k := []byte(key)
	// 分组秘钥
	block, err := aes.NewCipher(k)
	if err != nil {
		return "", err
	}
	// 获取秘钥块的长度
	blockSize := block.BlockSize()
	// 加密模式
	blockMode := cipher.NewCBCDecrypter(block, k[:blockSize])
	// 创建数组
	orig := make([]byte, len(cryptByte))
	// 解密
	blockMode.CryptBlocks(orig, cryptByte)
	// 去补全码
	PKCS7UnPadding := func(origData []byte) []byte {
		length := len(origData)
		unPadding := int(origData[length-1])
		return origData[:(length - unPadding)]
	}
	orig = PKCS7UnPadding(orig)
	return string(orig), nil
}
