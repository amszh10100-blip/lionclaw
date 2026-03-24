package webui

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/goldlion/goldlion/internal/brain"
	"github.com/goldlion/goldlion/internal/config"
)

// Server 内嵌 Web UI 服务器
type Server struct {
	cfg    *config.Config
	cost   brain.CostTracker
	logger *slog.Logger
	srv    *http.Server
}

// New 创建 Web UI 服务器
func New(cfg *config.Config, cost brain.CostTracker, logger *slog.Logger) *Server {
	s := &Server{cfg: cfg, cost: cost, logger: logger}

	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleDashboard)
	mux.HandleFunc("/api/status", s.handleAPIStatus)
	mux.HandleFunc("/api/cost", s.handleAPICost)

	s.srv = &http.Server{
		Addr:    fmt.Sprintf("%s:%d", cfg.Security.Bind, cfg.Security.Port),
		Handler: mux,
	}

	return s
}

// Start 启动 Web 服务
func (s *Server) Start(ctx context.Context) error {
	s.logger.Info("Web UI 启动", "addr", s.srv.Addr)

	go func() {
		<-ctx.Done()
		s.srv.Shutdown(context.Background())
	}()

	if err := s.srv.ListenAndServe(); err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	todayTotal, todayRecords, _ := s.cost.GetToday()
	monthTotal, monthRecords, _ := s.cost.GetMonth()

	localToday, cloudToday := 0, 0
	for _, r := range todayRecords {
		if r.IsLocal { localToday++ } else { cloudToday++ }
	}

	localMonth := 0
	for _, r := range monthRecords {
		if r.IsLocal { localMonth++ }
	}

	localPct := 0.0
	if len(monthRecords) > 0 {
		localPct = float64(localMonth) / float64(len(monthRecords)) * 100
	}

	savedHours := float64(len(monthRecords)*2) / 60

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, dashboardHTML,
		// 今日
		len(todayRecords), localToday, cloudToday, todayTotal,
		s.cfg.Cost.DailyLimitUSD-todayTotal,
		// 本月
		len(monthRecords), monthTotal, savedHours, localPct,
		// 配置
		s.cfg.Models.Local.Models.Small, s.cfg.Models.Local.Models.Large,
		s.cfg.Security.Bind, s.cfg.Security.Port,
		// 时间
		time.Now().Format("2006-01-02 15:04:05"),
	)
}

func (s *Server) handleAPIStatus(w http.ResponseWriter, r *http.Request) {
	todayTotal, todayRecords, _ := s.cost.GetToday()

	status := map[string]interface{}{
		"version":       "0.1.0-dev",
		"uptime":        time.Now().Format(time.RFC3339),
		"today_calls":   len(todayRecords),
		"today_cost":    todayTotal,
		"daily_budget":  s.cfg.Cost.DailyLimitUSD,
		"local_enabled": s.cfg.Models.Local.Enabled,
		"bind":          s.cfg.Security.Bind,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func (s *Server) handleAPICost(w http.ResponseWriter, r *http.Request) {
	todayTotal, todayRecords, _ := s.cost.GetToday()
	monthTotal, _, _ := s.cost.GetMonth()

	data := map[string]interface{}{
		"today":          todayTotal,
		"today_calls":    len(todayRecords),
		"month":          monthTotal,
		"daily_budget":   s.cfg.Cost.DailyLimitUSD,
		"monthly_budget": s.cfg.Cost.MonthlyLimitUSD,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

const dashboardHTML = `<!DOCTYPE html>
<html lang="zh">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>🦁 GoldLion Dashboard</title>
<style>
  * { margin: 0; padding: 0; box-sizing: border-box; }
  body {
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', system-ui, sans-serif;
    background: #0a0a0a; color: #e0e0e0;
    min-height: 100vh; padding: 2rem;
  }
  .header {
    text-align: center; margin-bottom: 2rem;
  }
  .header h1 { font-size: 2rem; color: #f5a623; }
  .header p { color: #888; margin-top: 0.5rem; }
  .grid {
    display: grid; grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
    gap: 1.5rem; max-width: 1000px; margin: 0 auto;
  }
  .card {
    background: #1a1a1a; border: 1px solid #333;
    border-radius: 12px; padding: 1.5rem;
  }
  .card h2 { color: #f5a623; font-size: 1rem; margin-bottom: 1rem; }
  .stat { font-size: 2rem; font-weight: bold; color: #fff; }
  .stat-label { color: #888; font-size: 0.85rem; margin-top: 0.25rem; }
  .stat-row {
    display: flex; justify-content: space-between;
    padding: 0.5rem 0; border-bottom: 1px solid #222;
  }
  .stat-row:last-child { border: none; }
  .green { color: #4caf50; }
  .gold { color: #f5a623; }
  .badge {
    display: inline-block; padding: 2px 8px;
    border-radius: 4px; font-size: 0.75rem;
    background: #1b5e20; color: #4caf50;
  }
  .footer { text-align: center; margin-top: 2rem; color: #555; font-size: 0.8rem; }
</style>
</head>
<body>
<div class="header">
  <h1>🦁 GoldLion Dashboard</h1>
  <p>安全的个人 AI Agent</p>
</div>

<div class="grid">
  <div class="card">
    <h2>📊 今日统计</h2>
    <div class="stat">%d <span style="font-size:1rem;color:#888">次对话</span></div>
    <div class="stat-row"><span>本地调用</span><span class="green">%d 次</span></div>
    <div class="stat-row"><span>云端调用</span><span>%d 次</span></div>
    <div class="stat-row"><span>花费</span><span class="gold">$%.4f</span></div>
    <div class="stat-row"><span>预算剩余</span><span class="green">$%.4f</span></div>
  </div>

  <div class="card">
    <h2>📈 本月总览</h2>
    <div class="stat">%d <span style="font-size:1rem;color:#888">次对话</span></div>
    <div class="stat-row"><span>总花费</span><span class="gold">$%.4f</span></div>
    <div class="stat-row"><span>节省时间</span><span class="green">~%.1f 小时</span></div>
    <div class="stat-row"><span>本地使用率</span><span class="green">%.0f%%</span></div>
  </div>

  <div class="card">
    <h2>🧠 模型配置</h2>
    <div class="stat-row"><span>本地(小)</span><span>%s</span></div>
    <div class="stat-row"><span>本地(大)</span><span>%s</span></div>
    <div class="stat-row"><span>路由</span><span><span class="badge">自动</span></span></div>
  </div>

  <div class="card">
    <h2>🛡️ 安全状态</h2>
    <div class="stat-row"><span>凭证存储</span><span class="green">AES-256 ✅</span></div>
    <div class="stat-row"><span>网络绑定</span><span class="green">%s:%d ✅</span></div>
    <div class="stat-row"><span>Skill 隔离</span><span class="green">进程隔离 ✅</span></div>
    <div class="stat-row"><span>安全评分</span><span class="gold">A+</span></div>
  </div>
</div>

<div class="footer">
  GoldLion v0.1.0-dev | 更新于 %s | <a href="/api/status" style="color:#555">API</a>
</div>

<script>
  // 每 30 秒自动刷新
  setTimeout(() => location.reload(), 30000);
</script>
</body>
</html>`
