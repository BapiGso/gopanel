package monitor

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v5"

	"gopanel/core/httpapi"
)

func TestMonitorStreamStopsAfterRequestCancellation(t *testing.T) {
	requestContext, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	request := httptest.NewRequest(http.MethodGet, "/events", nil).WithContext(requestContext)
	response := httptest.NewRecorder()
	streamContext := echo.New().NewContext(request, response)
	samples := 0
	err := httpapi.Stream(streamContext, time.Millisecond, func() (any, error) {
		samples++
		if samples == 2 {
			cancel()
		}
		return map[string]int{"sample": samples}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if samples != 2 || response.Code != http.StatusOK || !response.Flushed {
		t.Fatalf("unexpected stream: samples=%d status=%d flushed=%v", samples, response.Code, response.Flushed)
	}
	if response.Header().Get(echo.HeaderContentType) != "text/event-stream" || strings.Count(response.Body.String(), "data: ") != 2 {
		t.Fatalf("unexpected SSE response: %s", response.Body.String())
	}
}
