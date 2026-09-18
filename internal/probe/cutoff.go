package probe

import (
	"context"
	"fmt"
	"regexp"
	"strconv"

	"modelspy/internal/client"
)

type cutoff struct{}

func NewCutoff() Probe { return cutoff{} }

func (cutoff) Name() string { return "cutoff" }

func (cutoff) Description() string {
	return "询问训练知识截止日期并与候选模型比对(权重低,仅作辅助)"
}

var reDate = regexp.MustCompile(`(\d{4})[-/年.](\d{1,2})?`)

func (cutoff) Run(ctx context.Context, x *Ctx) Result {
	r := newResult(cutoff{})
	res, err := x.C.Chat(ctx, client.Request{
		Model:       x.Model,
		Messages:    []client.Message{{Role: "user", Content: "你的训练知识截止日期是哪年哪月?只回答日期,格式 YYYY-MM。不知道就回答\"不知道\"。"}},
		Temperature: 0,
		MaxTokens:   30,
	})
	if err != nil {
		r.Error = err.Error()
		return r
	}
	m := reDate.FindStringSubmatch(res.Text)
	if m == nil {
		observe(&r, "回答", truncate(res.Text, 60))
		r.Notes = append(r.Notes, "未给出可解析的日期")
		return r
	}
	year, _ := strconv.Atoi(m[1])
	month := 0
	if m[2] != "" {
		month, _ = strconv.Atoi(m[2])
	}
	val := fmt.Sprintf("%04d-%02d", year, month)
	observe(&r, "自称截止", val)
	for i := range x.DB.Candidates {
		c := &x.DB.Candidates[i]
		cy := 0
		fmt.Sscanf(c.Cutoff, "%d", &cy)
		switch {
		case year > cy+1:
			vote(&r, c.ID, -0.2)
		case year == cy || year == cy-1:
			vote(&r, c.ID, 0.3)
		default:
			vote(&r, c.ID, -0.1)
		}
	}
	return r
}
