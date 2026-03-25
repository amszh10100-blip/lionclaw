<div align="center">

# 🦁 LionClaw

**A secure, self-hosted personal AI Agent platform.**

[![CI](https://github.com/amszh10100-blip/lionclaw/actions/workflows/ci.yml/badge.svg)](https://github.com/amszh10100-blip/lionclaw/actions)
[![Go](https://img.shields.io/badge/Go-1.25-blue.svg)](https://go.dev)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

*Your AI, your hardware, your data. Zero cloud dependency.*

</div>

---

## ✨ Why LionClaw?

| Feature | LionClaw | Others |
|---------|----------|--------|
| 🔐 Encrypted credential vault | ✅ AES-256-GCM + Keychain | ❌ Plaintext |
| 🏠 Local-first (Ollama) | ✅ Zero API cost | ❌ Cloud required |
| 🎭 5 Built-in Scenarios | ✅ Switch with `/scenario` | ❌ Manual prompts |
| 🛡️ Skill Sandbox | ✅ Timeout + env isolation | ❌ Unrestricted |
| 📊 Audit Logging | ✅ Full operation history | ❌ None |
| 💰 Cost Tracking | ✅ Per-model breakdown | ❌ Limited |
| 🔄 One-click Migration | ✅ From OpenClaw | — |

## 🚀 Quick Start

### Docker (Recommended)

```bash
git clone https://github.com/amszh10100-blip/lionclaw.git
cd lionclaw
docker compose up -d
```

### From Source

```bash
git clone https://github.com/amszh10100-blip/lionclaw.git
cd lionclaw
make build
./bin/lionclaw setup    # Interactive setup wizard
./bin/lionclaw start    # Start the gateway
```

### Homebrew (coming soon)

```bash
brew tap amszh10100-blip/tap
brew install lionclaw
```

## 🎭 Built-in Scenarios

| Scenario | Command | Description |
|----------|---------|-------------|
| 🤖 Assistant | `/scenario assistant` | General-purpose AI helper |
| 🌐 Translator | `/scenario translator` | Auto Chinese↔English |
| 💻 Coder | `/scenario coder` | Code, debug, review |
| ✍️ Writer | `/scenario writer` | Copy, emails, reports |
| 📊 Daily Report | `/scenario daily` | Work report generator |

## 📋 Commands

| Command | Description |
|---------|-------------|
| `/start` | Welcome + register |
| `/status` | System status |
| `/cost` | Usage & cost breakdown |
| `/model` | Switch AI model |
| `/scenario` | Switch scenario |
| `/search` | Full-text search history |
| `/share` | Generate shareable status card |
| `/audit` | View operation audit log |
| `/export` | Export chat history |
| `/clear` | Clear session |

## 🏗️ Architecture

```
lionclaw/
├── cmd/lionclaw/     # CLI entry point
├── internal/
│   ├── audit/        # 📋 Operation audit logging
│   ├── brain/        # 🧠 AI model router (Ollama/OpenAI/Anthropic)
│   ├── channel/      # 💬 Telegram bot integration
│   ├── config/       # ⚙️ YAML configuration
│   ├── gateway/      # 🔀 Message gateway + commands
│   ├── memory/       # 🗄️ SQLite + FTS5 memory engine
│   ├── migrate/      # 🔄 OpenClaw migration tool
│   ├── protocol/     # 🔌 MCP protocol client
│   ├── scheduler/    # ⏰ Scheduled tasks
│   ├── scorecard/    # 🛡️ Security scorecard
│   ├── skill/        # 📦 Skill SDK + sandbox
│   ├── updater/      # 🔄 Self-update mechanism
│   ├── vault/        # 🔐 Encrypted credential store
│   └── webui/        # 🌐 Web dashboard
├── Dockerfile
├── docker-compose.yml
└── Makefile
```

## 🔐 Security

- **Vault**: AES-256-GCM encryption, macOS Keychain / Linux env-derived keys
- **Skill Sandbox**: 30s timeout, env whitelist, isolated working directory
- **Web UI**: Mandatory authentication (configurable via env)
- **Audit Trail**: Every AI operation logged with timestamp, model, tokens, cost

## 🛠️ Configuration

```bash
# Environment variables
OLLAMA_HOST=http://127.0.0.1:11434    # Ollama endpoint
LIONCLAW_WEBUI_USER=admin              # Web UI username
LIONCLAW_WEBUI_PASS=your-password      # Web UI password
LIONCLAW_MASTER_KEY=your-key           # Linux vault encryption key
```

## 🤝 Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## 📄 License

MIT — see [LICENSE](LICENSE)
