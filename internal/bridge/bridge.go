// Package bridge 将后端能力绑定到前端（WebView2 JS Bridge）。
package bridge

import (
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	webview "github.com/jchv/go-webview2"

	"whatismypc/internal/benchmark"
	"whatismypc/internal/monitor"
	"whatismypc/internal/sysinfo"
	"whatismypc/internal/webinfo"
)

const (
	AppName    = "What Is My PC"
	AppVersion = "V0.1b"
	BuildDate  = "2026-09-05"
	Author     = "ALing Studios"
	AppNote    = "本软件由AI辅助完成"
)

// workDir 数据目录（历史记录、跑分临时文件）。需要管理员权限，使用 ProgramData。
func workDir() string {
	pd := os.Getenv("ProgramData")
	if pd == "" {
		pd = os.TempDir()
	}
	d := filepath.Join(pd, "WhatIsMyPC")
	_ = os.MkdirAll(d, 0755)
	return d
}

// consentPath 用户协议同意标记文件。
func consentPath() string {
	return filepath.Join(workDir(), "consent.dat")
}

type gpuResult struct {
	fps float64
	ok  bool
}

// Bridge 持有运行状态。
type Bridge struct {
	w webview.WebView

	mu           sync.Mutex
	benchRunning bool
	gpuCh        chan gpuResult

	slowMu    sync.Mutex
	slowData  map[string]interface{}
	slowAt    time.Time
	slowFails int
}

// slowSnapshot 返回磁盘清单与 S.M.A.R.T. 缓存（20 秒过期；失败后 60 秒再重试）。
func (b *Bridge) slowSnapshot() map[string]interface{} {
	b.slowMu.Lock()
	defer b.slowMu.Unlock()

	ttl := 20 * time.Second
	if b.slowFails > 0 {
		ttl = 60 * time.Second
	}
	if b.slowData != nil && time.Since(b.slowAt) < ttl {
		return b.slowData
	}

	d, err := monitor.SnapshotSlow()
	if err != nil {
		b.slowFails++
		if b.slowData == nil {
			b.slowData = map[string]interface{}{
				"physicalDisks": []interface{}{},
				"smartPredict":  []interface{}{},
			}
			b.slowAt = time.Now()
		}
		return b.slowData
	}
	b.slowFails = 0
	b.slowData = d
	b.slowAt = time.Now()
	return d
}

