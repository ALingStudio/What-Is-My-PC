package webinfo

// AI token 加密存储：二进制内只保留 AES-256-GCM 密文，
// 运行时用派生密钥解密。可防止直接字符串扫描泄露。

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/base64"
	"sync"
)

// aiTokenCipher 为 token 的 AES-GCM 密文（nonce + 密文，base64）。
const aiTokenCipher = "9DMzBaVaxZJKm8zwSbEl3z79Ba1hiVQLg2+kbH3NPtquXCwA0IAYJZKEc0kCsU9sMaBZyGXSNVbLO4g4Fy/LtStzAr52+juki+2tHDB2kQ=="

// 派生密钥的种子（与加密时一致）。
var aiKeySeed = []byte("WhatIsMyPC#" + "ALingStudios#" + "2026#" + "agnes-key")

var (
	tokenOnce sync.Once
	tokenVal  string
)

// decryptAIToken 解密内置 token；失败返回空串（可被配置文件覆盖兜底）。
func decryptAIToken() string {
	tokenOnce.Do(func() {
		key := sha256.Sum256(aiKeySeed)
		raw, err := base64.StdEncoding.DecodeString(aiTokenCipher)
		if err != nil {
			return
		}
		block, err := aes.NewCipher(key[:])
		if err != nil {
			return
		}
		gcm, err := cipher.NewGCM(block)
		if err != nil {
			return
		}
		ns := gcm.NonceSize()
		if len(raw) <= ns {
			return
		}
		pt, err := gcm.Open(nil, raw[:ns], raw[ns:], nil)
		if err != nil {
			return
		}
		tokenVal = string(pt)
	})
	return tokenVal
}
