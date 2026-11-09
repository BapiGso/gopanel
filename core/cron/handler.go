package cron

import (
	"errors"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"sync"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/labstack/echo/v5"

	"gopanel/core/httpapi"
	"gopanel/core/storage"
)

type task struct {
	Paused          bool
	Schedule        string
	IntervalSeconds int
	Script          string
	gocron.Scheduler
}

func (job *task) LastRun() string {
	lastRun, _ := job.Scheduler.Jobs()[0].LastRunStartedAt()
	if lastRun.IsZero() {
		return ""
	}
	return lastRun.Format(time.RFC3339)
}

func (job *task) NextRun() string {
	nextRun, _ := job.Scheduler.Jobs()[0].NextRun()
	if nextRun.IsZero() {
		return ""
	}
	return nextRun.Format(time.RFC3339)
}

func (job *task) RunNow() error {
	return job.Scheduler.Jobs()[0].RunNow()
}

func (job *task) Name() string {
	return job.Scheduler.Jobs()[0].Name()
}

var (
	schedulerList   []task
	schedulerMutex  sync.Mutex
	cronNamePattern = regexp.MustCompile(`^[A-Za-z0-9._-]{1,64}$`)
)

func Index(ctx *echo.Context) error {
	schedulerMutex.Lock()
	defer schedulerMutex.Unlock()
	jobs := make([]jobView, 0, len(schedulerList))
	for jobIndex := range schedulerList {
		jobs = append(jobs, view(&schedulerList[jobIndex]))
	}
	return ctx.Render(http.StatusOK, "cron.template", jobs)
}

func createTask(name, script string, definition gocron.JobDefinition) (task, error) {
	if !cronNamePattern.MatchString(name) {
		return task{}, httpapi.NewError(http.StatusUnprocessableEntity, "invalid_name", "Job name must match [A-Za-z0-9._-]{1,64}")
	}
	if definition == nil {
		return task{}, httpapi.NewError(http.StatusUnprocessableEntity, "invalid_schedule", "Invalid job schedule")
	}
	scheduler, err := gocron.NewScheduler()
	if err != nil {
		return task{}, err
	}
	scriptPath := filepath.Join("data", "cron", name+".sh")
	_, err = scheduler.NewJob(definition, commandTask(scriptPath), gocron.WithName(name))
	if err != nil {
		_ = scheduler.Shutdown()
		return task{}, httpapi.NewError(http.StatusUnprocessableEntity, "invalid_schedule", err.Error())
	}
	return task{Scheduler: scheduler, Script: script}, nil
}

func commandTask(scriptPath string) gocron.Task {
	return gocron.NewTask(func() {
		_ = exec.Command(scriptRunnerCommand(), scriptRunnerArgs(scriptPath)...).Run()
	})
}

func findJob(name string) int {
	return slices.IndexFunc(schedulerList, func(job task) bool { return job.Name() == name })
}

func persistCronScript(name, content string) (string, error) {
	if !cronNamePattern.MatchString(name) {
		return "", echo.NewHTTPError(http.StatusBadRequest, "invalid cron job name")
	}
	directory := filepath.Join("data", "cron")
	if err := os.MkdirAll(directory, 0700); err != nil {
		return "", err
	}
	path := filepath.Join(directory, name+".sh")
	if err := storage.AtomicWrite(path, []byte(content)); err != nil {
		return "", err
	}
	return path, nil
}

func scriptRunnerCommand() string {
	return "sh"
}

func scriptRunnerArgs(scriptPath string) []string {
	return []string{scriptPath}
}

func removeCronScript(name string) error {
	if !cronNamePattern.MatchString(name) {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid cron job name")
	}
	err := os.Remove(filepath.Join("data", "cron", name+".sh"))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}
