#!/usr/bin/env bash
# 代码签名脚本（当前版本：暂不签名，预留流程）
#
# 用户提供签名信息后启用：
#   1. 将 PFX 证书放到 build/cert.pfx（或通过环境变量 CERT_PATH 指定）
#   2. 设置环境变量 CERT_PASSWORD
#   3. 运行本脚本
#
# 使用 osslsigncode 对 PE 文件签名：
#   osslsigncode sign -pkcs12 "$CERT_PATH" -pass "$CERT_PASSWORD" \
#     -n "What Is My PC" -i "https://github.com/ALingStudio" \
#     -t http://timestamp.digicert.com \
#     -in dist/WhatIsMyPC.exe -out dist/WhatIsMyPC-signed.exe
#
# 注意：自签名证书无法消除 SmartScreen 警告，需正式代码签名证书。

echo "当前版本未启用签名。如需签名，请按脚本内注释配置证书后运行。"
