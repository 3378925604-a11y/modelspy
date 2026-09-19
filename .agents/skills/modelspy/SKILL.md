---
name: modelspy
description: '检测中转站/套壳站背后真实模型。用户提到"测模型降智"、"查中转站跑的什么模型"、"这个API是真4o还是套壳"、"模型识别"、"model spy"时使用。'
disable-model-invocation: false
user-invocable: true
---

# ModelSpy - 模型行为取证

对任意 OpenAI 兼容接口做行为取证，识别背后真正在跑的模型家族与档位，输出带置信度的判定报告。

## 触发场景

- 用户怀疑中转站/代充站把高配模型换成低配
- 用户想验证某个 API 端点跑的是不是宣称的模型
- 用户说"测一下这个模型降没降智"、"帮我查查这个中转"

## 用法

### 第 0 步 · 确保二进制可用

优先使用本技能目录 `scripts/` 下已有的 modelspy 程序（Windows 为 `modelspy.exe`，Linux/macOS 为 `modelspy`）。若不存在，从官方 Release 下载到 `scripts/` 目录并按平台改名：

- Windows amd64：`https://github.com/3378925604-a11y/modelspy/releases/download/v0.1.0/modelspy-windows-amd64.exe` → 保存为 `scripts/modelspy.exe`
- Linux amd64：`https://github.com/3378925604-a11y/modelspy/releases/download/v0.1.0/modelspy-linux-amd64` → 保存为 `scripts/modelspy`，并 `chmod +x`
- macOS arm64：`https://github.com/3378925604-a11y/modelspy/releases/download/v0.1.0/modelspy-darwin-arm64` → 保存为 `scripts/modelspy`，并 `chmod +x`

下载后先运行 `-h` 确认可执行，再进入下一步。

### 第 1 步 · 执行检测

```powershell
& "SKILL_DIR\scripts\modelspy.exe" -base <接口地址> -model <模型名> [-key <API_KEY>] [-expect <宣称模型>] [-json] [-html <报告路径>]
```

### 参数说明

| 参数 | 必填 | 说明 |
|------|------|------|
| `-base` | ✅ | OpenAI 兼容接口地址，如 `https://api.openai.com/v1` |
| `-model` | ✅ | 调用时使用的模型名，如 `gpt-4o` |
| `-key` | ❌ | API key（默认读环境变量 `OPENAI_API_KEY`） |
| `-expect` | ❌ | 对方宣称的模型（默认同 `-model`） |
| `-probes` | ❌ | 只跑指定探针，逗号分隔：`identity,tokenizer,capability,cutoff,speed,toolcall` |
| `-json` | ❌ | 输出机器可读 JSON |
| `-html` | ❌ | 导出 HTML 可视化报告 |
| `-timeout` | ❌ | 单次请求超时秒数（默认 90） |

### 探针说明

| 探针 | 说明 |
|------|------|
| `identity` | 随机措辞询问模型身份，防固定话术背板 |
| `tokenizer` | 固定中英文语料的 prompt_tokens 比值，物理级指纹 |
| `capability` | 陷阱题库（strawberry、9.11 等），temp=0 下区分代际 |
| `cutoff` | 自报训练截止日期与候选模型比对 |
| `speed` | 流式 TPOT（每 token 耗时）分布，小模型快得多 |
| `toolcall` | Function calling 结构完整性检查 |

## 工作流程

1. 确认用户提供了接口地址和模型名
2. 如用户未提供 API key，提示设置环境变量或传 `-key`
3. 执行 modelspy 并等待结果（默认 10 分钟超时）
4. 解读报告：关注置信度排名和 `MATCHES`/`MISMATCH` 判定
5. 如结果置信度低，建议 `--repeat` 多跑几轮或只跑特定探针

## 输出解读

- `MATCHES`（退出码 0）：识别结果与声明一致
- `MISMATCH`（退出码 1）：疑似换壳
- 退出码 3：证据不足

## 局限

- 同家族盲区：4o 与 4.1 分词器相同，主要靠题库与速度区分
- 概率性结论：低置信时多跑几轮
- 对抗性缺口：中转可以"只对疑似测试的请求走真模型"
