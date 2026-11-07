package monitor

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"

	"gopanel/core/httpapi"
)

const probeReportPath = "/api/v1/probe/report"
const probeTimeHeader = "X-GoPanel-Probe-Time"
const probeNonceHeader = "X-GoPanel-Probe-Nonce"
const probeSignatureHeader = "X-GoPanel-Probe-Signature"

func probeSignature(key string, parts ...[]byte) string {
	signature := hmac.New(sha256.New, []byte(key))
	for _, part := range parts {
		signature.Write(part)
	}
	return hex.EncodeToString(signature.Sum(nil))
}

func probeRequestSignature(key, timestamp, nonce string) string {
	return probeSignature(key, []byte("gopanel-probe-v1\nrequest\nGET\n"+probeReportPath+"\n"+timestamp+"\n"+nonce))
}

func probeResponseSignature(key, timestamp, nonce string, content []byte) string {
	return probeSignature(key, []byte("gopanel-probe-v1\nresponse\n"+timestamp+"\n"+nonce+"\n"), content)
}

func (service *Service) probeRoutes() *echo.Echo {
	server := echo.New()
	server.HTTPErrorHandler = httpapi.ErrorHandler
	server.Use(middleware.Recover(), middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(10)), httpapi.Policy)
	server.GET(probeReportPath, service.probeReport)
	return server
}

func (service *Service) probeReport(ctx *echo.Context) error {
	request := ctx.Request()
	timestamp := request.Header.Get(probeTimeHeader)
	nonce := request.Header.Get(probeNonceHeader)
	seconds, err := strconv.ParseInt(timestamp, 10, 64)
	now := time.Now()
	if service.settings.Key == "" || err != nil || len(nonce) < 16 || len(nonce) > 80 || strings.ContainsAny(nonce, "\r\n") || time.Unix(seconds, 0).Before(now.Add(-time.Minute)) || time.Unix(seconds, 0).After(now.Add(time.Minute)) || !sameSecret(request.Header.Get(probeSignatureHeader), probeRequestSignature(service.settings.Key, timestamp, nonce)) {
		return httpapi.NewError(http.StatusUnauthorized, "invalid_probe_identity", "Use a fresh request signed with the shared monitor key")
	}
	service.mutex.RLock()
	report := service.latest
	service.mutex.RUnlock()
	if report.Sequence == 0 {
		return httpapi.NewError(http.StatusServiceUnavailable, "sampling_not_ready", "The probe has not produced its first sample")
	}
	content, err := json.Marshal(struct {
		Data Report `json:"data"`
	}{Data: report})
	if err != nil {
		return err
	}
	ctx.Response().Header().Set(probeSignatureHeader, probeResponseSignature(service.settings.Key, timestamp, nonce, content))
	ctx.Response().Header().Set("Cache-Control", "no-store")
	return ctx.Blob(http.StatusOK, "application/json", content)
}

func (service *Service) startProbeServer() error {
	if service.settings.Key == "" || service.settings.Role == "master" {
		return nil
	}
	listener, err := net.Listen("tcp", service.settings.probeListenAddress())
	if err != nil {
		return fmt.Errorf("listen for read-only probe requests: %w", err)
	}
	server := &http.Server{
		Handler: service.probeRoutes(), ReadHeaderTimeout: 3 * time.Second,
		ReadTimeout: 5 * time.Second, WriteTimeout: 5 * time.Second,
		IdleTimeout: 30 * time.Second, MaxHeaderBytes: 16 << 10,
	}
	service.probeServer = server
	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			service.logger.Error("read-only probe server failed", "error", err)
			service.cancel()
		}
	}()
	service.logger.Info("read-only probe listening", "address", listener.Addr().String())
	return nil
}

func (service *Service) stopProbeServer() {
	if service.probeServer == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := service.probeServer.Shutdown(ctx); err != nil {
		service.probeServer.Close()
	}
}

func (service *Service) pullLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(service.settings.SampleSeconds) * time.Second)
	defer ticker.Stop()
	for {
		service.pullPeers(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (service *Service) pullPeers(ctx context.Context) {
	jobs := make(chan string, len(service.settings.Peers))
	for _, peer := range service.settings.Peers {
		jobs <- peer
	}
	close(jobs)
	var workers sync.WaitGroup
	for range min(16, len(service.settings.Peers)) {
		workers.Go(func() {
			for address := range jobs {
				if ctx.Err() != nil {
					return
				}
				report, err := service.pull(ctx, address)
				if err == nil {
					err = service.acceptPeer(address, report)
				}
				if ctx.Err() != nil {
					return
				}
				message := ""
				if err != nil {
					message = err.Error()
				}
				service.mutex.Lock()
				previous := service.peerErrors[address]
				service.peerErrors[address] = message
				service.mutex.Unlock()
				if message != "" && message != previous {
					service.logger.Warn("monitor peer unavailable", "peer", address, "error", err)
				}
			}
		})
	}
	workers.Wait()
}

func (service *Service) pull(ctx context.Context, address string) (Report, error) {
	timestamp, nonce := strconv.FormatInt(time.Now().Unix(), 10), rand.Text()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, address+probeReportPath, nil)
	if err != nil {
		return Report{}, err
	}
	request.Header.Set(probeTimeHeader, timestamp)
	request.Header.Set(probeNonceHeader, nonce)
	request.Header.Set(probeSignatureHeader, probeRequestSignature(service.settings.Key, timestamp, nonce))
	response, err := service.client.Do(request)
	if err != nil {
		return Report{}, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return Report{}, fmt.Errorf("probe returned HTTP %d", response.StatusCode)
	}
	content, err := io.ReadAll(io.LimitReader(response.Body, (32<<10)+1))
	if err != nil {
		return Report{}, err
	}
	if len(content) > 32<<10 {
		return Report{}, fmt.Errorf("probe report exceeds 32 KiB")
	}
	if !sameSecret(response.Header.Get(probeSignatureHeader), probeResponseSignature(service.settings.Key, timestamp, nonce, content)) {
		return Report{}, fmt.Errorf("probe response identity does not match the monitor key or request nonce")
	}
	var envelope struct {
		Data Report `json:"data"`
	}
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&envelope); err != nil {
		return Report{}, err
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return Report{}, fmt.Errorf("probe response must contain exactly one JSON object")
	}
	return envelope.Data, nil
}

func (service *Service) acceptPeer(address string, report Report) error {
	service.mutex.Lock()
	defer service.mutex.Unlock()
	if report.Node.ID == service.settings.NodeID {
		return fmt.Errorf("a remote probe cannot replace the master's local identity")
	}
	if peer := service.peers[address]; peer.ID != "" && peer.ID != report.Node.ID {
		return fmt.Errorf("probe identity changed; verify the server and restart the master to trust the replacement")
	}
	for other, peer := range service.peers {
		if other != address && peer.ID == report.Node.ID {
			return fmt.Errorf("multiple peer IPs report the same node_id; keep each probe's generated identity unique")
		}
	}
	now := time.Now()
	if err := service.registry.Accept(report, now); err != nil {
		return err
	}
	view, exists := service.registry.Node(report.Node.ID, now)
	if exists {
		service.peers[address] = candidateHealth{ID: report.Node.ID, Name: report.Node.Name, LastSeen: view.LastSeen}
	}
	return nil
}
