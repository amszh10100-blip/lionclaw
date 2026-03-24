#!/bin/bash
set -e
cd /Users/app/.openclaw/workspace/projects/lionclaw/src

# 1. channel/telegram
sed -i '' 's/func(m ChatMessage)/func(m channel.Message)/g' internal/channel/telegram/bot.go

# 2. updater
sed -i '' '/"context"/a \
	"github.com/lionclaw/lionclaw/internal/config"
' internal/updater/updater.go

# 3. webui
sed -i '' '/"strings"/d' internal/webui/server.go

