package probe

import (
	"context"
	"encoding/json"
	"strings"

	"modelspy/internal/client"
)

type toolcall struct{}

func NewToolcall() Probe { return toolcall{} }

func (toolcall) Name() string { return "toolcall" }

func (toolcall) Description() string {
	return "Function calling 完整性检查:是否返回结构合法的工具调用(中转常阉割)"
}

func (toolcall) Run(ctx context.Context, x *Ctx) Result {
	r := newResult(toolcall{})
	fn := json.RawMessage(`{"name":"get_weather","description":"查询指定城市当前天气","parameters":{"type":"object","properties":{"city":{"type":"string","description":"城市名"}},"required":["city"]}}`)
	res, err := x.C.Chat(ctx, client.Request{
		Model:       x.Model,
		Messages:    []client.Message{{Role: "user", Content: "北京现在天气怎么样?"}},
		Temperature: 0,
		MaxTokens:   80,
		Tools: []client.Tool{{
			Type:     "function",
			Function: fn,
		}},
	})
	if err != nil {
		r.Error = err.Error()
		return r
	}
	if len(res.ToolCalls) == 0 {
		observe(&r, "返回", "未触发工具调用(直接文字回答:"+truncate(strings.TrimSpace(res.Text), 40)+")")
		r.Notes = append(r.Notes, "Function calling 不完整,可能是低配中转或模型本身不支持")
		return r
	}
	tc := res.ToolCalls[0]
	observe(&r, "工具名", tc.Function.Name)
	var args map[string]any
	if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
		observe(&r, "arguments", "非法 JSON: "+truncate(tc.Function.Arguments, 40))
		r.Notes = append(r.Notes, "arguments 不是合法 JSON")
		return r
	}
	observe(&r, "arguments", "合法 JSON")
	if tc.Function.Name == "get_weather" {
		vote(&r, "openai-compatible", 0.3)
	}
	return r
}
