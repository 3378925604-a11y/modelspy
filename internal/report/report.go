package report

import (
	"fmt"
	"html/template"
	"os"
	"sort"
	"strings"

	"modelspy/internal/db"
	"modelspy/internal/probe"
)

type Ranked struct {
	ID         string  `json:"id"`
	Score      float64 `json:"score"`
	Confidence float64 `json:"confidence"`
}

type Verdict struct {
	Expect     string   `json:"expect"`
	Top        []Ranked `json:"top"`
	Matches    bool     `json:"matches"`
	Flag       string   `json:"flag"`
	FlagDetail string   `json:"flag_detail"`
}

type Report struct {
	Target  string        `json:"target"`
	Model   string        `json:"model"`
	Version string        `json:"version"`
	Results []probe.Result `json:"results"`
	Verdict Verdict       `json:"verdict"`
}

func Build(baseURL, model, expect string, version string, results []probe.Result, d *db.DB) Report {
	total := map[string]float64{}
	for _, r := range results {
		for k, v := range r.Votes {
			total[k] += v
		}
	}
	type sv struct {
		id    string
		score float64
	}
	var list []sv
	for k, v := range total {
		if k == "openai-compatible" {
			continue
		}
		list = append(list, sv{k, v})
	}
	sort.Slice(list, func(i, j int) bool { return list[i].score > list[j].score })

	var ranked []Ranked
	sum := 0.0
	for _, s := range list {
		if s.score > 0 {
			sum += s.score
		}
	}
	for i, s := range list {
		if i >= 3 {
			break
		}
		conf := 0.0
		if sum > 0 && s.score > 0 {
			conf = s.score / sum
		}
		ranked = append(ranked, Ranked{ID: s.id, Score: s.score, Confidence: conf})
	}

	v := Verdict{Expect: expect, Top: ranked}
	if expect == "" {
		v.Expect = model
	}
	if len(ranked) == 0 {
		v.Flag = "INCONCLUSIVE"
		v.FlagDetail = "证据不足,无法判定"
	} else {
		exp := d.Match(v.Expect)
		if exp == nil {
			v.Flag = "UNKNOWN_EXPECT"
			v.FlagDetail = fmt.Sprintf("声明的 %q 不在指纹库中,只能给出识别排名", v.Expect)
		} else {
			v.Matches = ranked[0].ID == exp.ID
			if v.Matches && ranked[0].Confidence < 0.6 {
				v.Flag = "MATCHES_LOW_CONF"
				v.FlagDetail = "与声明一致,但置信度偏低(证据相互矛盾或库未校准)"
			} else if v.Matches {
				v.Flag = "MATCHES"
				v.FlagDetail = "各项特征与声明模型一致"
			} else if ranked[0].Confidence >= 0.6 {
				v.Flag = "MISMATCH"
				v.FlagDetail = fmt.Sprintf("识别结果 %s(置信度 %.0f%%)与声明 %s 不符,疑似换壳", ranked[0].ID, ranked[0].Confidence*100, v.Expect)
			} else {
				v.Flag = "SUSPECT"
				v.FlagDetail = fmt.Sprintf("与声明不符,但置信度不足(%.0f%%),建议多跑几轮", ranked[0].Confidence*100)
			}
		}
	}
	return Report{Target: baseURL, Model: model, Version: version, Results: results, Verdict: v}
}

func flagLabel(f string) string {
	switch f {
	case "MATCHES":
		return "✅ 一致"
	case "MATCHES_LOW_CONF":
		return "⚠️ 一致(低置信)"
	case "MISMATCH":
		return "❌ 不一致(疑似换壳)"
	case "SUSPECT":
		return "⚠️ 可疑"
	case "UNKNOWN_EXPECT":
		return "ℹ️ 声明不在库中"
	default:
		return "❓ 无法判定"
	}
}

