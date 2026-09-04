// Package webinfo 联网检索部件官网描述（页面1点击部件名称时调用）。
// 策略：DuckDuckGo HTML 检索优先，Bing 兜底；优先挑选官方域名结果。
package webinfo

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// Result 检索结果。
type Result struct {
	Title   string `json:"title"`
	Snippet string `json:"snippet"`
	URL     string `json:"url"`
	Source  string `json:"source"`
}

var ErrNotFound = errors.New("资源获取失败")

const userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36"

var httpClient = &http.Client{Timeout: 18 * time.Second}

// officialDomains 各品牌对应的官方域名（用于优先挑选）。
var officialDomains = map[string][]string{
	"intel":    {"intel.com", "intel.cn", "intel.com.cn", "ark.intel.com"},
	"amd":      {"amd.com"},
	"nvidia":   {"nvidia.com", "nvidia.cn", "geforce.com"},
	"samsung":  {"samsung.com", "samsungsemiconductor.com", "samsung.com.cn"},
	"seagate":  {"seagate.com"},
	"wd":       {"wd.com", "westerndigital.com"},
	"kingston": {"kingston.com"},
	"crucial":  {"crucial.com", "micron.com"},
	"msi":      {"msi.com"},
	"asus":     {"asus.com", "asus.com.cn"},
	"gigabyte": {"gigabyte.com", "gigabyte.cn"},
	"asrock":   {"asrock.com"},
	"corsair":  {"corsair.com"},
	"hynix":    {"skhynix.com"},
	"adata":    {"adata.com"},
	"biostar":  {"biostar.com.tw"},
	"colorful": {"colorful.cn", "colorfulgrape.com"},
	"realtek":  {"realtek.com"},
	"lenovo":   {"lenovo.com", "lenovo.com.cn"},
	"dell":     {"dell.com", "dell.com.cn"},
	"hp":       {"hp.com", "hp.com.cn"},
	"huawei":   {"huawei.com"},
	"honor":    {"honor.com", "hihonor.com"},
	"xiaomi":   {"mi.com", "xiaomi.com"},
}

// hintDomains 根据查询内容猜测应优先的官方域名列表。
func hintDomains(query string) []string {
	q := strings.ToLower(query)
	keywords := map[string][]string{
		"intel":    {"intel", "core i", "core ultra", "celeron", "pentium", "xeon"},
		"amd":      {"amd", "ryzen", "radeon", "athlon", "epyc"},
		"nvidia":   {"nvidia", "geforce", "rtx", "gtx", "quadro"},
		"samsung":  {"samsung", "三星"},
		"seagate":  {"seagate", "希捷"},
		"wd":       {"western digital", "wd ", "black sn", "blue sn", "西数"},
		"kingston": {"kingston", "金士顿"},
		"crucial":  {"crucial", "英睿达"},
		"msi":      {"msi", "微星"},
		"asus":     {"asus", "华硕", "rog"},
		"gigabyte": {"gigabyte", "技嘉", "aorus"},
		"asrock":   {"asrock", "华擎"},
		"corsair":  {"corsair", "美商海盗船"},
		"hynix":    {"hynix", "海力士"},
		"adata":    {"adata", "威刚", "xpg"},
		"biostar":  {"biostar", "映泰"},
		"colorful": {"colorful", "七彩虹"},
		"realtek":  {"realtek", "瑞昱"},
		"lenovo":   {"lenovo", "联想", "thinkpad", "拯救者"},
		"dell":     {"dell", "戴尔", "alienware"},
		"hp":       {" hp ", "惠普", "暗影精灵"},
		"huawei":   {"huawei", "华为", "matebook"},
		"honor":    {"honor", "荣耀", "magicbook"},
		"xiaomi":   {"xiaomi", "小米", "redmibook"},
	}
	var out []string
	for brand, kws := range keywords {
		for _, k := range kws {
			if strings.Contains(q, k) {
				out = append(out, officialDomains[brand]...)
				break
			}
		}
	}
	return out
}

type searchItem struct {
	Title   string
	URL     string
	Snippet string
}

func fetch(urlStr string) (string, error) {
	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("http %d", resp.StatusCode)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return "", err
	}
	return string(b), nil
}

var (
	reDDGBlock  = regexp.MustCompile(`(?s)<div[^>]*class="[^"]*result results_links[^"]*".*?</div>\s*</div>`)
	reDDGLink   = regexp.MustCompile(`<a[^>]*class="[^"]*result__a[^"]*"[^>]*href="([^"]+)"[^>]*>(.*?)</a>`)
	reDDGSnip   = regexp.MustCompile(`(?s)<a[^>]*class="[^"]*result__snippet[^"]*"[^>]*>(.*?)</a>`)
	reBingBlock = regexp.MustCompile(`(?s)<li[^>]*class="[^"]*b_algo[^"]*".*?</li>`)
	reHref      = regexp.MustCompile(`href="([^"]+)"`)
	reH2A       = regexp.MustCompile(`(?s)<h2[^>]*>.*?<a[^>]*href="([^"]+)"[^>]*>(.*?)</a>`)
	rePSnip     = regexp.MustCompile(`(?s)<p[^>]*>(.*?)</p>`)
)

