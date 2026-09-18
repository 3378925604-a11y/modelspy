package probe

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"modelspy/internal/client"
	"modelspy/internal/db"
)

func mockBackend() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Model    string `json:"model"`
			Messages []struct {
				Content string `json:"content"`
			} `json:"messages"`
			Stream bool `json:"stream"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		q := req.Messages[0].Content
		var answer string
		tokens := len([]rune(q)) + 10
		switch {
		case strings.Contains(q, "strawberry"):
			answer = "2" // 模拟 4o 的著名错误
		case strings.Contains(q, "9.11"):
			answer = "9.11" // 模拟错误
		case strings.Contains(q, "还剩几只"):
			answer = "0"
		case strings.Contains(q, "剪掉一个角"):
			answer = "4"
		case strings.Contains(q, "天平"):
			answer = "否"
		case strings.Contains(q, "底层大模型"), strings.Contains(q, "underlying LLM"),
			strings.Contains(q, "模型 ID"), strings.Contains(q, "基础模型名称"):
			answer = "GPT-4o"
		case strings.Contains(q, "截止"):
			answer = "2024-06"
		case strings.Contains(q, "The quick brown fox"):
			answer = "OK"
			tokens = 70 // EN ratio ≈ 0.26
		case strings.Contains(q, "人工智能"):
			answer = "OK"
			tokens = 125 // ZH ratio ≈ 1.28
		case strings.Contains(q, "报数"):
			w.Header().Set("Content-Type", "text/event-stream")
			f := w.(http.Flusher)
			for i := 1; i <= 10; i++ {
				fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"content\":\"%d\\n\"}}],\"usage\":{\"prompt_tokens\":%d,\"completion_tokens\":%d}}\n\n", i, tokens, i)
				f.Flush()
				time.Sleep(10 * time.Millisecond)
			}
			fmt.Fprint(w, "data: [DONE]\n\n")
			return
		default:
			answer = "OK"
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"choices":[{"message":{"content":%q}}],"usage":{"prompt_tokens":%d,"completion_tokens":5}}`, answer, tokens)
	}))
}

func TestRunAllAgainstMock(t *testing.T) {
	srv := mockBackend()
	defer srv.Close()
	d := db.Load()
	x := &Ctx{
		C:     client.New(srv.URL, "k", 10*time.Second),
		Model: "gpt-4o",
		Expect: "gpt-4o",
		ExpectCand: d.Match("gpt-4o"),
		DB:    d,
	}
	probes := []Probe{NewIdentity(), NewTokenizer(), NewCapability(), NewCutoff(), NewSpeed(), NewToolcall()}
	results := RunAll(context.Background(), x, probes, nil)
	if len(results) != len(probes) {
		t.Fatalf("want %d results, got %d", len(probes), len(results))
	}
	total := map[string]float64{}
	for _, r := range results {
		if r.Error != "" {
			t.Logf("probe %s error: %s", r.Probe, r.Error)
		}
		for k, v := range r.Votes {
			total[k] += v
		}
	}
	if total["gpt-4o"] <= total["deepseek-chat"] || total["gpt-4o"] <= total["qwen"] {
		t.Fatalf("mock should be identified as gpt-4o, scores: %v", total)
	}
}

func TestCapabilityBuckets(t *testing.T) {
	for _, q := range traps {
		_ = q
	}
	if got := len(traps); got != 5 {
		t.Fatalf("expected 5 traps, got %d", got)
	}
}
