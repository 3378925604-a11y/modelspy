package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"modelspy/internal/client"
	"modelspy/internal/db"
	"modelspy/internal/probe"
	"modelspy/internal/report"
)

const version = "v0.1.0"

func main() {
	base := flag.String("base", "", "OpenAI 兼容接口地址,如 https://api.openai.com/v1")
	key := flag.String("key", "", "API key(默认读环境变量 OPENAI_API_KEY)")
	model := flag.String("model", "", "调用的模型名,如 gpt-4o")
	expect := flag.String("expect", "", "声明的真实模型(默认同 --model)")
	only := flag.String("probes", "all", "逗号分隔探针: identity,tokenizer,capability,cutoff,speed,toolcall")
	timeout := flag.Int("timeout", 90, "单次请求超时秒数")
	jsonOut := flag.Bool("json", false, "输出 JSON")
	htmlOut := flag.String("html", "", "同时写出 HTML 报告")
	list := flag.Bool("list-probes", false, "列出探针")
	vflag := flag.Bool("version", false, "显示版本")
	flag.Parse()

	if *vflag {
		fmt.Println("modelspy", version)
		return
	}
	allProbes := []probe.Probe{
		probe.NewIdentity(), probe.NewTokenizer(), probe.NewCapability(),
		probe.NewCutoff(), probe.NewSpeed(), probe.NewToolcall(),
	}
	if *list {
		for _, p := range allProbes {
			fmt.Printf("%-12s %s\n", p.Name(), p.Description())
		}
		return
	}
	if *base == "" || *model == "" {
		fmt.Fprintln(os.Stderr, "用法: modelspy -base https://xxx/v1 -model gpt-4o [-expect gpt-4o] [-key sk-...]")
		fmt.Fprintln(os.Stderr, "      key 可用环境变量 OPENAI_API_KEY 提供;详见 README.md")
		os.Exit(2)
	}
	k := *key
	if k == "" {
		k = os.Getenv("OPENAI_API_KEY")
	}
	e := *expect
	if e == "" {
		e = *model
	}

	sel := allProbes
	if *only != "all" {
		want := map[string]bool{}
		for _, s := range strings.Split(*only, ",") {
			want[strings.TrimSpace(s)] = true
		}
		sel = nil
		for _, p := range allProbes {
			if want[p.Name()] {
				sel = append(sel, p)
			}
		}
	}

	d := db.Load()
	c := client.New(*base, k, time.Duration(*timeout)*time.Second)
	x := &probe.Ctx{C: c, Model: *model, Expect: e, ExpectCand: d.Match(e), DB: d}

	if !*jsonOut {
		fmt.Printf("modelspy %s — 正在对 %s (model=%s) 进行取证识别…\n", version, *base, *model)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	results := probe.RunAll(ctx, x, sel, func(name string, r probe.Result) {
		if !*jsonOut {
			s := "✔"
			if r.Error != "" {
				s = "✘ " + r.Error
			}
			fmt.Printf("  %-12s %s\n", name, s)
		}
	})
	rep := report.Build(*base, *model, e, version, results, d)

	if *jsonOut {
		b, _ := json.MarshalIndent(rep, "", "  ")
		fmt.Println(string(b))
	} else {
		var sb strings.Builder
		report.Print(&sb, rep)
		fmt.Print(sb.String())
	}
	if *htmlOut != "" {
		if err := report.WriteHTML(*htmlOut, rep); err != nil {
			fmt.Fprintln(os.Stderr, "HTML 写出失败:", err)
		} else if !*jsonOut {
			fmt.Println("HTML 报告:", *htmlOut)
		}
	}
	os.Exit(exitCode(rep.Verdict.Flag))
}

func exitCode(flag string) int {
	switch flag {
	case "MATCHES":
		return 0
	case "MISMATCH":
		return 1
	default:
		return 3
	}
}