func searchDDG(query string) ([]searchItem, error) {
	html, err := fetch("https://html.duckduckgo.com/html/?q=" + url.QueryEscape(query))
	if err != nil {
		return nil, err
	}
	var items []searchItem
	for _, block := range reDDGBlock.FindAllString(html, 12) {
		lm := reDDGLink.FindStringSubmatch(block)
		if lm == nil {
			continue
		}
		href := cleanDDGHref(lm[1])
		title := stripTags(lm[2])
		snippet := ""
		if sm := reDDGSnip.FindStringSubmatch(block); sm != nil {
			snippet = stripTags(sm[1])
		}
		if href == "" || title == "" {
			continue
		}
		items = append(items, searchItem{Title: title, URL: href, Snippet: snippet})
	}
	if len(items) == 0 {
		return nil, errors.New("ddg: no results")
	}
	return items, nil
}

func cleanDDGHref(href string) string {
	if strings.HasPrefix(href, "//") {
		href = "https:" + href
	}
	if strings.Contains(href, "duckduckgo.com/l/") || strings.Contains(href, "uddg=") {
		if u, err := url.Parse(href); err == nil {
			if uddg := u.Query().Get("uddg"); uddg != "" {
				return uddg
			}
		}
		return ""
	}
	if strings.HasPrefix(href, "http") {
		return href
	}
	return ""
}

func searchBing(query string) ([]searchItem, error) {
	html, err := fetch("https://www.bing.com/search?q=" + url.QueryEscape(query) + "&setlang=zh-CN&count=12")
	if err != nil {
		return nil, err
	}
	var items []searchItem
	for _, block := range reBingBlock.FindAllString(html, 12) {
		hm := reH2A.FindStringSubmatch(block)
		if hm == nil {
			continue
		}
		title := stripTags(hm[2])
		href := hm[1]
		snippet := ""
		if pm := rePSnip.FindStringSubmatch(block); pm != nil {
			snippet = stripTags(pm[1])
		}
		if href == "" || !strings.HasPrefix(href, "http") || title == "" {
			continue
		}
		items = append(items, searchItem{Title: title, URL: href, Snippet: snippet})
	}
	if len(items) == 0 {
		return nil, errors.New("bing: no results")
	}
	return items, nil
}

func stripTags(s string) string {
	s = regexp.MustCompile(`<[^>]+>`).ReplaceAllString(s, "")
	return htmlUnescape(strings.TrimSpace(s))
}

func htmlUnescape(s string) string {
	r := strings.NewReplacer(
		"&amp;", "&", "&lt;", "<", "&gt;", ">", "&quot;", "\"",
		"&#x27;", "'", "&#39;", "'", "&nbsp;", " ", "&#x2F;", "/",
	)
	s = r.Replace(s)
	// 数字实体
	num := regexp.MustCompile(`&#(\d+);`)
	s = num.ReplaceAllStringFunc(s, func(m string) string {
		var n int
		fmt.Sscanf(m, "&#%d;", &n)
		if n > 0 && n < 0x110000 {
			return string(rune(n))
		}
		return ""
	})
	return strings.TrimSpace(s)
}

func hostOf(rawURL string) string {
	if u, err := url.Parse(rawURL); err == nil {
		return strings.ToLower(u.Hostname())
	}
	return ""
}

// Lookup 检索部件官方信息：优先调用 AI 模型，失败时回退搜索引擎摘要。
func Lookup(query string) (*Result, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, ErrNotFound
	}
	if r, err := lookupAI(query); err == nil {
		return r, nil
	}
	return lookupSearch(query)
}

// lookupSearch 搜索引擎兜底检索。
func lookupSearch(query string) (*Result, error) {
	pref := hintDomains(query)
	searchQuery := query + " 官方 规格 specifications"

	var items []searchItem
	if r, err := searchDDG(searchQuery); err == nil {
		items = r
	} else if r, err2 := searchBing(searchQuery); err2 == nil {
		items = r
	} else {
		return nil, ErrNotFound
	}

	pick := func(pool []searchItem) *searchItem {
		// 优先官方域名
		if len(pref) > 0 {
			for _, it := range pool {
				h := hostOf(it.URL)
				for _, d := range pref {
					if h == d || strings.HasSuffix(h, "."+d) {
						it := it
						return &it
					}
				}
			}
		}
		return nil
	}

	var chosen *searchItem
	if c := pick(items); c != nil {
		chosen = c
	} else if len(items) > 0 {
		chosen = &items[0]
	}
	if chosen == nil {
		return nil, ErrNotFound
	}
	if chosen.Snippet == "" {
		chosen.Snippet = chosen.Title
	}
	return &Result{
		Title:   chosen.Title,
		Snippet: chosen.Snippet,
		URL:     chosen.URL,
		Source:  hostOf(chosen.URL),
	}, nil
}
