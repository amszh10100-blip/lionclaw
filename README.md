# 🦁 GoldLion

> 安全的个人 AI Agent — OpenClaw 的安全替代品，5 分钟上手。

## 核心特性

- 🛡️ **安全第一** — 凭证 AES-256 加密，Gateway 默认 localhost，Skill 进程隔离
- ⚡ **本地优先** — Ollama 智能路由，隐私数据永不上云
- 💰 **成本透明** — 每次调用显示模型+成本，支持日/月预算上限
- 📦 **单二进制** — 一条命令安装，5 分钟上手

## 快速开始

```bash
# 安装（TODO: P0 W5 提供安装脚本）
go build -o goldlion ./cmd/goldlion

# 配置
./goldlion setup

# 启动
./goldlion start
```

## 开发

```bash
make build    # 构建
make test     # 测试
make run      # 构建并运行
make fmt      # 格式化代码
```

## 状态

**P0 开发中** — Week 1-2 核心骨架

## License

MIT
