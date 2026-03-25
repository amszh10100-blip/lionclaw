# Changelog

## [2.0.0] - 2026-03-25

### Added
- 🎭 5 built-in scenario packs (assistant/translator/coder/writer/daily)
- 🌐 Real-time web dashboard with dark theme
- 📊 /share command for shareable status cards
- 🛡️ Skill security rating (A-F grade)
- 📋 Audit log system with /audit command and CSV export
- 🔒 Skill sandbox with timeout and env isolation
- 🐳 Docker support (Dockerfile + docker-compose.yml)
- 🔍 Ollama auto-detection in setup wizard
- 🧪 Tests for protocol, scorecard, updater packages

### Fixed
- Web UI authentication bypass via reverse proxy
- Linux vault master key stored in plaintext
- Hardcoded default password in Web UI
- Silent error swallowing in time.Parse calls
- Telegram bot goroutine explosion (added semaphore)
- Makefile missing -tags "fts5" in test target
- Nil pointer in store_test.go

### Changed
- Version injection via ldflags (removed hardcoded version)
- Ollama endpoint configurable via OLLAMA_HOST env
- Web UI credentials via environment variables

### Removed
- Leftover debug scripts

## [1.0.0] - 2026-03-24

### Added
- Initial release
- Telegram bot with native HTTP polling
- Ollama/OpenAI/Anthropic model routing
- SQLite + FTS5 memory engine
- Encrypted vault (macOS Keychain + Linux)
- Skill SDK with audit
- OpenClaw migration tool
- Security scorecard
- Cost tracking with budget alerts
- Web UI with basic auth
- GitHub Actions CI
