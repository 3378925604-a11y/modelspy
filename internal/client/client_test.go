package client

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestChatOnce(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Errorf("bad path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"choices":[{"message":{"content":"hello"}}],"usage":{"prompt_tokens":12,"completion_tokens":3}}`)
	}))
	defer srv.Close()
	c := New(srv.URL, "test-key", 5*time.Second)
	res, err := c.Chat(context.Background(), Request{Model: "gpt-4o", Messages: []Message{{Role: "user", Content: "hi"}}})
	if err != nil {
		t.Fatal(err)
	}
	if res.Text != "hello" || res.PromptTokens != 12 || res.CompletionTokens != 3 {
		t.Fatalf("bad result %+v", res)
	}
}

func TestChatStream(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher := w.(http.Flusher)
		for i := 0; i < 6; i++ {
			fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"content\":\"tok%d \"}}]}\n\n", i)
			flusher.Flush()
			time.Sleep(30 * time.Millisecond)
		}
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()
	c := New(srv.URL, "", 5*time.Second)
	res, err := c.Chat(context.Background(), Request{Model: "m", Messages: []Message{{Role: "user", Content: "x"}}, Stream: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.TTFT <= 0 || res.TPOT <= 0 || len(res.Text) < 6 {
		t.Fatalf("bad timing %+v", res)
	}
}

func TestStreamFallbackJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"choices":[{"message":{"content":"fallback"}}]}`)
	}))
	defer srv.Close()
	c := New(srv.URL, "", 5*time.Second)
	res, err := c.Chat(context.Background(), Request{Model: "m", Messages: []Message{{Role: "user", Content: "x"}}, Stream: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.Text != "fallback" {
		t.Fatalf("bad result %+v", res)
	}
}

func TestHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		fmt.Fprint(w, `{"error":"invalid key"}`)
	}))
	defer srv.Close()
	c := New(srv.URL, "", 5*time.Second)
	_, err := c.Chat(context.Background(), Request{Model: "m", Messages: []Message{{Role: "user", Content: "x"}}})
	if err == nil {
		t.Fatal("expected error")
	}
}
