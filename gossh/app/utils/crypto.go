package utils

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
)

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

func EncryptString(plaintext, key string) (string, error) {
	plaintext = strings.TrimSpace(plaintext)
	defer func() {
		if err := recover(); err != nil {
			slog.Error("AES加密错误", "err_msg", err)
		}
	}()
	if len(plaintext) == 0 {
		return "", nil
	}
	if len(key) != 32 {
		return "", errors.New("加密需要的key长度错误")
	}

	// 创建 cipher
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return "", err
	}

	// 创建 GCM 模式
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// 创建 nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	// 加密
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)

	// 返回 base64 编码的结果
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func DecryptString(encrypted, key string) (string, error) {
	encrypted = strings.TrimSpace(encrypted)
	defer func() {
		if err := recover(); err != nil {
			slog.Error("AES解密错误", "err_msg", err)
		}
	}()
	if len(encrypted) == 0 {
		return "", nil
	}
	if len(encrypted) < 1 || len(key) != 32 {
		return "", errors.New("解密需要的key长度错误")
	}
	// 解码 base64
	ciphertext, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return "", err
	}

	// 创建 cipher
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return "", err
	}

	// 创建 GCM 模式
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// 提取 nonce
	if len(ciphertext) < gcm.NonceSize() {
		return "", fmt.Errorf("ciphertext too short")
	}
	nonce, ciphertext := ciphertext[:gcm.NonceSize()], ciphertext[gcm.NonceSize():]

	// 解密
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}
