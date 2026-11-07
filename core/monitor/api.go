package monitor

import (
	"crypto/sha256"
	"crypto/subtle"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"

	"gopanel/core/httpapi"
)

func (service *Service) RegisterAPI(router *httpapi.Router, server *echo.Echo) {
	service.registerReadRoutes(router, "/monitor", "")
	public := server.Group(httpapi.Prefix + "/public/monitor")
	public.Use(middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(10)), httpapi.Policy, service.publicAccess)
	service.registerReadRoutes(router.WithGroup(public, "/public/monitor"), "", "Public")
	router.WithGroup(public, "/public/monitor").Handle(http.MethodGet, "/config", service.publicConfig, httpapi.Operation{ID: "getPublicMonitorConfig", Summary: "Read non-secret monitor presentation settings", Response: map[string]any{}, Security: []map[string][]string{}})
	server.GET("/status/monitor", service.publicPage)
	ingest := server.Group(httpapi.Prefix + "/monitor/reports")
	ingest.Use(middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(200)), httpapi.Policy, service.probeAuth)
	router.WithGroup(ingest, "/monitor/reports").Handle(http.MethodPost, "", service.receiveReport, httpapi.Operation{
		ID: "reportMonitorMetrics", Summary: "Submit a monitor-only report to a candidate using its cluster key; no panel commands can be executed",
		Request: Report{}, Response: peerReply{}, Security: []map[string][]string{{"probeToken": {}}},
	})
}

func (service *Service) registerReadRoutes(router *httpapi.Router, prefix, suffix string) {
	var security []map[string][]string
	if suffix == "Public" {
		security = []map[string][]string{}
	}
	router.Handle(http.MethodGet, prefix+"/nodes", service.nodes, httpapi.Operation{
		ID: "listMonitorNodes" + suffix, Summary: "Read cached, sanitized multi-node metrics; no live sampling occurs per request", Response: Overview{}, Security: security,
	})
	router.Handle(http.MethodGet, prefix+"/nodes/:id", service.node, httpapi.Operation{
		ID: "getMonitorNode" + suffix, Summary: "Read a node's cached metrics; online is based on report receipt time", Response: NodeView{}, Security: security,
	})
	router.Handle(http.MethodGet, prefix+"/nodes/:id/history", service.history, httpapi.Operation{
		ID: "getMonitorNodeHistory" + suffix, Summary: "Read bounded in-memory history; samples are not persisted across aggregator restart", Response: []Sample{}, Security: security,
	})
	router.Handle(http.MethodGet, prefix+"/events", service.events, httpapi.Operation{
		ID: "streamMonitorNodes" + suffix, Summary: "Stream sanitized aggregate snapshots as SSE; reconnecting returns the latest state", ResponseMedia: "text/event-stream", Response: "", Security: security,
	})
	router.Handle(http.MethodGet, prefix+"/cluster", service.cluster, httpapi.Operation{
		ID: "getMonitorCluster" + suffix, Summary: "Read best-effort preferred aggregator status; this is not a consensus or management leader", Response: ClusterView{}, Security: security,
	})
	if suffix == "" {
		router.Handle(http.MethodGet, prefix+"/status", service.local, httpapi.Operation{
			ID: "getMonitorStatus", Summary: "Read cached local metrics without blocking for a new CPU sample", Response: NodeView{},
		})
	}
}

func (service *Service) nodes(ctx *echo.Context) error {
	return httpapi.OK(ctx, service.Overview())
}

func (service *Service) node(ctx *echo.Context) error {
	view, exists := service.registry.Node(httpapi.Param(ctx, "id"), time.Now())
	if !exists {
		return httpapi.NewError(http.StatusNotFound, "not_found", "Monitor node not found")
	}
	return httpapi.OK(ctx, view)
}

func (service *Service) history(ctx *echo.Context) error {
	view, exists := service.registry.History(httpapi.Param(ctx, "id"))
	if !exists {
		return httpapi.NewError(http.StatusNotFound, "not_found", "Monitor node not found")
	}
	return httpapi.OK(ctx, view)
}

func (service *Service) local(ctx *echo.Context) error {
	view, exists := service.registry.Node(service.settings.NodeID, time.Now())
	if !exists {
		return httpapi.NewError(http.StatusServiceUnavailable, "sampling_not_ready", "The local collector has not produced its first sample")
	}
	return httpapi.OK(ctx, view)
}

func (service *Service) events(ctx *echo.Context) error {
	select {
	case service.streams <- struct{}{}:
		defer func() { <-service.streams }()
	default:
		return httpapi.NewError(http.StatusServiceUnavailable, "viewer_limit", "Monitor stream capacity reached; retry later")
	}
	return httpapi.Stream(ctx, time.Duration(service.settings.SampleSeconds)*time.Second, func() (any, error) {
		return service.Overview(), nil
	})
}

func (service *Service) cluster(ctx *echo.Context) error {
	return httpapi.OK(ctx, service.Cluster(time.Now()))
}

func (service *Service) publicConfig(ctx *echo.Context) error {
	return httpapi.OK(ctx, map[string]any{"title": service.settings.Title, "read_only": true, "coordination": service.Cluster(time.Now()).Coordination, "sample_seconds": service.settings.SampleSeconds})
}

func (service *Service) publicAccess(next echo.HandlerFunc) echo.HandlerFunc {
	return func(ctx *echo.Context) error {
		if !service.settings.Public {
			return httpapi.NewError(http.StatusNotFound, "not_found", "Public monitoring is disabled")
		}
		return next(ctx)
	}
}

func (service *Service) probeAuth(next echo.HandlerFunc) echo.HandlerFunc {
	return func(ctx *echo.Context) error {
		if service.settings.Role != "candidate" {
			return httpapi.NewError(http.StatusNotFound, "not_found", "This node does not accept remote metric reports")
		}
		authorization := ctx.Request().Header.Get("Authorization")
		if !strings.HasPrefix(authorization, "Bearer ") || !sameSecret(strings.TrimPrefix(authorization, "Bearer "), service.settings.ClusterKey) || !sameSecret(ctx.Request().Header.Get("X-GoPanel-Cluster"), service.settings.ClusterID) {
			return httpapi.NewError(http.StatusUnauthorized, "invalid_probe_credentials", "Use the monitor cluster key and X-GoPanel-Cluster header")
		}
		return next(ctx)
	}
}

func sameSecret(actual, expected string) bool {
	first := sha256.Sum256([]byte(actual))
	second := sha256.Sum256([]byte(expected))
	return subtle.ConstantTimeCompare(first[:], second[:]) == 1
}

func (service *Service) receiveReport(ctx *echo.Context) error {
	var report Report
	if err := httpapi.DecodeLimit(ctx, &report, 32<<10); err != nil {
		return err
	}
	if report.Node.ID == service.settings.NodeID {
		return httpapi.NewError(http.StatusConflict, "node_id_conflict", "A remote probe cannot replace the candidate's local node identity; use unique node_id values")
	}
	if err := service.registry.Accept(report, time.Now()); err != nil {
		return httpapi.NewError(http.StatusUnprocessableEntity, "invalid_report", err.Error())
	}
	return httpapi.OK(ctx, peerReply{ClusterID: service.settings.ClusterID, ID: service.settings.NodeID, Name: service.settings.Name})
}
