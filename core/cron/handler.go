package cron

import (
	"errors"
	"fmt"
	"github.com/go-co-op/gocron/v2"
	"github.com/labstack/echo/v5"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"time"
)

type task struct {
	Paused bool
	gocron.Scheduler
}

func (t *task) LastRun() string {
	lastRun, _ := t.Scheduler.Jobs()[0].LastRun()
	return lastRun.Format("2006-01-02T15:04")
}

func (t *task) NextRun() string {
	nextRun, _ := t.Scheduler.Jobs()[0].NextRun()
	return nextRun.Format("2006-01-02T15:04")
}

func (t *task) RunNow() error {
	return t.Scheduler.Jobs()[0].RunNow()
}

func (t *task) Name() string {
	name := t.Scheduler.Jobs()[0].Name()
	return name
}

var schedulerList []task

var cronNamePattern = regexp.MustCompile(`^[A-Za-z0-9._-]{1,64}$`)

func Index(c *echo.Context) error {
	req := &struct {
		Name      string `form:"name"         json:"name"`
		Frequency string `form:"frequency"    json:"frequency"`
		AtTime    string `form:"attime"       json:"attime"`
		Script    string `form:"script"       json:"script"`
		Index     int    `query:"index"`
	}{}
	if err := c.Bind(req); err != nil {
		return err
	}
	switch c.Request().Method {
	case "POST":
		// 转换字符串为整数
		var frequency, atTime int
		if _, err := fmt.Sscanf(req.Frequency, "%d", &frequency); err != nil {
			return c.JSON(400, map[string]string{"error": "Invalid frequency value"})
		}
		if _, err := fmt.Sscanf(req.AtTime, "%d", &atTime); err != nil {
			return c.JSON(400, map[string]string{"error": "Invalid attime value"})
		}

		if err := addCron(req.Name, req.Script, getJobDefinition(frequency, atTime)); err != nil {
			return c.JSON(500, map[string]string{"error": err.Error()})
		}
		return c.JSON(200, map[string]string{"message": "Cron job created successfully"})
	case "PUT":
		if req.Index < 0 || req.Index >= len(schedulerList) {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid index"})
		}
		switch c.QueryParam("type") {
		case "pause":
			if err := schedulerList[req.Index].StopJobs(); err != nil {
				return err
			}
			schedulerList[req.Index].Paused = true
		case "unpause":
			schedulerList[req.Index].Start()
			schedulerList[req.Index].Paused = false
		case "remove":
			name := schedulerList[req.Index].Name()
			if err := schedulerList[req.Index].Shutdown(); err != nil {
				return err
			}
			schedulerList = slices.Delete(schedulerList, req.Index, req.Index+1)
			if err := removeCronScript(name); err != nil {
				return err
			}
		case "runnow":
			if err := schedulerList[req.Index].RunNow(); err != nil {
				return err
			}
		}
		return c.JSON(200, "success")
	case "GET":
		return c.Render(http.StatusOK, "cron.template", schedulerList)
	}
	return echo.ErrMethodNotAllowed
}

func addCron(name, cmd string, jobDefinition gocron.JobDefinition) error {
	if jobDefinition == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid job definition")
	}
	scriptPath, err := persistCronScript(name, cmd)
	if err != nil {
		return err
	}

	var s task
	if s.Scheduler, err = gocron.NewScheduler(); err != nil {
		return err
	}

	_, err = s.NewJob(
		jobDefinition,
		gocron.NewTask(
			func() {
				_ = exec.Command(scriptRunnerCommand(), scriptRunnerArgs(scriptPath)...).Run()
			},
		),
		gocron.WithName(name),
	)
	if err != nil {
		return err
	}
	schedulerList = append(schedulerList, s)
	s.Start()
	return nil
}

func persistCronScript(name, content string) (string, error) {
	if !cronNamePattern.MatchString(name) {
		return "", echo.NewHTTPError(http.StatusBadRequest, "invalid cron job name")
	}
	dir := filepath.Join("data", "cron")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}

	path := filepath.Join(dir, name+".sh")
	if err := os.WriteFile(path, []byte(content), fs.FileMode(0600)); err != nil {
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

func getJobDefinition(fre, atTime int) gocron.JobDefinition {
	weekday := time.Weekday(atTime/(24*60)) % 7
	day := atTime / (24 * 60)
	hour := uint((atTime / 60) % 24)
	minute := uint(atTime % 60)

	switch fre {
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
