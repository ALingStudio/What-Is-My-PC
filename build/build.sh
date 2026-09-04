#!/usr/bin/env bash
# What Is My PC — 单文件打包脚本（Linux 交叉编译 → Windows x64 EXE）
set -e
cd "$(dirname "$0")/.."

echo "==> 1/4 生成图标"
python3 assets/make_icon.py

echo "==> 2/4 生成资源文件（图标 + 清单 + 版本信息 → syso）"
GVI="$(command -v goversioninfo || true)"
[ -z "$GVI" ] && [ -x "$(go env GOPATH)/bin/goversioninfo" ] && GVI="$(go env GOPATH)/bin/goversioninfo"
if [ -n "$GVI" ]; then
  "$GVI" -64 \
    -icon=assets/icon.ico -manifest=build/manifest.xml \
    -comment "整机信息 / 实时占用 / 跑分工具" \
    -company "ALing Studios" \
    -description "What Is My PC" \
    -file-version "0.1.0.0" \
    -internal-name "WhatIsMyPC" \
    -copyright "© 2026 ALing Studios" \
    -product-name "What Is My PC" \
    -product-version "0.1.0" \
    -o resource_windows_amd64.syso
else
  go run github.com/akavel/rsrc@latest -arch amd64 -ico assets/icon.ico -manifest build/manifest.xml -o resource_windows_amd64.syso
fi

echo "==> 3/4 编译（GOOS=windows GOARCH=amd64，GUI 子系统，去符号）"
mkdir -p dist
GOOS=windows GOARCH=amd64 go build -trimpath \
  -ldflags "-H windowsgui -s -w" \
  -o dist/WhatIsMyPC.exe .

echo "==> 4/4 完成"
ls -lh dist/WhatIsMyPC.exe
