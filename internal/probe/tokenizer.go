package probe

import (
	"context"
	"fmt"

	"modelspy/internal/client"
)

type tokenizer struct{}

func NewTokenizer() Probe { return tokenizer{} }

func (tokenizer) Name() string { return "tokenizer" }

func (tokenizer) Description() string {
	return "固定中英文语料的 prompt_tokens 比值探针(不同分词器特征差异大)"
}

const enText = "The quick brown fox jumps over the lazy dog, while the storm rolled across the quiet harbor and the sailors quietly wondered whether their tiny wooden boat would survive the terrible night ahead of them all."

const zhText = "人工智能技术近年来发展迅速,大语言模型在自然语言处理、代码生成和知识问答等任务上表现出色,但用户很难确认第三方接口背后真正运行的是哪一款模型。"

func (tokenizer) Run(ctx context.Context, x *Ctx) Result {
	r := newResult(tokenizer{})
	measure := func(text string) (int, int, error) {
		res, err := x.C.Chat(ctx, client.Request{
			Model:       x.Model,
			Messages:    []client.Message{{Role: "user", Content: text + "\n(不要回复任何内容,只要说 OK)"}},
			Temperature: 0,
			MaxTokens:   4,
		})
		if err != nil {
			return 0, 0, err
		}
		if res.PromptTokens <= 0 {
			return 0, 0, fmt.Errorf("usage 缺失,无法测量")
		}
		n := len([]rune(text))
		toks := res.PromptTokens - 10
		if toks < 1 {
			toks = 1
		}
		return toks, n, nil
	}
	tEN, nEN, err := measure(enText)
	if err != nil {
		r.Error = err.Error()
		return r
	}
	tZH, nZH, err := measure(zhText)
	if err != nil {
		r.Error = err.Error()
		return r
	}
	rEN, rZH := float64(tEN)/float64(nEN), float64(tZH)/float64(nZH)
	observe(&r, "EN tokens/char", fmt.Sprintf("%.3f", rEN))
	observe(&r, "ZH tokens/char", fmt.Sprintf("%.3f", rZH))
	for i := range x.DB.Candidates {
		c := &x.DB.Candidates[i]
		okEN, okZH := inRange(rEN, c.RatioEN), inRange(rZH, c.RatioZH)
		switch {
		case okEN && okZH:
			vote(&r, c.ID, 1.2)
		case okEN || okZH:
			vote(&r, c.ID, 0.2)
		default:
			vote(&r, c.ID, -1.0)
		}
	}
	return r
}
