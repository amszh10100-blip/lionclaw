package scheduler

import (
	"log/slog"
	"testing"
	"time"
)

func TestShouldRun_HourMinute(t *testing.T) {
	s := New(slog.Default())

	now := time.Date(2026, 3, 24, 9, 0, 0, 0, time.Local)

	job := &Job{Name: "test", Cron: "09:00", Enabled: true}
	if !s.shouldRun(job, now) {
		t.Error("should run at 09:00")
	}

	job2 := &Job{Name: "test2", Cron: "10:00", Enabled: true}
	if s.shouldRun(job2, now) {
		t.Error("should not run at 09:00 for 10:00 job")
	}
}

func TestShouldRun_Interval(t *testing.T) {
	s := New(slog.Default())

	now := time.Date(2026, 3, 24, 9, 0, 0, 0, time.Local)
	past := now.Add(-10 * time.Minute)

	job := &Job{Name: "test", Cron: "*/5", Enabled: true, LastRun: past}
	if !s.shouldRun(job, now) {
		t.Error("should run: 10min since last, interval 5min")
	}

	// 刚跑过
	recentJob := &Job{Name: "test2", Cron: "*/5", Enabled: true, LastRun: now.Add(-2 * time.Minute)}
	if s.shouldRun(recentJob, now) {
		t.Error("should not run: only 2min since last, interval 5min")
	}
}

func TestShouldRun_NoDuplicate(t *testing.T) {
	s := New(slog.Default())

	now := time.Date(2026, 3, 24, 9, 0, 0, 0, time.Local)

	// 今天已经跑过
	job := &Job{Name: "test", Cron: "09:00", Enabled: true, LastRun: now.Add(-1 * time.Minute)}
	if s.shouldRun(job, now) {
		t.Error("should not run twice on same day")
	}
}

func TestAddJob(t *testing.T) {
	s := New(slog.Default())

	s.AddJob(Job{Name: "j1", Cron: "09:00", Enabled: true})
	s.AddJob(Job{Name: "j2", Cron: "*/30", Enabled: false})

	if s.JobCount() != 2 {
		t.Errorf("JobCount = %d, want 2", s.JobCount())
	}

	jobs := s.Jobs()
	if jobs[0].Name != "j1" {
		t.Errorf("first job = %s, want j1", jobs[0].Name)
	}
}

func TestScheduler_String(t *testing.T) {
	s := New(slog.Default())
	s.AddJob(Job{Name: "morning", Cron: "09:00", Enabled: true, Prompt: "早上好，推送日报"})
	s.AddJob(Job{Name: "check", Cron: "*/60", Enabled: false, Prompt: "检查状态"})

	str := s.String()
	if len(str) == 0 {
		t.Error("String() should not be empty")
	}
}
