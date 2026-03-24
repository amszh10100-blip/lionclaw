#!/bin/bash
# GoldLion macOS LaunchAgent 安装脚本
set -e

BINARY=$(which goldlion 2>/dev/null || echo "$HOME/.goldlion/bin/goldlion")
PLIST="$HOME/Library/LaunchAgents/com.goldlion.agent.plist"
LOG_DIR="$HOME/.goldlion/logs"

mkdir -p "$LOG_DIR"
mkdir -p "$(dirname "$PLIST")"

cat > "$PLIST" << PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.goldlion.agent</string>
    <key>ProgramArguments</key>
    <array>
        <string>${BINARY}</string>
        <string>start</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>${LOG_DIR}/goldlion.log</string>
    <key>StandardErrorPath</key>
    <string>${LOG_DIR}/goldlion.err</string>
    <key>WorkingDirectory</key>
    <string>${HOME}/.goldlion</string>
</dict>
</plist>
PLIST

echo "✅ LaunchAgent 已创建: $PLIST"
echo ""
echo "管理命令:"
echo "  启动: launchctl load $PLIST"
echo "  停止: launchctl unload $PLIST"
echo "  状态: launchctl list | grep goldlion"
echo "  日志: tail -f $LOG_DIR/goldlion.log"