func Print(w *strings.Builder, rep Report) {
	fmt.Fprintln(w, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Fprintf(w, " ModelSpy %s  目标: %s (model=%s)\n", rep.Version, rep.Target, rep.Model)
	fmt.Fprintln(w, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	for _, r := range rep.Results {
		status := "✔"
		if r.Error != "" {
			status = "✘ " + r.Error
		} else if r.Skipped {
			status = "–"
		}
		fmt.Fprintf(w, "\n▸ %s [%s] %s\n", r.Probe, status, r.Description)
		for _, o := range r.Observations {
			fmt.Fprintf(w, "    %s: %s\n", o.Key, o.Value)
		}
		for _, n := range r.Notes {
			fmt.Fprintf(w, "    · %s\n", n)
		}
	}
	fmt.Fprintln(w, "\n──────── 候选排名 ────────")
	for _, t := range rep.Verdict.Top {
		bar := strings.Repeat("█", int(t.Confidence*20))
		fmt.Fprintf(w, "  %-16s %5.0f%%  %s\n", t.ID, t.Confidence*100, bar)
	}
	if len(rep.Verdict.Top) == 0 {
		fmt.Fprintln(w, "  (无)")
	}
	fmt.Fprintf(w, "\n结论: %s  %s\n", flagLabel(rep.Verdict.Flag), rep.Verdict.FlagDetail)
}

const htmlTmpl = `<!DOCTYPE html>
<html lang="zh-CN"><head><meta charset="utf-8"><title>ModelSpy 报告 - {{.Model}}</title>
<style>
body{font-family:system-ui,"Microsoft YaHei",sans-serif;max-width:860px;margin:40px auto;color:#222;padding:0 16px}
h1{font-size:22px}.verdict{padding:14px 18px;border-radius:10px;font-size:16px;font-weight:600;margin:16px 0}
.ok{background:#e6f6e6;border:1px solid #3a3}.bad{background:#fdecec;border:1px solid #c33}.warn{background:#fff7e0;border:1px solid #ca0}.info{background:#eef3fb;border:1px solid #68c}
table{border-collapse:collapse;width:100%;margin:10px 0}td,th{border:1px solid #ddd;padding:6px 10px;font-size:14px;text-align:left}
.probe{border-left:3px solid #888;padding-left:12px;margin:18px 0}
.bar{background:#4a90d9;height:14px;border-radius:3px;display:inline-block;vertical-align:middle}
.mono{font-family:ui-monospace,Consolas,monospace;font-size:13px}
small{color:#888}
</style></head><body>
<h1>ModelSpy {{.Version}} 检测报告</h1>
<p>目标: <span class="mono">{{.Target}}</span> · 请求模型: <span class="mono">{{.Model}}</span> · 声明期望: <span class="mono">{{.Verdict.Expect}}</span></p>
<div class="verdict {{if eq .Verdict.Flag "MATCHES"}}ok{{else if eq .Verdict.Flag "MISMATCH"}}bad{{else if or (eq .Verdict.Flag "SUSPECT") (eq .Verdict.Flag "MATCHES_LOW_CONF")}}warn{{else}}info{{end}}">
{{.Verdict.FlagDetail}}</div>
<h2>候选排名</h2>
<table><tr><th>候选</th><th>得分</th><th>置信度</th><th></th></tr>
{{range .Verdict.Top}}<tr><td>{{.ID}}</td><td>{{printf "%.2f" .Score}}</td><td>{{printf "%.0f%%" (mulf .Confidence 100.0)}}</td><td><span class="bar" style="width:{{printf "%.0f" (mulf .Confidence 200.0)}}px"></span></td></tr>
{{end}}</table>
<h2>探针明细</h2>
{{range .Results}}
<div class="probe"><b>{{.Probe}}</b> — {{.Description}}
{{if .Error}}<p style="color:#c33">✘ {{.Error}}</p>{{end}}
<table>
{{range .Observations}}<tr><td>{{.Key}}</td><td class="mono">{{.Value}}</td></tr>{{end}}
</table>
{{range .Notes}}<p><small>· {{.}}</small></p>{{end}}
</div>
{{end}}
<p><small>ModelSpy 为启发式统计识别工具,结论具有概率性;"实时切换路由"的中转可通过抽检多次检测。指纹库 v1 为社区种子数据。</small></p>
</body></html>`

var tmpl = template.Must(template.New("r").Funcs(template.FuncMap{
	"mulf": func(a, b float64) float64 { return a * b },
}).Parse(htmlTmpl))

func WriteHTML(path string, rep Report) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return tmpl.Execute(f, rep)
}
