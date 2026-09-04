package webinfo

// AI 模型检索：调用 OpenAI 兼容接口获取部件官方资料。
// 内置凭据为加密存储（见 ai_crypto.go），可用 ai.json 覆盖（见 README 二次修改须知）：
//   {"baseUrl": "...", "apiKey": "...", "model": "..."}

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	defaultAIBaseURL = "https://apihub.agnes-ai.com/v1"
	defaultAIModel   = "agnes-2.5-flash"
)

// AIConfig 接口配置。
type AIConfig struct {
	BaseURL string `json:"baseUrl"`
	APIKey  string `json:"apiKey"`
	Model   string `json:"model"`
}

// LoadAIConfig 读取配置（文件覆盖内置默认值；内置 token 为加密存储）。
// 查找顺序：①与 EXE 同目录的 ai.json（便携）②%ProgramData%\WhatIsMyPC\ai.json
func LoadAIConfig() AIConfig {
	cfg := AIConfig{BaseURL: defaultAIBaseURL, APIKey: decryptAIToken(), Model: defaultAIModel}

	var candidates []string
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), "ai.json"))
	}
	if pd := os.Getenv("ProgramData"); pd != "" {
		candidates = append(candidates, filepath.Join(pd, "WhatIsMyPC", "ai.json"))
	}

	for _, p := range candidates {
		b, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		var f AIConfig
		if json.Unmarshal(b, &f) != nil {
			continue
		}
		if f.BaseURL != "" {
			cfg.BaseURL = f.BaseURL
		}
		if f.APIKey != "" {
			cfg.APIKey = f.APIKey
		}
		if f.Model != "" {
			cfg.Model = f.Model
		}
		break
	}
	return cfg
}

const aiSystemPrompt = `你是硬件资料助手。根据用户给出的硬件型号，给出其官方资料摘要。
严格只输出一个 JSON 对象，不要输出任何其他内容，不要用 markdown 代码块包裹：
{"title":"官方名称","summary":"官方介绍的要点摘要，80-200字，包含关键规格与产品定位，不得编造数据","url":"官方产品页URL，不确定则留空字符串"}`

type aiReply struct {
	Title   string `json:"title"`
	Summary string `json:"summary"`
	URL     string `json:"url"`
}

var aiClient = &http.Client{Timeout: 45 * time.Second}

// lookupAI 调用 AI 模型获取官方资料。
func lookupAI(query string) (*Result, error) {
	cfg := LoadAIConfig()

	reqBody, err := json.Marshal(map[string]interface{}{
		"model": cfg.Model,
		"messages": []map[string]string{
			{"role": "system", "content": aiSystemPrompt},
			{"role": "user", "content": query},
		},
		"temperature": 0.3,
	})
	if err != nil {
		return nil, err
	}

	url := strings.TrimRight(cfg.BaseURL, "/") + "/chat/completions"
	req, err := http.NewRequest("POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := aiClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("AI 接口错误: http %d %s", resp.StatusCode, truncate(string(raw), 160))
	}

	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("AI 响应解析失败: %v", err)
	}
	if parsed.Error != nil {
		return nil, fmt.Errorf("AI 接口错误: %s", parsed.Error.Message)
	}
	if len(parsed.Choices) == 0 {
		return nil, fmt.Errorf("AI 响应为空")
	}

	content := parsed.Choices[0].Message.Content
	var reply aiReply
	if err := json.Unmarshal([]byte(extractJSON(content)), &reply); err != nil {
		return nil, fmt.Errorf("AI 内容解析失败: %v", err)
	}
	if strings.TrimSpace(reply.Summary) == "" {
		return nil, fmt.Errorf("AI 未返回有效内容")
	}

	title := strings.TrimSpace(reply.Title)
	if title == "" {
		title = query
	}
	u := strings.TrimSpace(reply.URL)
	if u != "" && !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") {
		u = ""
	}
	return &Result{
		Title:   title,
		Snippet: strings.TrimSpace(reply.Summary),
		URL:     u,
		Source:  cfg.Model + " · AI 摘要",
	}, nil
}

// extractJSON 从模型输出中提取 JSON 对象（容忍代码块包裹与前后杂讯）。
func extractJSON(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.Index(s, "```"); i >= 0 {
		s = s[i:]
		if j := strings.Index(s, "\n"); j >= 0 {
			s = s[j+1:]
		}
		if k := strings.Index(s, "```"); k >= 0 {
			s = s[:k]
		}
	}
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start >= 0 && end > start {
		return s[start : end+1]
	}
	return s
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
