package client

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	Base string
	Key  string
	HC   *http.Client
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type Tool struct {
	Type     string          `json:"type"`
	Function json.RawMessage `json:"function"`
}

type Request struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Stream      bool      `json:"stream,omitempty"`
	Tools       []Tool    `json:"tools,omitempty"`
}

type Result struct {
	Text             string
	PromptTokens     int
	CompletionTokens int
	TTFT             time.Duration
	TPOT             time.Duration
	Latency          time.Duration
	ToolCalls        []ToolCall
}

type apiResp struct {
	Choices []struct {
		Message struct {
			Content   string     `json:"content"`
			ToolCalls []ToolCall `json:"tool_calls"`
		} `json:"message"`
		Delta struct {
			Content   string     `json:"content"`
			ToolCalls []ToolCall `json:"tool_calls"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage *struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}

func New(base, key string, timeout time.Duration) *Client {
	return &Client{
		Base: strings.TrimRight(base, "/"),
		Key:  key,
		HC:   &http.Client{Timeout: timeout},
	}
}

func (c *Client) do(ctx context.Context, body any) (*http.Response, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.Base+"/chat/completions", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.Key != "" {
		req.Header.Set("Authorization", "Bearer "+c.Key)
	}
	return c.HC.Do(req)
}

func parseError(status int, body []byte) error {
	s := string(body)
	if len(s) > 200 {
		s = s[:200]
	}
	return fmt.Errorf("HTTP %d: %s", status, s)
}

func (c *Client) Chat(ctx context.Context, r Request) (*Result, error) {
	if r.Stream {
		return c.chatStream(ctx, r)
	}
	return c.chatOnce(ctx, r)
}

func (c *Client) chatOnce(ctx context.Context, r Request) (*Result, error) {
	start := time.Now()
	resp, err := c.do(ctx, r)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, parseError(resp.StatusCode, body)
	}
	var ar apiResp
	if err := json.Unmarshal(body, &ar); err != nil {
		return nil, fmt.Errorf("invalid json: %v", err)
	}
	if len(ar.Choices) == 0 {
		return nil, fmt.Errorf("empty choices")
	}
	res := &Result{
		Text:     ar.Choices[0].Message.Content,
		ToolCalls: ar.Choices[0].Message.ToolCalls,
		Latency:  time.Since(start),
	}
	if ar.Usage != nil {
		res.PromptTokens = ar.Usage.PromptTokens
		res.CompletionTokens = ar.Usage.CompletionTokens
	}
	return res, nil
}

func (c *Client) chatStream(ctx context.Context, r Request) (*Result, error) {
	r.Stream = true
	start := time.Now()
	resp, err := c.do(ctx, r)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, parseError(resp.StatusCode, body)
	}
	ct := strings.ToLower(resp.Header.Get("Content-Type"))
	if ct != "" && !strings.Contains(ct, "text/event-stream") {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		var ar apiResp
		if err := json.Unmarshal(body, &ar); err != nil {
			return nil, fmt.Errorf("invalid json: %v", err)
		}
		if len(ar.Choices) == 0 {
			return nil, fmt.Errorf("empty choices")
		}
		res := &Result{
			Text:      ar.Choices[0].Message.Content,
			ToolCalls: ar.Choices[0].Message.ToolCalls,
			Latency:   time.Since(start),
		}
		if ar.Usage != nil {
			res.PromptTokens = ar.Usage.PromptTokens
			res.CompletionTokens = ar.Usage.CompletionTokens
		}
		return res, nil
	}

	var sb strings.Builder
	res := &Result{}
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	var firstChunk, lastChunk time.Duration
	seen := false
	chunks := 0
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			break
		}
		var ar apiResp
		if json.Unmarshal([]byte(data), &ar) != nil {
			continue
		}
		if len(ar.Choices) > 0 {
			d := ar.Choices[0].Delta.Content
			if d == "" {
				d = ar.Choices[0].Message.Content
			}
			if len(ar.Choices[0].Message.ToolCalls) > 0 {
				res.ToolCalls = append(res.ToolCalls, ar.Choices[0].Message.ToolCalls...)
			}
			if d != "" {
				now := time.Since(start)
				if !seen {
					firstChunk = now
					seen = true
				}
				lastChunk = now
				chunks++
				sb.WriteString(d)
			}
		}
		if ar.Usage != nil {
			res.PromptTokens = ar.Usage.PromptTokens
			res.CompletionTokens = ar.Usage.CompletionTokens
		}
	}
	res.Text = sb.String()
	res.Latency = time.Since(start)
	res.TTFT = firstChunk
	if chunks > 1 {
		res.TPOT = (lastChunk - firstChunk) / time.Duration(chunks-1)
	}
	if res.Text == "" && len(res.ToolCalls) == 0 {
		return nil, fmt.Errorf("empty stream")
	}
	return res, nil
}
