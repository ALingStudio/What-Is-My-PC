// Package benchmark 跑分引擎（页面3）。
//
// 五个项目，每项 0~2000 分，总分 10000。分数 = 2000 × 基准值/实测值（或实测/基准），
// 基准常量集中定义于下方，首版按主流机型估算，可在实测后微调。
package benchmark

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// Item 一个测试项目。
type Item struct {
	Key  string `json:"key"`
	Name string `json:"name"`
	Desc string `json:"desc"`
}

// Items 测试顺序即展示顺序。
var Items = []Item{
	{Key: "cpu_single", Name: "CPU 单核", Desc: "素数筛 · 单核整数运算"},
	{Key: "cpu_multi", Name: "CPU 多核", Desc: "素数筛 · 全核并行运算"},
	{Key: "memory", Name: "内存", Desc: "内存拷贝带宽"},
	{Key: "disk", Name: "磁盘", Desc: "系统盘顺序读写"},
	{Key: "gpu", Name: "GPU", Desc: "WebGL 着色器渲染"},
}

// ---- 评分基准（可随实测校准）----
const (
	sieveLimit    = 300_000_000 // 素数筛规模
	refSingleSec  = 2.2         // 单核跑完一轮筛的参考耗时（秒）→ 1000 分
	refMultiCount = 12.0        // 3 秒内多核完成轮数参考 → 1000 分
	refMemGBps    = 20.0        // 内存带宽参考（GB/s）→ 1000 分
	refDiskMBps   = 900.0       // 磁盘混合读写参考（MB/s）→ 1000 分
	refGPUFPS     = 150.0       // GPU 渲染参考帧率 → 1000 分
)

// GradeThresholds 评级下限（从高到低）。
// 校准基准：RTX 4060 显卡的主流主机（如 i5-13400F/DDR4-3200/NVMe）
// 总分约落在 8700~9100 区间，对应 S 级。实测后可微调。
var GradeThresholds = []struct {
	Grade string
	Min   int
}{
	{"SSS", 9700}, {"SS", 9300}, {"S", 8700}, {"A", 7800},
	{"B", 6500}, {"C", 4500}, {"D", 0},
}

// Grade 总分 → 评级。
func Grade(total int) string {
	for _, g := range GradeThresholds {
		if total >= g.Min {
			return g.Grade
		}
	}
	return "D"
}

func clampScore(v float64) int {
	if v < 10 {
		return 10
	}
	if v > 2000 {
		return 2000
	}
	return int(math.Round(v))
}

// ---- 素数筛（位图只存奇数）----

func sieve(n int) int {
	size := (n >> 1) + 1 // 下标 i 表示 2i+1
	bits := make([]uint64, (size>>6)+1)
	sqrt := int(math.Sqrt(float64(n)))
	for i := 1; 2*i+1 <= sqrt; i++ {
		if bits[i>>6]&(1<<(uint(i)&63)) == 0 {
			p := 2*i + 1
			for j := (p * p) >> 1; j < size; j += p {
				bits[j>>6] |= 1 << (uint(j) & 63)
			}
		}
	}
	count := 1 // 2
	for i := 1; i < size; i++ {
		if bits[i>>6]&(1<<(uint(i)&63)) == 0 {
			count++
		}
	}
	return count
}

func benchCPUSingle() int {
	start := time.Now()
	sieve(sieveLimit)
	elapsed := time.Since(start).Seconds()
	return clampScore(2000 * refSingleSec / elapsed)
}

func benchCPUMulti() int {
	d := 3 * time.Second
	n := runtime.NumCPU()
	if n < 1 {
		n = 1
	}
	var done int64
	stop := time.Now().Add(d)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for time.Now().Before(stop) {
				sieve(sieveLimit)
				atomic.AddInt64(&done, 1)
			}
		}()
	}
	wg.Wait()
	return clampScore(2000 * float64(done) / refMultiCount)
}

func benchMemory() int {
	const sz = 256 << 20
	src := make([]byte, sz)
	dst := make([]byte, sz)
	for i := 0; i < sz; i += 4096 {
		src[i] = byte(i)
	}
	best := 0.0
	for r := 0; r < 3; r++ {
		start := time.Now()
		copy(dst, src)
		el := time.Since(start).Seconds()
		if el <= 0 {
			continue
		}
		g := float64(sz) * 2 / el / (1 << 30) // 读 + 写
		if g > best {
			best = g
		}
	}
	return clampScore(2000 * best / refMemGBps)
}