// Register 注册全部 JS 绑定。
func Register(w webview.WebView) {
	b := &Bridge{w: w}

	_ = w.Bind("app_getMeta", func() map[string]interface{} {
		return map[string]interface{}{
			"name": AppName, "version": AppVersion, "buildDate": BuildDate,
			"author": Author, "note": AppNote,
			"items": benchmark.Items,
			"gradeThresholds": func() []map[string]interface{} {
				var out []map[string]interface{}
				for _, g := range benchmark.GradeThresholds {
					out = append(out, map[string]interface{}{"grade": g.Grade, "min": g.Min})
				}
				return out
			}(),
		}
	})

	_ = w.Bind("app_getSystemInfo", func() (map[string]interface{}, error) {
		return sysinfo.Collect()
	})

	_ = w.Bind("app_getOfficialInfo", func(query string) (*webinfo.Result, error) {
		return webinfo.Lookup(query)
	})

	_ = w.Bind("app_getPerf", func() (map[string]interface{}, error) {
		fast, err := monitor.SnapshotFast()
		if err != nil {
			return nil, err
		}
		for k, v := range b.slowSnapshot() {
			fast[k] = v
		}
		return fast, nil
	})

	_ = w.Bind("app_getHistory", func() *benchmark.History {
		return benchmark.LoadHistory(workDir())
	})

	_ = w.Bind("app_startBenchmark", func() (map[string]interface{}, error) {
		b.mu.Lock()
		if b.benchRunning {
			b.mu.Unlock()
			return map[string]interface{}{"ok": false, "error": "测试正在进行中"}, nil
		}
		b.benchRunning = true
		b.gpuCh = make(chan gpuResult, 1)
		b.mu.Unlock()

		go b.runBenchmark()
		return map[string]interface{}{"ok": true}, nil
	})

	_ = w.Bind("app_gpuResult", func(fps float64) map[string]interface{} {
		b.mu.Lock()
		ch := b.gpuCh
		b.mu.Unlock()
		if ch != nil {
			select {
			case ch <- gpuResult{fps: fps, ok: fps > 0}:
			default:
			}
		}
		return map[string]interface{}{"ok": true}
	})

	_ = w.Bind("app_getConsent", func() map[string]interface{} {
		_, err := os.Stat(consentPath())
		return map[string]interface{}{"accepted": err == nil}
	})

	_ = w.Bind("app_acceptAgreement", func() map[string]interface{} {
		if err := os.WriteFile(consentPath(), []byte("accepted "+time.Now().Format("2006-01-02 15:04:05")), 0644); err != nil {
			return map[string]interface{}{"ok": false, "error": err.Error()}
		}
		return map[string]interface{}{"ok": true}
	})

	_ = w.Bind("app_exit", func() map[string]interface{} {
		go func() {
			time.Sleep(100 * time.Millisecond)
			w.Terminate()
		}()
		return map[string]interface{}{"ok": true}
	})

	_ = w.Bind("app_openURL", func(url string) map[string]interface{} {
		if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
			_ = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
		}
		return map[string]interface{}{"ok": true}
	})

	_ = w.Bind("app_saveShare", func(dataURL string) (map[string]interface{}, error) {
		p, err := b.saveShareImage(dataURL)
		if err != nil {
			return nil, err
		}
		// 打开资源管理器并选中该图片
		go func() {
			_ = exec.Command("explorer.exe", "/select,"+p).Start()
		}()
		return map[string]interface{}{"ok": true, "path": p}, nil
	})
}

// emit 在主线程向前端推送跑分事件。
func (b *Bridge) emit(jsonStr string) {
	b.w.Dispatch(func() {
		b.w.Eval("window.__benchEvent && window.__benchEvent(" + jsonStr + ");")
	})
}

func (b *Bridge) runBenchmark() {
	defer func() {
		b.mu.Lock()
		b.benchRunning = false
		b.mu.Unlock()
	}()

	b.emit(`{"phase":"start"}`)

	gpu := func() (float64, bool) {
		b.mu.Lock()
		ch := b.gpuCh
		b.mu.Unlock()
		if ch == nil {
			return 0, false
		}
		select {
		case r := <-ch:
			return r.fps, r.ok
		case <-time.After(25 * time.Second):
			return 0, false
		}
	}

	res := benchmark.Run(b.emit, gpu, workDir())
	h := benchmark.SaveRun(workDir(), res)

	b.emit(fmt.Sprintf(
		`{"phase":"done","total":%d,"grade":"%s","bestTotal":%d,"bestGrade":"%s"}`,
		res.Total, res.Grade, h.BestTotal, h.BestGrade,
	))
}

// saveShareImage 保存 canvas 分享图，返回文件路径。
func (b *Bridge) saveShareImage(dataURL string) (string, error) {
	const prefix = "data:image/png;base64,"
	if !strings.HasPrefix(dataURL, prefix) {
		return "", fmt.Errorf("无效的图像数据")
	}
	raw, err := base64.StdEncoding.DecodeString(dataURL[len(prefix):])
	if err != nil {
		return "", fmt.Errorf("解码失败: %v", err)
	}

	stamp := time.Now().Format("20060102_150405")
	name := fmt.Sprintf("WhatIsMyPC_分享图_%s.png", stamp)

	var lastErr error
	var candidates []string
	if pics, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates, filepath.Join(pics, "Pictures"))
	}
	candidates = append(candidates, workDir())

	for _, dir := range candidates {
		if dir == "" {
			continue
		}
		if err := os.MkdirAll(dir, 0755); err != nil {
			lastErr = err
			continue
		}
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, raw, 0644); err != nil {
			lastErr = err
			continue
		}
		return p, nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("无可用保存目录")
	}
	return "", lastErr
}
