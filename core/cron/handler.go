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
	lastRun, _ := job.Scheduler.Jobs()[0].LastRun()
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
	return ctx.Render(http.StatusOK, "cron.template", schedulerList)
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

func addCron(name, script string, definition gocron.JobDefinition) error {
	schedulerMutex.Lock()
	defer schedulerMutex.Unlock()
	if findJob(name) >= 0 {
		return httpapi.NewError(http.StatusConflict, "already_exists", "A job with this name already exists")
	}
	job, err := createTask(name, script, definition)
	if err != nil {
		return err
	}
	if _, err := persistCronScript(name, script); err != nil {
		_ = job.Shutdown()
		return err
	}
	schedulerList = append(schedulerList, job)
	job.Start()
	return nil
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

func getJobDefinition(frequency, atTime int) gocron.JobDefinition {
	weekday := time.Weekday(atTime/(24*60)) % 7
	day := atTime / (24 * 60)
	hour := uint((atTime / 60) % 24)
	minute := uint(atTime % 60)
	switch frequency {
	case 43199:
		return gocron.MonthlyJob(1, gocron.NewDaysOfTheMonth(day), gocron.NewAtTimes(gocron.NewAtTime(hour, minute, 0)))
	case 10079:
		return gocron.WeeklyJob(1, gocron.NewWeekdays(weekday), gocron.NewAtTimes(gocron.NewAtTime(hour, minute, 0)))
	case 1439:
		return gocron.DailyJob(1, gocron.NewAtTimes(gocron.NewAtTime(hour, minute, 0)))
	case 43200:
		return gocron.DurationJob(time.Minute * time.Duration(atTime))
	}
	return nil
}