func benchDisk(dir string) (int, error) {
	path := filepath.Join(dir, "bench.tmp")
	const total = 512 << 20
	const chunk = 4 << 20
	buf := make([]byte, chunk)

	// 顺序写
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return 0, err
	}
	start := time.Now()
	for written := 0; written < total; written += chunk {
		if _, err := f.Write(buf); err != nil {
			f.Close()
			return 0, err
		}
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return 0, err
	}
	f.Close()
	wMBps := float64(total) / time.Since(start).Seconds() / (1 << 20)

	// 顺序读
	rf, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	start = time.Now()
	for {
		n, rerr := rf.Read(buf)
		if n == 0 || rerr != nil {
			break
		}
	}
	rf.Close()
	rMBps := float64(total) / time.Since(start).Seconds() / (1 << 20)

	_ = os.Remove(path)
	return clampScore(2000 * (wMBps+rMBps) / 2 / refDiskMBps), nil
}

// RunResult 一次完整跑分结果。
type RunResult struct {
	Scores map[string]int
	Total  int
	Grade  string
}

// GPUSource 由 bridge 注入：向 JS 请求 WebGL 测试结果。
// 返回帧率与是否成功；失败/超时时由调用方决定降级策略。
type GPUSource func() (fps float64, ok bool)

// Run 顺序执行全部项目。emit 用于推送进度事件（JSON 字符串）。
// workDir 用于磁盘测试与历史记录存储。
func Run(emit func(string), gpu GPUSource, workDir string) *RunResult {
	scores := make(map[string]int)
	push := func(index, score int) {
		scores[Items[index].Key] = score
		if emit != nil {
			emit(fmt.Sprintf(`{"phase":"item","index":%d,"key":"%s","score":%d}`, index, Items[index].Key, score))
		}
	}

	// 1. CPU 单核
	push(0, benchCPUSingle())
	// 2. CPU 多核
	push(1, benchCPUMulti())
	// 3. 内存
	push(2, benchMemory())
	// 4. 磁盘
	if s, err := benchDisk(workDir); err == nil {
		push(3, s)
	} else {
		push(3, 10)
	}
	// 5. GPU（由前端 WebGL 完成）
	gpuScore := 100 // 无法完成 WebGL 测试时的保底分
	if gpu != nil {
		if emit != nil {
			emit(`{"phase":"gpu"}`)
		}
		if fps, ok := gpu(); ok && fps > 0 {
			gpuScore = clampScore(2000 * fps / refGPUFPS)
		}
	}
	push(4, gpuScore)

	total := 0
	for _, s := range scores {
		total += s
	}
	return &RunResult{Scores: scores, Total: total, Grade: Grade(total)}
}

// ---- 历史记录 ----

// RunRecord 一次跑分记录。
type RunRecord struct {
	Date   string         `json:"date"`
	Total  int            `json:"total"`
	Grade  string         `json:"grade"`
	Scores map[string]int `json:"scores"`
}

// History 持久化结构。
type History struct {
	LastRun   *RunRecord `json:"lastRun,omitempty"`
	BestTotal int        `json:"bestTotal"`
	BestGrade string     `json:"bestGrade"`
	BestDate  string     `json:"bestDate,omitempty"`
}

func historyPath(dir string) string { return filepath.Join(dir, "history.json") }

// LoadHistory 读取历史；不存在时返回空结构。
func LoadHistory(dir string) *History {
	h := &History{}
	b, err := os.ReadFile(historyPath(dir))
	if err != nil {
		return h
	}
	_ = json.Unmarshal(b, h)
	return h
}

// SaveRun 记录一次跑分并持久化。
func SaveRun(dir string, r *RunResult) *History {
	h := LoadHistory(dir)
	rec := &RunRecord{
		Date:   time.Now().Format("2006-01-02 15:04"),
		Total:  r.Total,
		Grade:  r.Grade,
		Scores: r.Scores,
	}
	h.LastRun = rec
	if r.Total > h.BestTotal {
		h.BestTotal = r.Total
		h.BestGrade = r.Grade
		h.BestDate = rec.Date
	}
	b, _ := json.MarshalIndent(h, "", "  ")
	_ = os.WriteFile(historyPath(dir), b, 0644)
	return h
}
