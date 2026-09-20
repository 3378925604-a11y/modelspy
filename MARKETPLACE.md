# 各市场提交文案（modelspy skill）

> 这份文件只用于往第三方 skill 市场投稿时复制粘贴，可随时删除，不影响技能本体。
> 内容按 `skills/modelspy/SKILL.md`、README 与 `main.go` 实际实现核对过。

## 基本信息

| 字段 | 值 |
|---|---|
| Skill name | `modelspy` |
| 仓库 | https://github.com/3378925604-a11y/modelspy |
| 技能路径 | `skills/modelspy/SKILL.md`（另有 `.claude/`、`.agents/`、`.opencode/` 三份同内容副本） |
| Release | https://github.com/3378925604-a11y/modelspy/releases/tag/v0.1.0 |
| License | MIT |
| 语言 / 依赖 | Go 单文件二进制，零运行时依赖（Windows / Linux amd64 / macOS arm64） |
| 分类 | Developer Tools / AI & LLM / Testing & QA / Security |
| 标签 | `model-detection`, `openai-compatible`, `llm`, `api-proxy`, `behavioral-forensics`, `qa`, `chinese` |

## 一句话简介（列表页用）

- 中文：识别中转站/套壳 API 背后真正在跑的大模型，六路行为探针投票出带置信度的结论。
- English: Identify which LLM actually serves an OpenAI-compatible endpoint — six behavioral probes vote out a confidence-ranked verdict.

## 长描述（详情页 / 投稿表单用）

宣称 GPT-4o、实际路由到 4o-mini 或蒸馏小模型，是中转站和"代充"前端最常见的吃信息差手法。现有连通性检测做的是"验证"——先要对方声明是什么，再核对自洽性，对方撒谎或只在被测试时走真模型就查不出来。

ModelSpy 做的是"识别"：不依赖任何声明，直接从行为证据反推真实模型家族与档位。六路探针各自独立打分、按权重聚合，输出候选排名 + 置信度，而不是黑白结论：

- 分词器指纹：固定中英文语料的 `prompt_tokens` 比值，跨家族差异是物理级的，几乎无法伪装
- 陷阱题库：temp=0 下不同代际必对/必错的已知问题（strawberry、9.11 等）
- 生成速度侧信道：流式 TPOT（每 token 耗时）分布，小模型明显更快
- 自我披露：随机措辞、多语言询问身份，防固定话术背板（权重低）
- 知识截止：自报训练截止日期与候选模型比对（权重低）
- Function calling：工具调用结构是否完整（低配中转常阉割）

用法：`modelspy -base <接口地址> -model <模型名> [-key <KEY>] [-expect <宣称模型>] [-html report.html]`。退出码 `0`=与声明一致、`1`=疑似换壳、`3`=证据不足，可直接接进 CI 或脚本做长期监控。

诚实声明：同家族（4o 与 4.1 分词器相同）主要靠题库与速度区分；v0.1 指纹库是社区种子数据，低置信时建议多跑几轮；不抓包、不逆向、不需要对方配合，只用公开 API 行为。

## 触发场景（英文市场用）

Detect which model an API endpoint really runs · verify a reseller isn't downgrading GPT-4o to a cheaper model · "is this really Claude/GPT" · audit OpenAI-compatible proxy behavior · spot model substitution in API 中转站/代充站.

## 安装方式（写进市场详情）

```bash
# skills.sh / 通用 CLI（仓库已被发现，实测找到 1 个技能：modelspy）
npx skills add 3378925604-a11y/modelspy

# Claude Code
cp -r .claude/skills/modelspy ~/.claude/skills/

# OpenCode
cp -r .opencode/skills/modelspy ~/.config/opencode/skills/

# Codex / Gemini CLI / 通用 Agent
cp -r .agents/skills/modelspy ~/.agents/skills/
```

首次运行时技能会自动从 GitHub Release 拉对应平台的二进制到 `scripts/` 目录，无需手动装 Go 或编译。

## 投稿状态

| 平台 | 方式 | 状态 |
|---|---|---|
| GitHub | `git push` | ✅ 已完成：`skills/` 规范路径 + 中英双语 description 已上线 |
| skills.sh (Vercel) | 自动索引 | ⏳ 索引来自 CLI 匿名安装遥测，没有投稿入口；`npx skills add` 已实测可发现并读取到最新描述，等安装量产生后会出现在搜索里 |
| agentskill.sh | 连 GitHub 认领 | 🔑 需要你在浏览器用 GitHub 账号登录授权，无法代跑 |
| LobeHub Marketplace | GitHub 抓取 / 表单 | 🔑 需要 LobeHub 账号提交，无法代跑 |
| SkillsMP | REST API / MCP | 🔑 提交要 GitHub token（本机未装 `gh`），暂未被爬取（站内查不到） |
| QASkills.sh | `npx @qaskills/cli publish` | 🔑 第三方发布 CLI，需要其账号 token；包真实存在（`@qaskills/cli@0.4.1`，作者 PramodDutta / TheTestingAcademy，偏 QA 分类） |
| Skillerr | `npx @skillerr/add publish` | ⚠️ 需先打 `.skill` 包；`@skillerr/add@0.6.0` 的描述自称"sealed install"闭源客户端，不是开放协议 CLI，交出 GitHub 凭据前请自行判断 |

## 一个本机环境坑

这台机器的全局 git 配了 `http.proxy=http://127.0.0.1:65532`，而该端口现在没在监听，所以 `git push`、`git clone`、`npx skills add`（内部会 clone）全部报 `Failed to connect to github.com port 443 via 127.0.0.1`。本次提交是在命令级临时绕过（`-c http.proxy=`），没有改你的全局配置。长期处理二选一：把代理工具开着，或者 `git config --global --unset http.proxy` / `--unset https.proxy`。
