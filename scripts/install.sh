#!/bin/bash
# GoldLion 一键安装脚本
# curl -fsSL goldlion.ai/install | sh
set -e

VERSION="0.1.0-dev"
INSTALL_DIR="$HOME/.goldlion/bin"
CONFIG_DIR="$HOME/.goldlion"

# 颜色
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo ""
echo "🦁 GoldLion v${VERSION} 安装程序"
echo "   安全的个人 AI Agent"
echo ""

# 检测系统
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
    x86_64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *) echo -e "${RED}❌ 不支持的架构: $ARCH${NC}"; exit 1 ;;
esac

echo "📦 系统: ${OS}/${ARCH}"

# 创建目录
mkdir -p "$INSTALL_DIR"
mkdir -p "$CONFIG_DIR/data"
mkdir -p "$CONFIG_DIR/memory"
mkdir -p "$CONFIG_DIR/skills"

# 下载二进制（TODO: 替换为实际下载地址）
BINARY_URL="https://github.com/goldlion/goldlion/releases/download/v${VERSION}/goldlion-${OS}-${ARCH}"

echo "⬇️  下载 GoldLion..."
if command -v curl &>/dev/null; then
    # 开发阶段：如果本地有编译好的二进制，直接复制
    if [ -f "./bin/goldlion" ]; then
        cp ./bin/goldlion "$INSTALL_DIR/goldlion"
        echo "   (使用本地编译版本)"
    else
        echo -e "${YELLOW}⚠️  发布版本尚未上线，请先手动编译:${NC}"
        echo "   cd src && make build"
        echo "   cp bin/goldlion $INSTALL_DIR/"
        exit 1
    fi
else
    echo -e "${RED}❌ 需要 curl${NC}"
    exit 1
fi

chmod +x "$INSTALL_DIR/goldlion"

# 添加到 PATH
SHELL_RC=""
if [ -f "$HOME/.zshrc" ]; then
    SHELL_RC="$HOME/.zshrc"
elif [ -f "$HOME/.bashrc" ]; then
    SHELL_RC="$HOME/.bashrc"
fi

if [ -n "$SHELL_RC" ]; then
    if ! grep -q "goldlion/bin" "$SHELL_RC" 2>/dev/null; then
        echo "" >> "$SHELL_RC"
        echo '# GoldLion' >> "$SHELL_RC"
        echo "export PATH=\"$INSTALL_DIR:\$PATH\"" >> "$SHELL_RC"
        echo "   ✅ 已添加到 PATH ($SHELL_RC)"
    fi
fi

# 检测 Ollama
echo ""
echo "🔍 检测环境..."

if command -v ollama &>/dev/null; then
    echo "   ✅ Ollama 已安装"
    # 检测可用模型
    MODELS=$(ollama list 2>/dev/null | tail -n +2 | awk '{print $1}' | head -5)
    if [ -n "$MODELS" ]; then
        echo "   📦 可用模型:"
        echo "$MODELS" | while read m; do echo "      - $m"; done
    fi
else
    echo -e "   ${YELLOW}⚠️  Ollama 未安装（本地模型需要）${NC}"
    echo "   安装: https://ollama.ai/download"
fi

echo ""
echo -e "${GREEN}✅ GoldLion 安装完成！${NC}"
echo ""
echo "下一步:"
echo "  1. 运行配置引导:  goldlion setup"
echo "  2. 启动 Agent:     goldlion start"
echo ""
echo "🔒 安全提醒:"
echo "  - 所有凭证加密存储 (AES-256)"
echo "  - Gateway 默认仅本地访问"
echo "  - 数据目录: $CONFIG_DIR"
