//go:build windows

// What Is My PC — 整机信息 / 监控 / 跑分工具
// 作者：ALing Studios　构建：见 bridge.BuildDate
package main

import (
	"embed"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"syscall"
	"unsafe"

	webview "github.com/jchv/go-webview2"

	"whatismypc/internal/bridge"
)

//go:embed all:web
var webFS embed.FS

func main() {
	sub, err := fs.Sub(webFS, "web")
	if err != nil {
		fatal("内嵌资源缺失：" + err.Error())
		return
	}

	// 内嵌页面通过本机回环服务提供（WebView2 对 http 源支持最完整）
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fatal("本地服务初始化失败：" + err.Error())
		return
	}
	srv := &http.Server{Handler: http.FileServer(http.FS(sub))}
	go func() { _ = srv.Serve(ln) }()

	w := webview.NewWithOptions(webview.WebViewOptions{
		AutoFocus: true,
		WindowOptions: webview.WindowOptions{
			Title:  bridge.AppName,
			Width:  1220,
			Height: 820,
			IconId: 1, // 与 rsrc 嵌入的图标资源 ID 对应
			Center: true,
		},
	})
	if w == nil {
		fatal("未检测到 WebView2 运行时组件。\n请安装 Microsoft Edge WebView2 Runtime 后重新运行本软件。\n下载地址：https://developer.microsoft.com/zh-cn/microsoft-edge/webview2/")
		return
	}
	defer w.Destroy()

	w.SetSize(1024, 700, webview.HintMin)
	bridge.Register(w)
	w.Navigate(fmt.Sprintf("http://%s/", ln.Addr().String()))
	w.Run()
}

// fatal 弹出系统消息框后退出。
func fatal(msg string) {
	user32 := syscall.NewLazyDLL("user32.dll")
	mb := user32.NewProc("MessageBoxW")
	text, _ := syscall.UTF16PtrFromString(msg)
	caption, _ := syscall.UTF16PtrFromString("What Is My PC")
	_, _, _ = mb.Call(0, uintptr(unsafe.Pointer(text)), uintptr(unsafe.Pointer(caption)), 0x10)
}
