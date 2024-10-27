package cron

import (
	"github.com/go-co-op/gocron/v2"
	"testing"
	"time"
)

func TestCron(t *testing.T) {
	s, _ := gocron.NewScheduler()
	// 每3秒执行一次
	j, _ := s.NewJob(
		gocron.DurationJob(
			10*time.Second,
		),
		gocron.NewTask(
			func(a string, b int) {
				t.Log(a, b)
			},
			"hello",
			1,
		),
	)
	s.Start()
	time.Sleep(time.Minute)
	t.Logf("%v", j)

}
