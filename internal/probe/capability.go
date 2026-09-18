package probe

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"modelspy/internal/client"
)

type capability struct{}

func NewCapability() Probe { return capability{} }

func (capability) Name() string { return "capability" }

func (capability) Description() string {
	return "temperature=0 已知陷阱题库,按得分划分强/中/弱档(模型代际指纹)"
}

type trapQ struct {
	prompt string
	check  func(string) bool
}

var reDigit = regexp.MustCompile(`\d+`)

var traps = []trapQ{
	{
		prompt: "只回答一个阿拉伯数字,不要任何其他文字:strawberry 这个英文单词里一共有几个字母 r?",
		check:  func(s string) bool { return reDigit.FindString(s) == "3" },
	},
	{
		prompt: "只回答 \"9.11\" 或 \"9.9\" 中的一个,不要其他文字:9.11 和 9.9 哪个更大?",
		check:  func(s string) bool { return strings.Contains(s, "9.9") && !strings.Contains(s, "9.11") },
	},
	{
		prompt: "树上站着 10 只鸟,猎人开枪打死其中 1 只,树上还剩几只鸟?只回答一个阿拉伯数字。",
		check:  func(s string) bool { return strings.HasPrefix(strings.TrimSpace(s), "0") },
	},
	{
		prompt: "一个三角形纸片,沿直线剪掉一个角,剩下的图形还剩几个角?只回答一个阿拉伯数字。",
		check:  func(s string) bool { return reDigit.FindString(s) == "4" },
	},
	{
		prompt: "只回答 \"是\" 或 \"否\":把 1 千克棉花和 1 千克铁同时放到天平两端,棉花那一端会更高。",
		check:  func(s string) bool { return strings.Contains(s, "否") },
	},
}

func (capability) Run(ctx context.Context, x *Ctx) Result {
	r := newResult(capability{})
	scored := 0
	correct := 0
	for _, q := range traps {
		res, err := x.C.Chat(ctx, client.Request{
			Model:       x.Model,
			Messages:    []client.Message{{Role: "user", Content: q.prompt}},
			Temperature: 0,
			MaxTokens:   30,
		})
		if err != nil {
			continue
		}
		scored++
		ok := q.check(res.Text)
		if ok {
			correct++
		}
		observe(&r, "题目"+runeToStr(rune('A'+scored-1)), verdict(ok, res.Text))
	}
	if scored == 0 {
		r.Error = "全部请求失败"
		return r
	}
	bucket := "weak"
	switch {
	case correct >= 4:
		bucket = "strong"
	case correct >= 3:
		bucket = "medium"
	}
	observe(&r, "得分", fmt.Sprintf("%d/%d → %s", correct, scored, bucket))
	for i := range x.DB.Candidates {
		c := &x.DB.Candidates[i]
		if c.Capability == bucket {
			vote(&r, c.ID, 1.5)
		} else if diff := absDiff(c.Capability, bucket); diff == 1 {
			vote(&r, c.ID, -0.6)
		} else {
			vote(&r, c.ID, -1.2)
		}
	}
	return r
}

func absDiff(a, b string) int {
	order := map[string]int{"weak": 0, "medium": 1, "strong": 2}
	d := order[a] - order[b]
	if d < 0 {
		return -d
	}
	return d
}

func verdict(ok bool, out string) string {
	s := "✗ " + truncate(strings.TrimSpace(out), 30)
	if ok {
		s = "✓ " + truncate(strings.TrimSpace(out), 30)
	}
	return s
}

func runeToStr(r rune) string { return string([]rune{r}) }
