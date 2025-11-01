package clients

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

/*************** Public types ***************/

// DeepSeek 定义：OpenAI 兼容的 Chat API 客户端
type DeepSeek interface {
	ChatOnce(ctx context.Context, model string, messages []ChatMessage, opt *ChatOptions) (string, error)
	Stream(ctx context.Context, model string, messages []ChatMessage, opt *ChatOptions, onDelta func(string)) (string, error)
}

type ChatMessage struct {
	Role    string `json:"role"`    // "system" | "user" | "assistant"
	Content string `json:"content"` // 纯文本足够；需要多模态可扩展
}

type ChatOptions struct {
	Temperature *float64 `json:"temperature,omitempty"`
	TopP        *float64 `json:"top_p,omitempty"`
	MaxTokens   *int     `json:"max_tokens,omitempty"`
	Stop        []string `json:"stop,omitempty"`
	// 允许附加原生参数（如 logit_bias, response_format...）
	Extra map[string]any `json:"-"` // 不序列化到 body 根级；见 buildBody
	// 自定义 header（如路由标签等）
	Headers map[string]string `json:"-"`
	// 覆盖默认超时（默认 60s；流式会建立长链接，但读写仍受 ctx 控制）
	Timeout time.Duration `json:"-"`
}

type Prompt struct {
	Messages []ChatMessage
	Options  *ChatOptions
}

/*************** Client impl ***************/

type deepSeekClient struct {
	base *url.URL
	key  string
	http *http.Client
	// path 可配置（默认 /v1/chat/completions）
	chatPath string
}

func NewDeepSeek(baseURL, apiKey string, opts ...func(*deepSeekClient)) (DeepSeek, error) {
	if baseURL == "" {
		baseURL = "https://api.deepseek.com"
	}
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("deepseek: invalid baseURL: %w", err)
	}
	c := &deepSeekClient{
		base:     u,
		key:      apiKey,
		http:     &http.Client{Timeout: 60 * time.Second},
		chatPath: "/chat/completions",
	}
	for _, f := range opts {
		f(c)
	}
	return c, nil
}

// 可选：自定义路径或 http.Client
func WithChatPath(p string) func(*deepSeekClient) {
	return func(c *deepSeekClient) { c.chatPath = p }
}
func WithHTTPClient(h *http.Client) func(*deepSeekClient) {
	return func(c *deepSeekClient) { c.http = h }
}

/*************** Non-stream ***************/

func (c *deepSeekClient) ChatOnce(ctx context.Context, model string, messages []ChatMessage, opt *ChatOptions) (string, error) {
	body, headers := c.buildBody(model, messages, opt, false /*stream*/)
	endpoint := c.endpoint()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("deepseek %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	var out chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	if len(out.Choices) == 0 {
		return "", nil
	}
	return out.Choices[0].Message.Content, nil
}

/*************** Stream ***************/

func (c *deepSeekClient) Stream(ctx context.Context, model string, messages []ChatMessage, opt *ChatOptions, onDelta func(string)) (string, error) {
	body, headers := c.buildBody(model, messages, opt, true /*stream*/)
	endpoint := c.endpoint()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	if resp.StatusCode >= 300 {
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("deepseek %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}

	defer resp.Body.Close()
	reader := bufio.NewReader(resp.Body)
	var full strings.Builder

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return full.String(), err
		}
		line = strings.TrimSpace(line)
		// SSE: 只关心以 "data: " 开头
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			break
		}
		if data == "" {
			continue
		}
		var ev streamChunk
		if err := json.Unmarshal([]byte(data), &ev); err != nil {
			// 忽略非 JSON 行
			continue
		}
		if len(ev.Choices) == 0 {
			continue
		}
		delta := ev.Choices[0].Delta.Content
		if delta != "" {
			full.WriteString(delta)
			if onDelta != nil {
				onDelta(delta)
			}
		}
	}
	return full.String(), nil
}

/*************** internals ***************/

func (c *deepSeekClient) endpoint() string {
	u := *c.base
	// 确保 path 拼接正确
	if !strings.HasPrefix(c.chatPath, "/") {
		u.Path += "/" + c.chatPath
	} else {
		u.Path += c.chatPath
	}
	return u.String()
}

func (c *deepSeekClient) buildBody(model string, messages []ChatMessage, opt *ChatOptions, stream bool) ([]byte, map[string]string) {
	if opt == nil {
		opt = &ChatOptions{}
	}
	// 根 payload
	payload := map[string]any{
		"model":    model,
		"messages": messages,
		"stream":   stream,
	}
	if opt.Temperature != nil {
		payload["temperature"] = *opt.Temperature
	}
	if opt.TopP != nil {
		payload["top_p"] = *opt.TopP
	}
	if opt.MaxTokens != nil {
		payload["max_tokens"] = *opt.MaxTokens
	}
	if len(opt.Stop) > 0 {
		payload["stop"] = opt.Stop
	}
	// 附加扩展字段
	for k, v := range opt.Extra {
		payload[k] = v
	}
	b, _ := json.Marshal(payload)

	headers := map[string]string{
		"Authorization": "Bearer " + c.key,
		"Content-Type":  "application/json",
		"Accept":        "application/json",
	}
	// 流式时 DeepSeek / OpenAI 会返回 SSE data: 行；Accept 可不强制 text/event-stream
	for k, v := range opt.Headers {
		headers[k] = v
	}
	return b, headers
}

/*************** wire types ***************/

// 非流式响应
type chatResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index        int         `json:"index"`
		Message      ChatMessage `json:"message"`
		FinishReason string      `json:"finish_reason"`
	} `json:"choices"`
	// usage/token 信息可按需添加
}

// 流式 chunk（SSE data:）
type streamChunk struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index int `json:"index"`
		Delta struct {
			Role    string `json:"role,omitempty"`
			Content string `json:"content,omitempty"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}
