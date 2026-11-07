package cron

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/labstack/echo/v5"

	"gopanel/core/httpapi"
)

func TestCronAPIUsesStableNamesAndExplicitPauseState(t *testing.T) {
	oldDirectory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	oldJobs := schedulerList
	schedulerList = nil
	t.Cleanup(func() {
		for _, job := range schedulerList {
			_ = job.Shutdown()
		}
		schedulerList = oldJobs
		_ = os.Chdir(oldDirectory)
	})
	server := echo.New()
	server.HTTPErrorHandler = httpapi.ErrorHandler
	RegisterAPI(httpapi.NewRouter(server.Group(httpapi.Prefix)))
	send := func(method, path, body string, status int) {
		t.Helper()
		request := httptest.NewRequest(method, httpapi.Prefix+path, strings.NewReader(body))
		if body != "" {
			request.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		}
		response := httptest.NewRecorder()
		server.ServeHTTP(response, request)
		if response.Code != status {
			t.Fatalf("%s %s = %d %s, want %d", method, path, response.Code, response.Body.String(), status)
		}
	}
	job := `{"name":"backup","interval_seconds":3600,"script":"echo backup"}`
	send(http.MethodPost, "/cron/jobs", job, http.StatusCreated)
	send(http.MethodPost, "/cron/jobs", job, http.StatusConflict)
	send(http.MethodPatch, "/cron/jobs/backup", `{"paused":true}`, http.StatusOK)
	send(http.MethodPatch, "/cron/jobs/backup", `{"paused":true}`, http.StatusOK)
	send(http.MethodPatch, "/cron/jobs/backup", `{"paused":false}`, http.StatusOK)
	send(http.MethodPut, "/cron/jobs/backup", `{"name":"backup","schedule":"0 2 * * *","script":"echo changed"}`, http.StatusOK)
	send(http.MethodGet, "/cron/jobs/backup", "", http.StatusOK)
	send(http.MethodDelete, "/cron/jobs/backup", "", http.StatusOK)
	send(http.MethodGet, "/cron/jobs/backup", "", http.StatusNotFound)
	if len(schedulerList) != 0 {
		t.Fatal("deleted job remains in scheduler list")
	}
}
