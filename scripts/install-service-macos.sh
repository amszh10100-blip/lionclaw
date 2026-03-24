#!/bin/bash
# LionClaw macOS LaunchAgent 安装脚本
set -e

BINARY=$(which lionclaw 2>/dev/null || echo "$HOME/.lionclaw/bin/lionclaw")
PLIST="$HOME/Library/LaunchAgents/com.lionclaw.agent.plist"
LOG_DIR="$HOME/.lionclaw/logs"

mkdir -p "$LOG_DIR"
mkdir -p "$(dirname "$PLIST")"

cat > "$PLIST" << PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.lionclaw.agent</string>
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
    <string>${LOG_DIR}/lionclaw.log</string>
    <key>StandardErrorPath</key>
    <string>${LOG_DIR}/lionclaw.err</string>
    <key>WorkingDirectory</key>
    <string>${HOME}/.lionclaw</string>
</dict>
</plist>
PLIST

echo "✅ LaunchAgent 已创建: $PLIST"
echo ""
echo "管理命令:"
echo "  启动: launchctl load $PLIST"
echo "  停止: launchctl unload $PLIST"
echo "  状态: launchctl list | grep lionclaw"
echo "  日志: tail -f $LOG_DIR/lionclaw.log"
