package cron

import (
	"github.com/go-co-op/gocron/v2"
	"github.com/labstack/echo/v5"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"gopanel/core/httpapi"
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

func TestPersistCronScriptRejectsInvalidName(t *testing.T) {
	if _, err := persistCronScript("../evil", "echo 1"); err == nil {
		t.Fatal("persistCronScript() should reject invalid job names")
	}
}

func TestPersistCronScriptWritesUnderDataCron(t *testing.T) {
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldWD)
	})

	scriptPath, err := persistCronScript("backup", "#!/bin/sh\necho hello\n")
	if err != nil {
		t.Fatalf("persistCronScript() error = %v", err)
	}
	wantPath := filepath.Join("data", "cron", "backup.sh")
	if scriptPath != wantPath {
		t.Fatalf("scriptPath = %q, want %q", scriptPath, wantPath)
	}

	data, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(data) != "#!/bin/sh\necho hello\n" {
		t.Fatalf("script contents = %q", string(data))
	}
}

func TestAPIRejectsUnknownJobName(t *testing.T) {
	oldSchedulerList := schedulerList
	schedulerList = nil
	t.Cleanup(func() {
		schedulerList = oldSchedulerList
	})

	e := echo.New()
	e.HTTPErrorHandler = httpapi.ErrorHandler
	RegisterAPI(httpapi.NewRouter(e.Group(httpapi.Prefix)))
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/cron/jobs/missing-job", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}
