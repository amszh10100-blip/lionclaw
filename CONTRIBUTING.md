# Contributing to LionClaw 🦁

感谢你对 LionClaw 的兴趣！

## 快速开始

```bash
git clone https://github.com/amszh10100-blip/lionclaw.git
cd lionclaw
make build   # 编译
make test    # 测试
```

## 开发流程

1. Fork 仓库
2. 创建 feature 分支 (`git checkout -b feat/my-feature`)
3. 提交变更 (`git commit -m "feat: 描述"`)
4. 推送 (`git push origin feat/my-feature`)
5. 创建 Pull Request

## Commit 规范

使用 [Conventional Commits](https://www.conventionalcommits.org/)：

- `feat:` 新功能
- `fix:` 修复
- `test:` 测试
- `docs:` 文档
- `refactor:` 重构
- `perf:` 性能优化

## 测试

所有 PR 必须通过测试：

```bash
CGO_ENABLED=1 go test -tags "fts5" -race ./internal/...
```

## 安全

发现安全问题？请**不要**公开提 Issue。发邮件到 security@lionclaw.dev（占位）。

## 行为准则

参见 [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md)。
