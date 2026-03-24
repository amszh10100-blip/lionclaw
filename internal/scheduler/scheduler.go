package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Job 定时任务
type Job struct {
	Name    string    `json:"name"`
	Cron    string    `json:"cron"`    // 简化 cron: "HH:MM" 或 "*/N" (每 N 分钟)
	Prompt  string    `json:"prompt"`  // 要发给 LLM 的提示
	ChatID  string    `json:"chat_id"` // 目标聊天
	Enabled bool      `json:"enabled"`
	LastRun time.Time `json:"last_run"`
}

// Handler 任务执行回调
type Handler func(ctx context.Context, job Job) error

// Scheduler 简单的定时调度器
type Scheduler struct {
	jobs    []Job
	handler Handler
	logger  *slog.Logger
	mu      sync.RWMutex
	stopCh  chan struct{}
}

// New 创建调度器
func New(logger *slog.Logger) *Scheduler {
	return &Scheduler{
		logger: logger,
		stopCh: make(chan struct{}),
	}
}

// AddJob 添加任务
func (s *Scheduler) AddJob(job Job) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.jobs = append(s.jobs, job)
	s.logger.Info("添加定时任务", "name", job.Name, "cron", job.Cron)
}

// SetHandler 设置任务执行回调
func (s *Scheduler) SetHandler(h Handler) {
	s.handler = h
}

// Start 启动调度循环
func (s *Scheduler) Start(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	s.logger.Info("调度器已启动", "jobs", len(s.jobs))

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case now := <-ticker.C:
			s.tick(ctx, now)
		}
	}
}

// Stop 停止调度器
func (s *Scheduler) Stop() {
	close(s.stopCh)
}

func (s *Scheduler) tick(ctx context.Context, now time.Time) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for i := range s.jobs {
		job := &s.jobs[i]
		if !job.Enabled {
			continue
		}

		if s.shouldRun(job, now) {
			s.logger.Info("触发定时任务", "name", job.Name)
			job.LastRun = now

			if s.handler != nil {
				go func(j Job) {
					if err := s.handler(ctx, j); err != nil {
						s.logger.Error("任务执行失败", "name", j.Name, "error", err)
					}
				}(*job)
			}
		}
	}
}

// shouldRun 判断是否该执行
func (s *Scheduler) shouldRun(job *Job, now time.Time) bool {
	cron := strings.TrimSpace(job.Cron)

	// 格式1: "HH:MM" — 每天定时
	if len(cron) == 5 && cron[2] == ':' {
		parts := strings.Split(cron, ":")
		hour, _ := strconv.Atoi(parts[0])
		minute, _ := strconv.Atoi(parts[1])

		if now.Hour() == hour && now.Minute() == minute {
			// 确保今天没跑过
			if job.LastRun.Day() != now.Day() || job.LastRun.Month() != now.Month() {
				return true
			}
		}
		return false
	}

	// 格式2: "*/N" — 每 N 分钟
	if strings.HasPrefix(cron, "*/") {
		interval, err := strconv.Atoi(cron[2:])
		if err != nil || interval <= 0 {
			return false
		}
		if now.Minute()%interval == 0 {
			elapsed := now.Sub(job.LastRun)
			if elapsed >= time.Duration(interval-1)*time.Minute {
				return true
			}
		}
		return false
	}

	// 格式3: 标准 cron (简化版: "M H * * *")
	if parts := strings.Fields(cron); len(parts) >= 2 {
		minute, err1 := strconv.Atoi(parts[0])
		hour, err2 := strconv.Atoi(parts[1])
		if err1 == nil && err2 == nil {
			if now.Hour() == hour && now.Minute() == minute {
				if job.LastRun.Day() != now.Day() || job.LastRun.Month() != now.Month() {
					return true
				}
			}
		}
	}

	return false
}

// Jobs 返回任务列表（可修改）
func (s *Scheduler) Jobs() []Job {
	return s.jobs
}

// JobCount 返回任务数量
func (s *Scheduler) JobCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.jobs)
}

// String 格式化显示
func (s *Scheduler) String() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var sb strings.Builder
	for _, j := range s.jobs {
		status := "⏸"
		if j.Enabled {
			status = "▶"
		}
		sb.WriteString(fmt.Sprintf("  %s %s [%s] %s\n", status, j.Name, j.Cron, j.Prompt[:min(50, len(j.Prompt))]))
	}
	return sb.String()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
