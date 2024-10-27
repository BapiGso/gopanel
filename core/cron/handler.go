package cron

import (
	"encoding/json"
	"fmt"
	"github.com/go-co-op/gocron/v2"
	"github.com/labstack/echo/v4"
	"github.com/spf13/viper"
	"log/slog"
	"net/http"
	"os/exec"
	"time"
)

func Index(c echo.Context) error {
	req := &struct {
		Name      string `form:"name"         json:"name"`
		Frequency int    `form:"frequency"    json:"frequency"`
		AtTime    int    `form:"attime"       json:"attime"`
		Script    string `form:"script"       json:"script"`
		Enable    bool   `json:"enable"`
	}{
		Enable: true,
	}
	if err := c.Bind(req); err != nil {
		return err
	}
	switch c.Request().Method {
	case "POST":
		//viper.Set("cron."+req.Name, req)
		//tmp := map[string]any{req.Name: req}
		//maps.Copy(tmp, viper.GetStringMap("cron"))
		//viper.Set("cron", tmp)
		//if err := viper.WriteConfig(); err != nil {
		//	return err // 处理错误
		//}
		go addCron(req.Name, req.Script, req.Frequency, req.AtTime)
		return c.JSON(200, "success")
	//case "PUT":
	//	switch c.QueryParam("type") {
	//	case "pause":
	//		viper.Set("cron."+c.QueryParam("name")+".enable", false)
	//	case "unpause":
	//		viper.Set("cron."+c.QueryParam("name")+".enable", true)
	//	case "remove":
	//
	//	}
	//	if err := viper.WriteConfig(); err != nil {
	//		return err // 处理错误
	//	}
	//	return c.JSON(200, "success")
	case "GET":
		jsonData, _ := json.Marshal(viper.Get("cron"))
		return c.Render(http.StatusOK, "cron.template", string(jsonData))
	}
	return echo.ErrMethodNotAllowed
}

func addCron(name, cmd string, fre, atTime int) {

	s, err := gocron.NewScheduler()
	if err != nil {
		slog.Error("create NewScheduler err", err)
	}
	j, err := s.NewJob(
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
		slog.Error("create Job err", err)
	}
	slog.Info(fmt.Sprintf("Job running:%v", j.ID()))
	s.Start()
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
