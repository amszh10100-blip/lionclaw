# 🦁 LionClaw

> **安全的个人 AI Agent** — OpenClaw 的安全替代品，5 分钟上手。

[![Go](https://img.shields.io/badge/Go-1.23+-00ADD8?logo=go)](https://go.dev)
[![License](https://img.shields.io/badge/License-MIT-green)](LICENSE)
[![Security](https://img.shields.io/badge/Security-A+-gold)](#安全)

## 为什么选择 LionClaw？

| | OpenClaw 🦞 | LionClaw 🦁 |
|---|---|---|
| **安全** | 6 CVE，明文凭证 | AES-256 加密 + 零信任 |
| **安装** | "三天配不好" | 一条命令，5 分钟上手 |
| **成本** | 一句 hi 花 $11 | 本地优先，自动路由 |
| **更新** | 更新后频繁崩溃 | 原子更新 + 自动回滚 |
| **记忆** | 纯 Markdown | FTS5 全文搜索 |

## 快速开始

### 安装

```bash
# 从源码编译（需要 Go 1.23+）
git clone https://github.com/lionclaw/lionclaw.git
cd lionclaw
make build
```

### 配置

```bash
# 交互式引导
./bin/lionclaw setup
```

按提示输入：
1. Telegram Bot Token（从 @BotFather 获取）
2. API Key（可选，本地模型零成本）
3. 自动检测硬件，推荐并下载最佳本地模型

### 启动

```bash
./bin/lionclaw start
```

打开 Telegram，给你的 Bot 发消息，开始对话！

## 功能

### 🛡️ 安全第一

- **凭证加密**：所有 API Key 使用 AES-256-GCM 加密，主密钥存 OS Keychain
- **零信任网络**：Gateway 默认仅 `127.0.0.1`，外网不可访问
- **Skill 隔离**：每个 Skill 在独立进程中运行（macOS sandbox-exec）
- **权限声明**：Skill 必须声明所需权限，安装时可审查

### 🧠 智能模型路由

```
低复杂度 (你好/ok)     → 本地 8B 模型  ($0)
中复杂度 (怎么做/帮我)  → 本地 32B 模型 ($0)
高复杂度 (分析/设计)    → 云端 Opus     ($$)
隐私内容 (密码/银行)    → 强制本地      ($0)
```

### 💰 成本透明

- 每次回复显示：`⚡ 模型名 | $成本`
- 日/月预算上限 + 80% 预警
- `/cost` 实时查看花费
- Web 仪表盘：`http://127.0.0.1:18790`

### 🔍 记忆搜索

- SQLite + FTS5 全文搜索
- 跨会话记忆连续
- 上下文自动压缩（超 40 条自动摘要）
- `/search 关键词` 搜索历史对话
- `/export` 导出为 Markdown

### ⏰ 场景包

预配置的自动化场景：

| 场景 | 描述 | 命令 |
|------|------|------|
| ☀️ 晨间简报 | 每天 9:00 推送 | `/enable morning_brief` |
| 🔧 GitHub 巡逻 | 每 2 小时检查 | `/enable github_patrol` |
| 📅 会议助手 | 每小时提醒 | `/enable meeting_prep` |
| 📊 周价值报告 | 每天 9:00 | `/enable weekly_report` |

### 🔄 从 OpenClaw 迁移

```bash
./bin/lionclaw migrate ~/.openclaw
```

一键迁移：记忆 + Skills + 配置 + 自动修复明文凭证。

## Telegram 命令

| 命令 | 功能 |
|------|------|
| `/help` | 所有命令 |
| `/status` | 系统状态 |
| `/cost` | 成本统计 |
| `/stats` | 详细统计 + 节省时间 |
| `/model` | 模型配置 |
| `/search <词>` | 搜索记忆 |
| `/export` | 导出记忆 |
| `/clear` | 清除会话 |
| `/scenario` | 场景包列表 |
| `/enable <名>` | 启用场景 |
| `/disable <名>` | 停用场景 |
| `/route <文本>` | 测试路由决策 |

## CLI 命令

```bash
lionclaw start       # 启动 Gateway
lionclaw setup       # 交互式配置
lionclaw status      # 查看状态
lionclaw cost        # 成本统计
lionclaw skill create <name>   # 创建 Skill
lionclaw skill list            # 列出 Skills
lionclaw skill audit <path>    # 安全审计
lionclaw vault set <key> <val> # 存储凭证
lionclaw vault list            # 列出凭证
lionclaw migrate <dir>         # 从 OpenClaw 迁移
```

## 开发

```bash
make build    # 编译（含 FTS5）
make test     # 运行测试
make fmt      # 格式化代码
make cover    # 测试覆盖率
```

### 项目结构

```
internal/
├── brain/        # LLM 抽象 (Ollama/Anthropic/OpenAI/路由/成本)
├── channel/      # 渠道抽象 + Telegram
├── config/       # 配置管理
├── gateway/      # 核心网关 + 命令系统
├── memory/       # SQLite + FTS5 + 压缩
├── migrate/      # OpenClaw 迁移
├── protocol/     # MCP 客户端
├── scheduler/    # Cron 调度
├── scorecard/    # 安全评分卡
├── skill/        # Skill 管理 + SDK
├── updater/      # 原子更新
├── vault/        # 加密凭证
└── webui/        # Web 仪表盘
```

## 安全

- AES-256-GCM 凭证加密
- macOS Keychain / Linux secret-service 主密钥
- 默认 localhost 绑定
- Skill 进程隔离 (sandbox-exec)
- Skill 权限声明系统
- 安全评分卡：`lionclaw migrate` 自动对比

## License

MIT
