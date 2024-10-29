package cron

import (
	"encoding/json"
	"github.com/go-co-op/gocron/v2"
	"github.com/labstack/echo/v4"
	"log/slog"
	"net/http"
	"os/exec"
	"time"
)

// 全局map存储所有scheduler
var schedulerMap = make(map[string]gocron.Scheduler)

func Index(c echo.Context) error {
	req := &struct {
		Name      string `form:"name"         json:"name"`
		Frequency int    `form:"frequency"    json:"frequency"`
		AtTime    int    `form:"attime"       json:"attime"`
		Script    string `form:"script"       json:"script"`
	}{}
	if err := c.Bind(req); err != nil {
		return err
	}

	switch c.Request().Method {
	case "POST":
		if err := addCron(req.Name, req.Script, req.Frequency, req.AtTime); err != nil {
			return err
		}
		return c.JSON(200, "success")
	case "PUT":
		switch c.QueryParam("type") {
		case "pause":
			if s, exists := schedulerMap[c.QueryParam("name")]; exists {
				if err := s.StopJobs(); err != nil {
					return err
				}
				return c.JSON(200, "success")
			}
		case "unpause":
			if oldS, exists := schedulerMap[c.QueryParam("name")]; exists {
				oldS.Start()
				return c.JSON(200, "success")
			}
		case "remove":
			if s, exists := schedulerMap[c.QueryParam("name")]; exists {
				s.Shutdown()
				delete(schedulerMap, c.QueryParam("name"))
				return c.JSON(200, "success")
			}
		}
		return c.JSON(404, "scheduler not found")
	case "GET":
		jsonData, _ := json.Marshal(schedulerMap)
		return c.Render(http.StatusOK, "cron.template", string(jsonData))
	}
	return echo.ErrMethodNotAllowed
}

func addCron(name, cmd string, fre, atTime int) error {
	// 如果已存在，先关闭旧的
	if oldS, exists := schedulerMap[name]; exists {
		oldS.Shutdown()
	}

	// 创建新的scheduler
	s, err := gocron.NewScheduler()
	if err != nil {

		return err
	}

	_, err = s.NewJob(
		getJobDefinition(fre, atTime),
		gocron.NewTask(
			func() {
				if err := exec.Command(cmd).Run(); err != nil {
					slog.Error("cmd exec err", err)
				} else {
					slog.Info("success", time.Now().Unix())
				}
			},
		),
	)
	if err != nil {
		return err
	}

	// 保存到map中
	schedulerMap[name] = s

	s.Start()
	return nil
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
