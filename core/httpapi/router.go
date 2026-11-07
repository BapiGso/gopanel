package httpapi

import (
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/labstack/echo/v5"
)

type Operation struct {
	ID            string
	Summary       string
	Request       any
	Response      any
	Status        int
	RequestMedia  string
	ResponseMedia string
	Security      []map[string][]string
}

type Router struct {
	group  *echo.Group
	paths  map[string]any
	prefix string
}

func NewRouter(group *echo.Group) *Router {
	return &Router{group: group, paths: make(map[string]any)}
}

func (router *Router) WithGroup(group *echo.Group, prefix string) *Router {
	return &Router{group: group, paths: router.paths, prefix: prefix}
}

func (router *Router) Handle(method, path string, handler echo.HandlerFunc, operation Operation) {
	if operation.Status == 0 {
		operation.Status = http.StatusOK
	}
	if operation.RequestMedia == "" && operation.Request == nil && method != http.MethodGet {
		next := handler
		handler = func(ctx *echo.Context) error {
			if err := decode(ctx, &struct{}{}, true); err != nil {
				return err
			}
			return next(ctx)
		}
	}
	router.group.Add(method, path, handler)

	parameters := []any{}
	openAPIPath := router.prefix + path
	for _, segment := range strings.Split(path, "/") {
		if strings.HasPrefix(segment, ":") || segment == "*" {
			name := strings.TrimPrefix(segment, ":")
			if name == "*" {
				name = "path"
			}
			openAPIPath = strings.Replace(openAPIPath, segment, "{"+name+"}", 1)
			parameters = append(parameters, map[string]any{
				"name": name, "in": "path", "required": true,
				"schema": map[string]any{"type": "string", "minLength": 1},
			})
		}
	}
	responseMedia := operation.ResponseMedia
	responseSchema := schemaFor(reflect.TypeOf(operation.Response))
	if responseMedia == "" {
		responseMedia = echo.MIMEApplicationJSON
		responseSchema = map[string]any{
			"type": "object", "required": []string{"data"},
			"properties": map[string]any{"data": responseSchema},
		}
	}
	definition := map[string]any{
		"operationId": operation.ID,
		"summary":     operation.Summary,
		"parameters":  parameters,
		"responses": map[string]any{
			strconv.Itoa(operation.Status): map[string]any{
				"description": http.StatusText(operation.Status),
				"content":     map[string]any{responseMedia: map[string]any{"schema": responseSchema}},
			},
			"default": map[string]any{
				"description": "Error; inspect error.code and error.message. No operation is performed on invalid input.",
				"content": map[string]any{echo.MIMEApplicationJSON: map[string]any{
					"schema": map[string]any{"$ref": "#/components/schemas/ErrorResponse"},
				}},
			},
		},
	}
	if operation.Request != nil {
		requestMedia := operation.RequestMedia
		if requestMedia == "" {
			requestMedia = echo.MIMEApplicationJSON
		}
		definition["requestBody"] = map[string]any{
			"required": true,
			"content":  map[string]any{requestMedia: map[string]any{"schema": schemaFor(reflect.TypeOf(operation.Request))}},
		}
	}
	if operation.Security != nil {
		definition["security"] = operation.Security
	}
	methods, exists := router.paths[openAPIPath].(map[string]any)
	if !exists {
		methods = make(map[string]any)
		router.paths[openAPIPath] = methods
	}
	methods[strings.ToLower(method)] = definition
}

