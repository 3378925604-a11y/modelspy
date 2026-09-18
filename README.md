# ModelSpy 🔍

**不问它"你是什么模型",直接告诉你"你到底是什么模型"。**

一个单文件命令行工具:对任意 OpenAI 兼容接口(中转站、API 代理、"Plus 代充"前端)做**行为取证**,
识别背后真正在跑的模型家族与档位,输出带置信度的判定报告。

## 它解决什么问题

中转站/套壳站的套路是吃信息差:宣称 GPT-4o,实际路由到 4o-mini、国产模型或蒸馏小模型。
现有的连通性检测工具(如 model-check)做的是"**验证**":先要求对方声明是什么,再核对自洽性——
但只要对方声明撒谎、或只在被测试时路由真模型,就查不出来。

ModelSpy 做的是"**识别**":不依赖任何声明,直接从多维行为证据推断真实模型:

| 证据维度 | 原理 | 可被中转伪装吗 |
|---|---|---|
| 分词器指纹 | 固定中英文语料的 `prompt_tokens` 比值,不同分词器差异是物理级的 | ❌ 极难 |
| 陷阱题库 | temp=0 下已知不同代际模型必对/必错的问题(strawberry、9.11 等) | ⚠️ 难 |
| 生成速度侧信道 | 流式 TPOT(每 token 耗时)分布,小模型快得多 | ⚠️ 可通过降速伪装 |
| 自我披露 | 随机措辞、多语言询问身份,防固定话术背板 | ⚠️ 容易(权重低) |
| 知识截止 | 自报训练截止日期与候选模型比对 | ⚠️ 容易(权重低) |
| Function calling | 工具调用是否结构完整(低配中转常阉割) | ⚠️ 中等 |

每路证据按独立权重投票,聚合输出**候选排名 + 置信度**,而不是黑白结论。

## 快速开始

### Windows:下载即用(零依赖)

到 [Releases](../../releases) 下载 `modelspy-windows-amd64.exe`,打开 cmd:

```bat
modelspy-windows-amd64.exe -base https://你的中转域名/v1 -model gpt-4o
```

macOS / Linux:

```bash
chmod +x modelspy-darwin-arm64 && ./modelspy-darwin-arm64 -base https://xxx/v1 -model gpt-4o
```

自行编译:

```bash
go build -o modelspy .
```

### 常用参数

```text
-base    接口地址(OpenAI 兼容,如 https://api.openai.com/v1)  必填
-model   调用时使用的模型名                                    必填
-key     API key(默认读环境变量 OPENAI_API_KEY)
-expect  对方宣称的模型(默认同 -model),用于一致性判定
-probes  只跑指定探针,如 identity,tokenizer
-html    导出 HTML 可视化报告
-json    输出机器可读 JSON
```

### 示例输出

```text
──────── 候选排名 ────────
  gpt-4.1-mini      61%  ████████████
  gpt-4o            12%  ██
  deepseek-chat     10%  █

结论: ❌ 不一致(疑似换壳)  识别结果 gpt-4.1-mini(置信度 61%)与声明 gpt-4o 不符
```

退出码:`0`=一致,`1`=不一致(疑似换壳),`3`=证据不足——可直接接入脚本做自动化监控。

## 局限(诚实声明)

1. **概率性结论**:v0.1 指纹库是社区种子数据,未经大规模校准,低置信时请多跑几轮。
2. **对抗性缺口**:中转可以"只对疑似测试的请求走真模型"。单轮检测抓不住,长期方案是常驻抽测(roadmap)。
3. **同家族盲区**:4o 与 4.1 分词器相同,主要靠题库与速度区分。
4. 本工具不抓包、不逆向、不需要对方配合,只使用公开 API 行为,不产生合规风险。

## Roadmap

- [ ] `--repeat N` 多轮随机抽测 + 统计显著性聚合(对抗"应试路由")
- [ ] `modelspy watch` 常驻守护模式,定时抽检 + 异常告警(webhook/ntfy)
- [ ] `modelspy calibrate` 半自动指纹库校准:连一个你信任的官方端点自动生成 DB 条目
- [ ] 指纹库众包上传(匿名化特征向量,不上传对话内容)
- [ ] TEE 远程认证校验(Secure Inference 类协议,硬件级终解)
- [ ] Web UI 与公开榜单

## 贡献

- 校准 `internal/db/db.json` 中的特征区间(最有价值的贡献!)
- 提供更多跨代际有区分度的陷阱题(`internal/probe/capability.go`)
- 新探针类型:实现 `probe.Probe` 接口即可

```bash
go test ./...   # 全离线,内置 mock server
```

## License

MIT
