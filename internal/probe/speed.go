package probe

import (
	"context"
	"fmt"
	"sort"
	"time"

	"modelspy/internal/client"
)

type speed struct{}

func NewSpeed() Probe { return speed{} }

func (speed) Name() string { return "speed" }

func (speed) Description() string {
	return "流式输出 TPOT 侧信道:生成速度特征与小模型/大模型的相关性"
}

func (speed) Run(ctx context.Context, x *Ctx) Result {
	r := newResult(speed{})
	const prompt = "请从 1 开始逐行报数,一直报到 80,每行一个数字,不要任何其他文字。"
	var tpots []float64
	var ttfts []float64
	for i := 0; i < 2; i++ {
		res, err := x.C.Chat(ctx, client.Request{
			Model:       x.Model,
			Messages:    []client.Message{{Role: "user", Content: prompt}},
			Temperature: 0,
			MaxTokens:   260,
			Stream:      true,
		})
		if err != nil {
			if len(tpots) == 0 {
				r.Error = err.Error()
				return r
			}
			continue
		}
		if res.TPOT > 0 {
			tpots = append(tpots, float64(res.TPOT.Microseconds())/1000.0)
		}
		if res.TTFT > 0 {
			ttfts = append(ttfts, float64(res.TTFT.Milliseconds()))
		}
	}
	if len(tpots) == 0 {
		r.Error = "接口不支持流式输出或无有效时间数据"
		return r
	}
	sort.Float64s(tpots)
	median := tpots[len(tpots)/2]
	observe(&r, "中位 TPOT", fmt.Sprintf("%.1f ms/token", median))
	if len(ttfts) > 0 {
		observe(&r, "TTFT", fmt.Sprintf("%.0f ms", ttfts[0]))
	}
	for i := range x.DB.Candidates {
		c := &x.DB.Candidates[i]
		switch {
		case median >= c.TPOTMsMin && median <= c.TPOTMsMax:
			vote(&r, c.ID, 0.4)
		case median < c.TPOTMsMin*0.5 && c.TPOTMsMin >= 10:
			vote(&r, c.ID, -0.6)
			r.Notes = append(r.Notes, fmt.Sprintf("生成速度(%.0fms/token)明显快于 %s 的典型范围,更像小模型", median, c.ID))
		default:
			vote(&r, c.ID, -0.1)
		}
	}
	_ = time.Now
	return r
}
