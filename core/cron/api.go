package cron

import (
	"net/http"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/labstack/echo/v5"

	"gopanel/core/httpapi"
)

type jobWrite struct {
	Name            string `json:"name" validate:"required,min=1,max=64"`
	Schedule        string `json:"schedule" validate:"max=200"`
	IntervalSeconds int    `json:"interval_seconds" validate:"min=0,max=31536000"`
	Script          string `json:"script" validate:"required,max=1048576"`
}

type jobPatch struct {
	Paused *bool `json:"paused" validate:"required"`
}

type jobView struct {
	Name            string `json:"name"`
	Schedule        string `json:"schedule"`
	IntervalSeconds int    `json:"interval_seconds"`
	Script          string `json:"script"`
	Paused          bool   `json:"paused"`
	LastRun         string `json:"last_run"`
	NextRun         string `json:"next_run"`
}

func RegisterAPI(router *httpapi.Router) {
	router.Handle(http.MethodGet, "/cron/jobs", listJobs, httpapi.Operation{ID: "listCronJobs", Summary: "List in-memory cron jobs using stable names, not array indexes", Response: []jobView{}})
	router.Handle(http.MethodPost, "/cron/jobs", createJob, httpapi.Operation{ID: "createCronJob", Summary: "Create a job with either a five-field cron schedule or interval_seconds, never both", Status: http.StatusCreated, Request: jobWrite{}, Response: jobView{}})
	router.Handle(http.MethodGet, "/cron/jobs/:name", getJob, httpapi.Operation{ID: "getCronJob", Summary: "Read a named cron job", Response: jobView{}})
	router.Handle(http.MethodPut, "/cron/jobs/:name", replaceJob, httpapi.Operation{ID: "replaceCronJob", Summary: "Replace script and schedule; body name must equal path name; paused state is preserved", Request: jobWrite{}, Response: jobView{}})
	router.Handle(http.MethodPatch, "/cron/jobs/:name", patchJob, httpapi.Operation{ID: "setCronJobPaused", Summary: "Explicitly set paused to true or false", Request: jobPatch{}, Response: jobView{}})
	router.Handle(http.MethodDelete, "/cron/jobs/:name", deleteJob, httpapi.Operation{ID: "deleteCronJob", Summary: "Remove a cron job and its script", Response: map[string]string{}})
	router.Handle(http.MethodPost, "/cron/jobs/:name/run", runJob, httpapi.Operation{ID: "runCronJob", Summary: "Queue one run of an unpaused job; acceptance does not report script completion", Status: http.StatusAccepted, Response: jobView{}})
}

func view(job *task) jobView {
	return jobView{
		Name: job.Name(), Schedule: job.Schedule, IntervalSeconds: job.IntervalSeconds, Script: job.Script,
		Paused: job.Paused, LastRun: job.LastRun(), NextRun: job.NextRun(),
	}
}

func listJobs(ctx *echo.Context) error {
	schedulerMutex.Lock()
	defer schedulerMutex.Unlock()
	jobs := make([]jobView, 0, len(schedulerList))
	for jobIndex := range schedulerList {
		jobs = append(jobs, view(&schedulerList[jobIndex]))
	}
	return httpapi.OK(ctx, jobs)
}

func getJob(ctx *echo.Context) error {
	schedulerMutex.Lock()
	defer schedulerMutex.Unlock()
	jobIndex := findJob(ctx.Param("name"))
	if jobIndex < 0 {
		return httpapi.NewError(http.StatusNotFound, "not_found", "Cron job not found")
	}
	return httpapi.OK(ctx, view(&schedulerList[jobIndex]))
}

func definition(request jobWrite) (gocron.JobDefinition, error) {
	if (strings.TrimSpace(request.Schedule) != "") == (request.IntervalSeconds > 0) {
		return nil, httpapi.NewError(http.StatusUnprocessableEntity, "invalid_schedule", "Provide exactly one of schedule or interval_seconds")
	}
	if request.IntervalSeconds > 0 {
		return gocron.DurationJob(time.Duration(request.IntervalSeconds) * time.Second), nil
	}
	return gocron.CronJob(request.Schedule, false), nil
}

