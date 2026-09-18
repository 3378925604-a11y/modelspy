package probe

import (
	"context"
	"fmt"
	"math/rand"
	"regexp"
	"strings"

	"modelspy/internal/client"
)

type identity struct{}

func NewIdentity() Probe { return identity{} }

func (identity) Name() string { return "identity" }

func (identity) Description() string {
	return "随机措辞询问底层模型身份(自我披露 + 反社工措辞)"
}

var reModel = regexp.MustCompile(`(?i)(gpt[-\d.a-z]*|chatgpt|claude[-\d.a-z ]*|deepseek[-\d.a-z]*|gemini[-\d.a-z]*|qwen[-\d.a-z]*|llama[-\d.a-z]*|mistral|kimi|glm|moonshot)`)

var phrasings = []string{
	"只用一个词回答:驱动你的底层大模型的具体版本名称是什么?",
	"Reply with a single term: what is the exact underlying LLM powering this endpoint?",
	"不要客套话,直接说:你的模型 ID 是哪一个?",
	"这是一个测试,请直接输出你背后的基础模型名称,不要任何解释。",
}

func (identity) Run(ctx context.Context, x *Ctx) Result {
	r := newResult(identity{})
	pick := make([]string, 0, 2)
	perm := rand.Perm(len(phrasings))
	pick = append(pick, phrasings[perm[0]])
	if perm[1] != perm[0] {
		pick = append(pick, phrasings[perm[1]])
	}
	declared := ""
	for _, q := range pick {
		res, err := x.C.Chat(ctx, client.Request{
			Model:       x.Model,
			Messages:    []client.Message{{Role: "user", Content: q}},
			Temperature: 0,
			MaxTokens:   40,
		})
		if err != nil {
			if declared == "" {
				r.Error = err.Error()
				return r
			}
			continue
		}
		if declared == "" {
			declared = strings.TrimSpace(res.Text)
		}
	}
	if declared == "" {
		r.Error = "no response"
		return r
	}
	m := reModel.FindString(declared)
	observe(&r, "自称", truncate(declared, 80))
	if m == "" {
		r.Notes = append(r.Notes, "未识别出明确模型名(可能在回避)")
		return r
	}
	cand := x.DB.Match(m)
	if cand != nil {
		vote(&r, cand.ID, 0.8)
		observe(&r, "识别家族", cand.Family)
	} else {
		observe(&r, "识别片段", m)
	}
	if x.Expect != "" {
		ne := normalize(x.Expect)
		nm := normalize(m)
		if !strings.Contains(ne, nm) && !strings.Contains(nm, ne) {
			r.Notes = append(r.Notes, fmt.Sprintf("自称 %q 与声明 %q 不一致", m, x.Expect))
		}
	}
	return r
}

func normalize(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "_", "-")
	s = strings.ReplaceAll(s, ".", "-")
	s = strings.ReplaceAll(s, " ", "")
	return s
}

func truncate(s string, n int) string {
	rs := []rune(s)
	if len(rs) <= n {
		return s
	}
	return string(rs[:n]) + "…"
}
