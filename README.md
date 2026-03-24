# 🦁 LionClaw

> LionClaw — A secure, self-hosted personal AI Agent platform.

[![CI](https://github.com/amszh10100-blip/lionclaw/actions/workflows/ci.yml/badge.svg)](https://github.com/amszh10100-blip/lionclaw/actions/workflows/ci.yml)
[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)](https://go.dev)
[![License](https://img.shields.io/badge/License-MIT-green)](LICENSE)

## ✨ Feature Highlights

- **Telegram Bot Integration**: Chat natively using local or cloud LLMs.
- **Smart Model Routing**: Zero-cost local Ollama models (e.g., Llama 3, Qwen) for simple tasks, cloud models (Opus) for complex reasoning.
- **Full-Text Search Memory (FTS5)**: Fast SQLite-backed memory with automatic summarization and context compression.
- **Security Scorecard**: Automated auditing of your setup and agent permissions.
- **OpenClaw Migration Tool**: 1-click migration from OpenClaw, automatically securing plaintext credentials.
- **Web UI**: Built-in authenticated dashboard for status and cost monitoring.
- **Skill SDK**: Fully compatible with OpenClaw skills, running in isolated processes.
- **User-Level Rate Limiting**: Prevent abuse and manage your daily token/cost budgets.
- **Cost Tracking**: Real-time spending alerts and cost attribution per prompt.

## 🚀 Quick Start

### Installation

```bash
git clone https://github.com/amszh10100-blip/lionclaw.git
cd lionclaw
make build
./bin/lionclaw
```

## ⚙️ Configuration

Start the interactive setup guide to configure your AI agent:

```bash
./bin/lionclaw setup
```

The wizard will help you:
1. Set up your Telegram Bot Token.
2. Configure your API keys (securely encrypted).
3. Detect your hardware and recommend the best local models.

Once configured, simply start the daemon:

```bash
./bin/lionclaw start
```

## 🏗️ Architecture Overview

LionClaw is written in pure Go with zero external C dependencies except SQLite (`mattn/go-sqlite3`). It is designed for maximum security and minimal overhead.

- **Startup time:** ~9ms
- **Memory footprint:** ~18MB
- **Search latency:** ~3ms

Key components include:
- `brain/`: LLM abstraction layer with cost routing.
- `memory/`: SQLite + FTS5 memory engine.
- `gateway/`: Core event loop and command system.
- `vault/`: AES-256-GCM encrypted credential storage.

## 🛡️ Security Scorecard

LionClaw automatically audits your setup and compares it against best practices. You can run an audit on any skill:

```bash
lionclaw skill audit <path>
```

**Scorecard Checks:**
- Ensures API keys are encrypted in Vault, never plaintext.
- Validates network bindings (defaults to `127.0.0.1`).
- Verifies macOS sandbox/process isolation for Skills.

## 🔄 Migration from OpenClaw

Migrating from OpenClaw takes less than a minute. We will automatically import your memory, skills, and configuration, while securing your plaintext credentials.

```bash
./bin/lionclaw migrate ~/.openclaw
```

## 🤝 Contributing

We welcome contributions! Please check our [Contributing Guidelines](CONTRIBUTING.md) for more details on how to run tests, format code, and submit pull requests.

## 📄 License

This project is licensed under the [MIT License](LICENSE).