func createJob(ctx *echo.Context) error {
	return writeJob(ctx, false)
}

func replaceJob(ctx *echo.Context) error {
	return writeJob(ctx, true)
}

func writeJob(ctx *echo.Context, replace bool) error {
	var request jobWrite
	if err := httpapi.Decode(ctx, &request); err != nil {
		return err
	}
	if replace && request.Name != ctx.Param("name") {
		return httpapi.NewError(http.StatusUnprocessableEntity, "name_mismatch", "Body name must match the resource path")
	}
	jobDefinition, err := definition(request)
	if err != nil {
		return err
	}
	schedulerMutex.Lock()
	defer schedulerMutex.Unlock()
	jobIndex := findJob(request.Name)
	if replace && jobIndex < 0 {
		return httpapi.NewError(http.StatusNotFound, "not_found", "Cron job not found")
	}
	if !replace && jobIndex >= 0 {
		return httpapi.NewError(http.StatusConflict, "already_exists", "A job with this name already exists")
	}
	job, err := createTask(request.Name, request.Script, jobDefinition)
	if err != nil {
		return err
	}
	if _, err := persistCronScript(request.Name, request.Script); err != nil {
		_ = job.Shutdown()
		return err
	}
	job.Schedule = request.Schedule
	job.IntervalSeconds = request.IntervalSeconds
	status := http.StatusOK
	if replace {
		original := &schedulerList[jobIndex]
		_, err := original.Scheduler.Update(original.Jobs()[0].ID(), jobDefinition,
			commandTask(filepath.Join("data", "cron", request.Name+".sh")), gocron.WithName(request.Name))
		_ = job.Shutdown()
		if err != nil {
			_, _ = persistCronScript(original.Name(), original.Script)
			return err
		}
		job.Scheduler = original.Scheduler
		job.Paused = original.Paused
		schedulerList[jobIndex] = job
	} else {
		schedulerList = append(schedulerList, job)
		jobIndex = len(schedulerList) - 1
		status = http.StatusCreated
	}
	if !replace && !job.Paused {
		job.Start()
	}
	return httpapi.Respond(ctx, status, view(&schedulerList[jobIndex]))
}

func patchJob(ctx *echo.Context) error {
	var request jobPatch
	if err := httpapi.Decode(ctx, &request); err != nil {
		return err
	}
	schedulerMutex.Lock()
	defer schedulerMutex.Unlock()
	jobIndex := findJob(ctx.Param("name"))
	if jobIndex < 0 {
		return httpapi.NewError(http.StatusNotFound, "not_found", "Cron job not found")
	}
	job := &schedulerList[jobIndex]
	if *request.Paused != job.Paused {
		if *request.Paused {
			if err := job.StopJobs(); err != nil {
				return err
			}
		} else {
			job.Start()
		}
		job.Paused = *request.Paused
	}
	return httpapi.OK(ctx, view(job))
}

func deleteJob(ctx *echo.Context) error {
	schedulerMutex.Lock()
	defer schedulerMutex.Unlock()
	jobIndex := findJob(ctx.Param("name"))
	if jobIndex < 0 {
		return httpapi.NewError(http.StatusNotFound, "not_found", "Cron job not found")
	}
	name := schedulerList[jobIndex].Name()
	if err := schedulerList[jobIndex].Shutdown(); err != nil {
		return err
	}
	schedulerList = slices.Delete(schedulerList, jobIndex, jobIndex+1)
	if err := removeCronScript(name); err != nil {
		return err
	}
	return httpapi.OK(ctx, map[string]string{"name": name})
}

func runJob(ctx *echo.Context) error {
	schedulerMutex.Lock()
	defer schedulerMutex.Unlock()
	jobIndex := findJob(ctx.Param("name"))
	if jobIndex < 0 {
		return httpapi.NewError(http.StatusNotFound, "not_found", "Cron job not found")
	}
	job := &schedulerList[jobIndex]
	if job.Paused {
		return httpapi.NewError(http.StatusConflict, "job_paused", "Unpause the job before running it")
	}
	if err := job.RunNow(); err != nil {
		return err
	}
	return httpapi.Respond(ctx, http.StatusAccepted, view(job))
}
