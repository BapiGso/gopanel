package httpapi

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/labstack/echo/v5"
)

func Stream(ctx *echo.Context, interval time.Duration, snapshot func() (any, error)) error {
	data, err := snapshot()
	if err != nil {
		return err
	}
	ctx.Response().Header().Set(echo.HeaderContentType, "text/event-stream")
	ctx.Response().Header().Set(echo.HeaderCacheControl, "no-store")
	ctx.Response().Header().Set("X-Accel-Buffering", "no")
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		payload := map[string]any{"data": data}
		if err != nil {
			failure := NewError(http.StatusServiceUnavailable, "snapshot_failed", err.Error())
			if known, ok := err.(*Error); ok {
				failure = known
			}
			payload = map[string]any{"error": failure}
		}
		encoded, marshalError := json.Marshal(payload)
		if marshalError != nil {
			return marshalError
		}
		if _, err := ctx.Response().Write(append(append([]byte("data: "), encoded...), '\n', '\n')); err != nil {
			return err
		}
		if err := http.NewResponseController(ctx.Response()).Flush(); err != nil {
			return err
		}
		select {
		case <-ctx.Request().Context().Done():
			return nil
		case <-ticker.C:
			data, err = snapshot()
		}
	}
}
