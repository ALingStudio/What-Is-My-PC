# What Is My PC

整机信息 / 实时占用 / 跑分测试工具 · Windows x64 单文件版

- 作者：ALing Studios
- 版本：V0.1b（验收通过后升级 V1.0）
- 构建日期：2026-09-05
- 备注：本软件由AI辅助完成

## 功能

| 页面 | 内容 |
| --- | --- |
| 电脑配置 | 顶部居中显示电脑名称；靠右展示 CPU、主板、内存、显卡、磁盘、网络、显示器、系统等配件；点击部件名称可联网获取官网描述（失败显示"资源获取失败"） |
| 实时占用 | 任务管理器风格：CPU / 内存 / GPU / 各磁盘的占用曲线、温度、S.M.A.R.T. 健康状态、卷占用 |
| 跑分测试 | 五项测试（CPU 单核 / 多核、内存、磁盘、GPU），总分 10000，评级 D~SSS，支持生成分享图 |
| 关于软件 | 作者、版本、构建日期、备注 |

## 使用方法

1. 双击运行 `WhatIsMyPC.exe`
2. **首次启动会弹出《用户协议与免责声明》**，阅读并点击"我已阅读并同意"后才能进入
3. 首次运行会弹出 UAC 管理员授权窗口（读取温度与 S.M.A.R.T. 需要），点"是"
4. 依赖：Windows 10/11 + Microsoft Edge WebView2 运行时（Win11 自带，Win10 一般已内置）
5. 历史跑分记录保存在 `C:\ProgramData\WhatIsMyPC\history.json`
6. 分享图默认保存到当前用户的"图片"文件夹

## 协议与隐私

- 首次启动强制展示免责声明，未同意不可使用；同意状态记录于本机。
- 反拆包条款：**严禁恶意拆包、反编译、破解或二次打包分发**，违者开发者（ALing Studios）保留依法追究（含起诉）的权利。
- "官网信息"功能使用 **Agnes 2.5 flash** AI 模型，硬件型号文本将交由 Agnes 处理，详见 [Agnes 隐私条款](https://www.agnes-ai.com/zh-Hans/docs/privacy-policy)。
- 硬件信息仅读取自本机，跑分历史不上传任何服务器。

## ⚠️ 二次修改须知（重要）

本项目仓库中的 **`ai.json`（AI 配置文件）里 `baseUrl` / `apiKey` / `model` 均为空**——内置接口凭据不会随源码分发。

**二次修改 / 二次构建前，必须先在项目根目录的 `ai.json` 中填写你自己的 AI 配置：**

```json
{
  "baseUrl": "你的 AI 接口地址（如 https://xxx/v1）",
  "apiKey":  "你的 token",
  "model":   "模型名称"
}
```

三项全部留空时，程序会回退到内置默认接口与凭据；若内置凭据为空或失效，"官网信息"功能将不可用（显示"资源获取失败"）。

## 构建（开发者）

```bash
# Linux 交叉编译环境（Go 1.22+ / Python3 + Pillow）
bash build/build.sh      # 产物：dist/WhatIsMyPC.exe
bash build/sign.sh       # 代码签名（当前未启用，按脚本注释配置证书）
```

## 目录结构

```
├── main.go                  # 入口：窗口 + WebView2
├── ai.json                  # AI 接口配置（仓库模板，三项留空）
├── internal/
│   ├── sysinfo/             # 硬件配置采集（CIM/WMI）
│   ├── webinfo/             # 官网信息检索（AI 模型 + 搜索兜底）
│   ├── monitor/             # 占用率 / 温度 / S.M.A.R.T. 采样
│   ├── benchmark/           # 跑分算法与评级
│   ├── psexec/              # 静默 PowerShell 进程封装
│   └── bridge/              # Go ↔ JS 桥接（含用户协议同意状态）
├── web/                     # 前端页面（编译时嵌入 EXE）
├── assets/                  # 图标源文件与生成脚本
├── build/                   # 清单 / 打包脚本 / 签名脚本
├── third_party/sys          # golang.org/x/sys 本地镜像（离线构建用）
└── dist/WhatIsMyPC.exe      # 交付产物
```

---

内容由 AI 生成