func (router *Router) Document() map[string]any {
	return map[string]any{
		"openapi": "3.1.0",
		"info": map[string]any{
			"title": "GoPanel HTTP API", "version": "1.0.0",
			"description": "Authenticated administration API. JSON writes only; GET is read-only. Native Caddyfile and FRP TOML are stored as content strings. Saving does not start or reload services. Filesystem paths may contain slashes; encode each segment. Lifecycle operations are explicit POST endpoints. Use HTTPS with the existing panel session cookie or HTTP Basic credentials. Cron jobs and terminals are in-memory resources.",
		},
		"servers":  []any{map[string]any{"url": Prefix}},
		"paths":    router.paths,
		"security": []any{map[string]any{"panelSession": []string{}}, map[string]any{"panelBasic": []string{}}},
		"components": map[string]any{
			"securitySchemes": map[string]any{
				"panelSession": map[string]any{"type": "apiKey", "in": "cookie", "name": "panel_token"},
				"panelBasic":   map[string]any{"type": "http", "scheme": "basic"},
				"probeToken":   map[string]any{"type": "http", "scheme": "bearer", "description": "Monitor-only cluster key; never grants panel administration permissions"},
			},
			"schemas": map[string]any{
				"ErrorResponse": map[string]any{
					"type": "object", "required": []string{"error"},
					"properties": map[string]any{"error": schemaFor(reflect.TypeOf(Error{}))},
				},
			},
		},
	}
}

func schemaFor(valueType reflect.Type) map[string]any {
	if valueType == nil {
		return map[string]any{}
	}
	for valueType.Kind() == reflect.Pointer {
		valueType = valueType.Elem()
	}
	if valueType.PkgPath() == "time" && valueType.Name() == "Time" {
		return map[string]any{"type": "string", "format": "date-time"}
	}
	switch valueType.Kind() {
	case reflect.Struct:
		properties := make(map[string]any)
		required := []string{}
		for fieldIndex := 0; fieldIndex < valueType.NumField(); fieldIndex++ {
			field := valueType.Field(fieldIndex)
			name := strings.Split(field.Tag.Get("json"), ",")[0]
			if field.Anonymous && name == "" {
				embedded := schemaFor(field.Type)
				if fields, ok := embedded["properties"].(map[string]any); ok {
					for name, property := range fields {
						properties[name] = property
					}
				}
				continue
			}
			if name == "-" || !field.IsExported() {
				continue
			}
			if name == "" {
				name = field.Name
			}
			property := schemaFor(field.Type)
			if format := field.Tag.Get("format"); format != "" {
				property["format"] = format
			}
			for _, constraint := range strings.Split(field.Tag.Get("validate"), ",") {
				if constraint == "required" {
					required = append(required, name)
				}
				if values, ok := strings.CutPrefix(constraint, "oneof="); ok {
					property["enum"] = strings.Fields(values)
				}
				if value, ok := strings.CutPrefix(constraint, "len="); ok {
					if length, err := strconv.Atoi(value); err == nil && property["type"] == "string" {
						property["minLength"] = length
						property["maxLength"] = length
					}
				}
				for _, bound := range []struct{ tag, numeric, text string }{{"min=", "minimum", "minLength"}, {"max=", "maximum", "maxLength"}} {
					if value, ok := strings.CutPrefix(constraint, bound.tag); ok {
						if number, err := strconv.Atoi(value); err == nil {
							keyword := bound.numeric
							if property["type"] == "array" {
								if bound.tag == "min=" {
									keyword = "minItems"
								} else {
									keyword = "maxItems"
								}
							} else if property["type"] == "string" {
								keyword = bound.text
							}
							property[keyword] = number
						}
					}
				}
			}
			properties[name] = property
		}
		return map[string]any{"type": "object", "properties": properties, "required": required, "additionalProperties": false}
	case reflect.Slice, reflect.Array:
		if valueType.Elem().Kind() == reflect.Uint8 {
			return map[string]any{"type": "string", "contentEncoding": "base64"}
		}
		return map[string]any{"type": "array", "items": schemaFor(valueType.Elem())}
	case reflect.Map:
		return map[string]any{"type": "object", "additionalProperties": schemaFor(valueType.Elem())}
	case reflect.String:
		return map[string]any{"type": "string"}
	case reflect.Bool:
		return map[string]any{"type": "boolean"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return map[string]any{"type": "integer"}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return map[string]any{"type": "integer", "minimum": 0}
	case reflect.Float32, reflect.Float64:
		return map[string]any{"type": "number"}
	default:
		return map[string]any{}
	}
}
