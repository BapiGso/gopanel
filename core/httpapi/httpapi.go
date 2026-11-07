package httpapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v5"

	"gopanel/core/config"
	"gopanel/core/storage"
)

const Prefix = "/api/v1"

var validate = func() *validator.Validate {
	checker := validator.New()
	checker.RegisterTagNameFunc(func(field reflect.StructField) string {
		return strings.Split(field.Tag.Get("json"), ",")[0]
	})
	return checker
}()

type Error struct {
	Status  int    `json:"-"`
	Code    string `json:"code" validate:"required"`
	Message string `json:"message" validate:"required"`
	Details any    `json:"details,omitempty"`
}

func (failure *Error) Error() string {
	return failure.Message
}

func (failure *Error) StatusCode() int {
	return failure.Status
}

func NewError(status int, code, message string) *Error {
	return &Error{Status: status, Code: code, Message: message}
}

func OK(ctx *echo.Context, data any) error {
	return Respond(ctx, http.StatusOK, data)
}

func Respond(ctx *echo.Context, status int, data any) error {
	return ctx.JSON(status, map[string]any{"data": data})
}

func Param(ctx *echo.Context, name string) string {
	value := ctx.Param(name)
	if ctx.Request().URL.RawPath != "" {
		if decoded, err := url.PathUnescape(value); err == nil {
			return decoded
		}
	}
	return value
}

func Decode(ctx *echo.Context, target any) error {
	return decode(ctx, target, false)
}

func DecodeLimit(ctx *echo.Context, target any, maximum int64) error {
	return decodeLimit(ctx, target, false, maximum)
}

func decode(ctx *echo.Context, target any, optional bool) error {
	return decodeLimit(ctx, target, optional, 4<<20)
}

func decodeLimit(ctx *echo.Context, target any, optional bool, maximum int64) error {
	request := ctx.Request()
	if optional && request.ContentLength == 0 {
		return nil
	}
	mediaType, _, err := mime.ParseMediaType(request.Header.Get(echo.HeaderContentType))
	if err != nil || mediaType != echo.MIMEApplicationJSON {
		return NewError(http.StatusUnsupportedMediaType, "unsupported_media_type", "Use Content-Type: application/json")
	}
	request.Body = http.MaxBytesReader(ctx.Response(), request.Body, maximum)
	decoder := json.NewDecoder(request.Body)
	var raw json.RawMessage
	if err := decoder.Decode(&raw); err != nil {
		var oversized *http.MaxBytesError
		if errors.As(err, &oversized) {
			return NewError(http.StatusRequestEntityTooLarge, "body_too_large", fmt.Sprintf("JSON body exceeds %d bytes", maximum))
		}
		return NewError(http.StatusBadRequest, "invalid_json", "Invalid JSON body: "+err.Error())
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return NewError(http.StatusBadRequest, "invalid_json", "Send exactly one JSON object")
	}
	if !bytes.HasPrefix(bytes.TrimSpace(raw), []byte("{")) {
		return NewError(http.StatusBadRequest, "invalid_json", "Send a JSON object")
	}
	fields := json.NewDecoder(bytes.NewReader(raw))
	fields.DisallowUnknownFields()
	if err := fields.Decode(target); err != nil {
		return NewError(http.StatusBadRequest, "invalid_json", "Invalid JSON body: "+err.Error())
	}
	if err := validate.Struct(target); err != nil {
		issues := []map[string]string{}
		var failures validator.ValidationErrors
		if errors.As(err, &failures) {
			for _, field := range failures {
				issues = append(issues, map[string]string{"field": field.Field(), "rule": field.Tag(), "parameter": field.Param()})
			}
		}
		return &Error{
			Status:  http.StatusUnprocessableEntity,
			Code:    "validation_failed",
			Message: "Invalid request fields",
			Details: issues,
		}
	}
	return nil
}

func Policy(next echo.HandlerFunc) echo.HandlerFunc {
	return func(ctx *echo.Context) error {
		ctx.Response().Header().Set(echo.HeaderCacheControl, "no-store")
		ctx.Response().Header().Set("X-Content-Type-Options", "nosniff")
		request := ctx.Request()
		origin := request.Header.Get("Origin")
		if origin != "" && (request.Method != http.MethodGet || strings.EqualFold(request.Header.Get("Upgrade"), "websocket")) {
			parsed, err := url.Parse(origin)
			if err != nil || parsed.Host != request.Host || parsed.Scheme != ctx.Scheme() {
				return NewError(http.StatusForbidden, "cross_origin_forbidden", "Cross-origin administration requests are not allowed")
			}
		}
		if ctx.Request().URL.RawQuery != "" {
			return NewError(http.StatusBadRequest, "unexpected_query", "This endpoint does not accept query parameters; use resource paths and documented JSON bodies")
		}
		return next(ctx)
	}
}

func IsRequest(ctx *echo.Context) bool {
	return ctx.Request().URL.Path == Prefix || strings.HasPrefix(ctx.Request().URL.Path, Prefix+"/")
}

func ErrorHandler(ctx *echo.Context, err error) {
	response, _ := echo.UnwrapResponse(ctx.Response())
	if err == nil || response != nil && response.Committed {
		return
	}
	failure := NewError(http.StatusInternalServerError, "internal_error", "Internal server error")
	var apiError *Error
	var httpError *echo.HTTPError
	var configError *config.ValidationError
	switch {
	case errors.As(err, &apiError):
		failure = apiError
	case errors.Is(err, storage.ErrRevisionConflict):
		failure = NewError(http.StatusConflict, "revision_conflict", "Configuration changed; read it again before saving")
	case errors.As(err, &configError):
		failure = NewError(http.StatusUnprocessableEntity, "invalid_config", configError.Error())
		var fields validator.ValidationErrors
		if errors.As(err, &fields) {
			issues := []map[string]string{}
			for _, field := range fields {
				issues = append(issues, map[string]string{"field": field.Field(), "rule": field.Tag(), "parameter": field.Param()})
			}
			failure.Details = issues
		}
	case errors.As(err, &httpError):
		failure.Status = httpError.Code
		failure.Code = statusCode(httpError.Code)
		if httpError.Code < http.StatusInternalServerError {
			failure.Message = fmt.Sprint(httpError.Message)
		}
	case echo.StatusCode(err) != 0:
		status := echo.StatusCode(err)
		failure = NewError(status, statusCode(status), http.StatusText(status))
	case errors.Is(err, os.ErrNotExist):
		failure = NewError(http.StatusNotFound, "not_found", "Resource not found")
	case errors.Is(err, os.ErrExist):
		failure = NewError(http.StatusConflict, "already_exists", "Resource already exists")
	case errors.Is(err, os.ErrPermission):
		failure = NewError(http.StatusForbidden, "forbidden", "Permission denied")
	}
	if failure.Status >= http.StatusInternalServerError {
		ctx.Logger().Error("API request failed", "error", err)
	}
	_ = ctx.JSON(failure.Status, map[string]any{"error": failure})
}

func statusCode(status int) string {
	switch status {
	case http.StatusBadRequest:
		return "bad_request"
	case http.StatusUnauthorized:
		return "unauthorized"
	case http.StatusForbidden:
		return "forbidden"
	case http.StatusNotFound:
		return "not_found"
	case http.StatusMethodNotAllowed:
		return "method_not_allowed"
	case http.StatusConflict:
		return "conflict"
	case http.StatusTooManyRequests:
		return "rate_limited"
	default:
		return "request_failed"
	}
}
