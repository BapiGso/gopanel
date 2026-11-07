package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/labstack/echo/v5"

	"gopanel/core/storage"
)

func TestDecodeRequiresJSONAndExplicitBooleans(t *testing.T) {
	server := echo.New()
	server.HTTPErrorHandler = ErrorHandler
	server.PATCH("/settings", func(ctx *echo.Context) error {
		var request struct {
			Autostart *bool `json:"autostart" validate:"required"`
		}
		if err := Decode(ctx, &request); err != nil {
			return err
		}
		return OK(ctx, request)
	})
	cases := []struct {
		name, contentType, body string
		status                  int
	}{
		{"false is a value", echo.MIMEApplicationJSON, `{"autostart":false}`, http.StatusOK},
		{"missing boolean", echo.MIMEApplicationJSON, `{}`, http.StatusUnprocessableEntity},
		{"null boolean", echo.MIMEApplicationJSON, `{"autostart":null}`, http.StatusUnprocessableEntity},
		{"string boolean", echo.MIMEApplicationJSON, `{"autostart":"true"}`, http.StatusBadRequest},
		{"option wrapper", echo.MIMEApplicationJSON, `{"option":{"autostart":true}}`, http.StatusBadRequest},
		{"multiple objects", echo.MIMEApplicationJSON, `{"autostart":true}{}`, http.StatusBadRequest},
		{"null object", echo.MIMEApplicationJSON, `null`, http.StatusBadRequest},
		{"form rejected", echo.MIMEApplicationForm, `autostart=on`, http.StatusUnsupportedMediaType},
		{"no content type", "", `{"autostart":true}`, http.StatusUnsupportedMediaType},
	}
	for _, example := range cases {
		t.Run(example.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPatch, "/settings", strings.NewReader(example.body))
			request.Header.Set(echo.HeaderContentType, example.contentType)
			response := httptest.NewRecorder()
			server.ServeHTTP(response, request)
			if response.Code != example.status {
				t.Fatalf("status = %d, want %d; body = %s", response.Code, example.status, response.Body.String())
			}
			var payload map[string]any
			if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
				t.Fatal(err)
			}
			if example.status == http.StatusOK && !strings.Contains(response.Body.String(), `"autostart":false`) {
				t.Fatalf("explicit false was not preserved: %s", response.Body.String())
			}
			if example.status != http.StatusOK && payload["error"] == nil {
				t.Fatalf("missing error envelope: %s", response.Body.String())
			}
		})
	}
}

func TestBodylessActionsRejectHiddenParameters(t *testing.T) {
	server := echo.New()
	server.HTTPErrorHandler = ErrorHandler
	group := server.Group(Prefix)
	group.Use(Policy)
	router := NewRouter(group)
	calls := 0
	router.Handle(http.MethodPost, "/service/start", func(ctx *echo.Context) error {
		calls++
		return OK(ctx, map[string]string{"state": "running"})
	}, Operation{ID: "start", Summary: "Start"})
	for _, example := range []struct {
		path, body string
		status     int
	}{
		{Prefix + "/service/start", "", http.StatusOK},
		{Prefix + "/service/start", "{}", http.StatusOK},
		{Prefix + "/service/start?option=stop", "", http.StatusBadRequest},
		{Prefix + "/service/start", `{"param":"stop"}`, http.StatusBadRequest},
	} {
		request := httptest.NewRequest(http.MethodPost, example.path, strings.NewReader(example.body))
		if example.body != "" {
			request.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		}
		response := httptest.NewRecorder()
		server.ServeHTTP(response, request)
		if response.Code != example.status {
			t.Fatalf("status = %d, want %d: %s", response.Code, example.status, response.Body.String())
		}
	}
	if calls != 2 {
		t.Fatalf("action performed %d times, want 2", calls)
	}
}

func TestConfigSaveRejectsLostUpdates(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config")
	if err := os.WriteFile(path, []byte("initial"), 0600); err != nil {
		t.Fatal(err)
	}
	file := &storage.File{Path: path}
	initial, err := file.Read()
	if err != nil {
		t.Fatal(err)
	}
	var writers sync.WaitGroup
	failures := make(chan error, 2)
	for _, content := range []string{"first", "second"} {
		writers.Go(func() {
			_, err := file.Save(content, initial.Revision)
			failures <- err
		})
	}
	writers.Wait()
	close(failures)
	successes, conflicts := 0, 0
	for err := range failures {
		if err == nil {
			successes++
		} else if errors.Is(err, storage.ErrRevisionConflict) {
			conflicts++
		} else {
			t.Fatal(err)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("successful writes = %d, conflicts = %d", successes, conflicts)
	}
	current, err := file.Read()
	if err != nil || current.Content == "initial" || current.Revision == initial.Revision {
		t.Fatalf("current = %+v, error = %v", current, err)
	}
}

func TestSchemaDescribesStrictRequestFields(t *testing.T) {
	schema := schemaFor(reflect.TypeOf(ConfigWrite{}))
	properties := schema["properties"].(map[string]any)
	revision := properties["revision"].(map[string]any)
	if revision["minLength"] != 64 || revision["maxLength"] != 64 || schema["additionalProperties"] != false {
		t.Fatalf("incorrect schema: %#v", schema)
	}
}

func TestPathParametersAreDecodedExactlyOnce(t *testing.T) {
	server := echo.New()
	server.GET("/rules/:id", func(ctx *echo.Context) error { return OK(ctx, Param(ctx, "id")) })
	for _, example := range []struct{ path, want string }{
		{"/rules/ipv4%3Ainput%3A1", "ipv4:input:1"},
		{"/rules/percent%25name", "percent%name"},
		{"/rules/literal%252Fname", "literal%2Fname"},
	} {
		response := httptest.NewRecorder()
		server.ServeHTTP(response, httptest.NewRequest(http.MethodGet, example.path, nil))
		var payload struct {
			Data string `json:"data"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
			t.Fatal(err)
		}
		if payload.Data != example.want {
			t.Fatalf("path %s: value = %q, want %q", example.path, payload.Data, example.want)
		}
	}
}
