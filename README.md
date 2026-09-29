# ModelSpy 🔍

[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Release](https://img.shields.io/github/v/release/3378925604-a11y/modelspy?color=brightgreen)](releases)
[![Downloads](https://img.shields.io/github/downloads/3378925604-a11y/modelspy/total?color=success)](releases)
[![Go build](https://img.shields.io/badge/go-single%20binary-00ADD8.svg)](#-快速上手)
[![Platform](https://img.shields.io/badge/windows%20·%20linux%20·%20macos-included-lightgrey.svg)](releases)

**付了 GPT-6 的钱，对面跑的是哪台模型？** 一个单文件命令行工具，对任意 OpenAI 兼容接口做**行为取证**——不问它"你是什么模型"，直接从 6 路行为证据推断真正在跑的模型家族与档位，输出带置信度的判定报告。

> **Detect what model is really behind an OpenAI-compatible endpoint.** ModelSpy runs a behavioral-forensics battery against any proxy, reseller or gateway — tokenizer fingerprint, trap-question bank, generation-speed side channel, tool-call integrity and more — and returns a ranked verdict with confidence. No self-declaration required, zero dependencies, one binary.

## 🚀 快速上手

Windows —— 下载即用，不需要装任何东西：

```bat
modelspy-windows-amd64.exe -base https://你的中转域名/v1 -model gpt-6-astra
```

macOS / Linux：

```bash
chmod +x modelspy-darwin-arm64 && ./modelspy-darwin-arm64 -base https://xxx/v1 -model gpt-6-astra
```

自行编译：`go build -o modelspy .`　（[Releases](releases) 下载三个平台的单文件二进制）

## 📊 你会看到什么

```text
──────── 候选排名 ────────
  gpt-5.6-luna    61%  ████████████
  gpt-5.6-sol     22%  ████
  deepseek-v4.1   10%  █

结论: ❌ 不一致(疑似换壳)  识别结果 gpt-5.6-luna(置信度 61%)与声明 gpt-6-astra 不符
```

退出码专为自动化设计，可以直接扔进 CI 或定时任务：

| 退出码 | 含义 | 能拿来干什么 |
|---|---|---|
| `0` | 与声明一致 | 放行 |
| `1` | 不一致（疑似换壳） | 告警 / 停止续费 |
| `3` | 证据不足 | 加轮次重跑 |

## 🎯 谁需要它

- **买过中转站 / "Plus 代充" / 拼车 API 的人**——花 GPT-6 的钱，实际拿到蒸馏小模型，这是最常见的坑。
- **自建网关、AI 路由、Agent 产品的运维**——上游悄悄换型号、悄悄降量化，你的线上质量会无声塌掉。
- **给团队采购 LLM API 的人**——续约谈判前拿一份带置信度的判定报告，比对方一句"我们是官方直连"有用。
- **做评测、写教程、卖课的人**——需要证明"我测的确实是这个模型"。

## 🔬 它凭什么认得出来

```
   ┌─ 分词器指纹 ── 固定语料的 prompt_tokens 比值（物理级差异）
   ├─ 陷阱题库 ──── temp=0 下各代模型必对/必错的问题
   ├─ 速度侧信道 ── 流式 TPOT 每 token 耗时分布
   ├─ 自我披露 ──── 随机措辞 + 多语言问身份，防背固定话术
   ├─ 知识截止 ──── 自报截止日期与候选模型比对
   └─ Function calling ── 工具调用结构是否被阉割
                                   │
                          每路独立权重投票
                                   ▼
                   候选排名 + 置信度 + 退出码
```

关键设计：**不依赖对方任何声明**。宣称什么不重要，行为露馅。

| 证据维度 | 原理 | 可被中转伪装吗 |
|---|---|---|
| 分词器指纹 | 固定中英文语料的 `prompt_tokens` 比值，不同分词器差异是物理级的 | ❌ 极难 |
| 陷阱题库 | temp=0 下已知不同代际模型必对/必错的问题（strawberry、9.11 等） | ⚠️ 难 |
| 生成速度侧信道 | 流式 TPOT（每 token 耗时）分布，小模型快得多 | ⚠️ 可通过降速伪装 |
| 自我披露 | 随机措辞、多语言询问身份，防固定话术背板 | ⚠️ 容易（权重低） |
| 知识截止 | 自报训练截止日期与候选模型比对 | ⚠️ 容易（权重低） |
| Function calling | 工具调用是否结构完整（低配中转常阉割） | ⚠️ 中等 |

## ⚖️ 和"验证类"工具的区别

市面上多数工具做的是**验证**：先要对方声明、或先连一个你信任的官方端点做参考指纹，再核对自洽性。问题是——**很多人正是因为没有官方 key 才来测中转站**。

| | 需要先声明/可信参考端点 | 输出 | 依赖 |
|---|---|---|---|
| 连通性检查类（model-check 等） | ✅ 需要 | 自洽 / 不自洽 | — |
| 单 token 分布指纹类 | ✅ 需要可信参考端点 | 相似度分数 | 需运行时 |
| **ModelSpy** | ❌ **不需要** | **候选排名 + 置信度 + 退出码** | **单文件，零依赖** |

## ⚙️ 参数

```text
-base    接口地址（OpenAI 兼容，如 https://api.openai.com/v1）   必填
-model   调用时使用的模型名                                      必填
-key     API key（默认读环境变量 OPENAI_API_KEY）
-expect  对方宣称的模型（默认同 -model），用于一致性判定
-probes  只跑指定探针，如 identity,tokenizer
-html    导出 HTML 可视化报告（发给采购/谈判用这个）
-json    输出机器可读 JSON
```

## ⚠️ 局限（诚实声明）

1. **概率性结论**：v0.1 指纹库是社区种子数据，未经大规模校准，低置信时请多跑几轮。
2. **对抗性缺口**：中转可以"只对疑似测试的请求走真模型"。单轮检测抓不住，长期方案是常驻抽测（见 roadmap）。
3. **同家族盲区**：GPT-6 与 GPT-5.6 分词器相同，主要靠题库与速度区分。
4. **合规**：本工具不抓包、不逆向、不需要对方配合，只使用公开 API 的行为表现。

同一问题域的公开研究与报道（ModelSpy 未实现它们的方法，只是站在同一个战场上）：

- Tomáš Bruckner, *One Token Is Enough: Fingerprinting and Verifying Large Language Models from Single-Token Output Distributions*, [arXiv:2607.10252](https://arxiv.org/abs/2607.10252)
- *Real Money, Fake Models: Deceptive Model Claims in Shadow APIs* —— 中文解读见[安全内参《花真金白银，买了个寂寞？解密大模型"影子API"的地下黑产》](https://www.secrss.com/articles/90918)

## 🗺️ Roadmap

**v0.1 已实现**：6 路探针、候选排名与置信度、HTML/JSON 报告、三平台单文件、退出码可接自动化。

**下一步**：

- [ ] `--repeat N` 多轮随机抽测 + 统计显著性聚合（对抗"应试路由"）
- [ ] `modelspy watch` 常驻守护模式，定时抽检 + 异常告警（webhook / ntfy）
- [ ] `modelspy calibrate` 半自动指纹库校准：连一个你信任的官方端点自动生成 DB 条目
- [ ] 指纹库众包上传（匿名化特征向量，不上传对话内容）
- [ ] TEE 远程认证校验（Secure Inference 类协议，硬件级终解）
- [ ] Web UI 与公开榜单

## 🤝 贡献

- 校准 `internal/db/db.json` 中的特征区间（**最有价值的贡献**）
- 提供更多跨代际有区分度的陷阱题（`internal/probe/capability.go`）
- 新探针类型：实现 `probe.Probe` 接口即可

```bash
go test ./...   # 全离线，内置 mock server
```

## 🧩 装成 Agent Skill

让 Claude Code / OpenCode / Codex / Gemini CLI 直接会用这个工具（仓库内已带四份 SKILL.md）：

```bash
# 通用：npx skills CLI，一次装到四个 agent
npx skills@latest add 3378925604-a11y/modelspy -g -y -a opencode claude-code codex gemini-cli

# Claude Code 插件市场
/plugin marketplace add 3378925604-a11y/modelspy
/plugin install modelspy@3378925604-a11y-modelspy

# 手工复制
cp -r .claude/skills/modelspy ~/.claude/skills/      # Claude Code
cp -r .opencode/skills/modelspy ~/.config/opencode/skills/  # OpenCode
cp -r .agents/skills/modelspy ~/.agents/skills/      # Codex / 通用
```

## License

MIT
