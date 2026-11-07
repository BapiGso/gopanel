package monitor

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"os"
	"sync"
	"time"
)

type Collector interface {
	Collect(context.Context) (Metrics, error)
	Close(context.Context) error
}

type candidateHealth struct {
	ID       string
	Name     string
	LastSeen time.Time
}

type peerReply struct {
	ClusterID string `json:"cluster_id"`
	ID        string `json:"id"`
	Name      string `json:"name"`
}

type Service struct {
	settings    Config
	collector   Collector
	registry    *Registry
	client      *http.Client
	logger      *slog.Logger
	bootID      string
	mutex       sync.RWMutex
	latest      Report
	peers       map[string]candidateHealth
	started     bool
	closed      bool
	cancel      context.CancelFunc
	done        chan struct{}
	streams     chan struct{}
	probeServer *http.Server
	peerErrors  map[string]string
}

func NewService(settings Config, collector Collector, logger *slog.Logger) (*Service, error) {
	if err := settings.Validate(); err != nil {
		return nil, err
	}
	settings.Tags = append([]string{}, settings.Tags...)
	settings.Candidates = append([]string{}, settings.Candidates...)
	settings.Interfaces = append([]string{}, settings.Interfaces...)
	settings.Peers = append([]string{}, settings.Peers...)
	for index, peer := range settings.Peers {
		settings.Peers[index], _ = peerAddress(peer)
	}
	if settings.Latitude != nil {
		latitude, longitude := *settings.Latitude, *settings.Longitude
		settings.Latitude, settings.Longitude = &latitude, &longitude
	}
	if logger == nil {
		logger = slog.Default()
	}
	if collector == nil {
		var err error
		collector, err = NewCollector(settings.Interfaces)
		if err != nil {
			return nil, err
		}
	}
	tlsConfig := &tls.Config{MinVersion: tls.VersionTLS12}
	if settings.CAFile != "" {
		content, err := os.ReadFile(settings.CAFile)
		if err != nil {
			collector.Close(context.Background())
			return nil, fmt.Errorf("read monitor CA certificate: %w", err)
		}
		roots, err := x509.SystemCertPool()
		if err != nil {
			roots = x509.NewCertPool()
		}
		if !roots.AppendCertsFromPEM(content) {
			collector.Close(context.Background())
			return nil, fmt.Errorf("monitor ca_file does not contain PEM certificates")
		}
		tlsConfig.RootCAs = roots
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = tlsConfig
	transport.MaxIdleConnsPerHost = 2
	transport.ResponseHeaderTimeout = 2 * time.Second
	service := &Service{
		settings: settings, collector: collector,
		registry: NewRegistry(time.Duration(settings.OfflineSeconds)*time.Second, settings.HistorySamples, settings.MaximumNodes),
		client:   &http.Client{Transport: transport, Timeout: 3 * time.Second, CheckRedirect: func(request *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }},
		logger:   logger, bootID: rand.Text(), peers: map[string]candidateHealth{}, done: make(chan struct{}), streams: make(chan struct{}, 64), peerErrors: map[string]string{},
	}
	if settings.Role == "candidate" {
		service.peers[settings.AdvertiseURL] = candidateHealth{ID: settings.NodeID, Name: settings.Name, LastSeen: time.Now()}
	}
	return service, nil
}

func (service *Service) Start(ctx context.Context) error {
	service.mutex.Lock()
	if service.started || service.closed {
		service.mutex.Unlock()
		return nil
	}
	runContext, cancel := context.WithCancel(ctx)
	service.cancel = cancel
	if err := service.startProbeServer(); err != nil {
		cancel()
		service.mutex.Unlock()
		return err
	}
	service.started = true
	service.mutex.Unlock()
	go service.run(runContext)
	return nil
}

func (service *Service) Wait() {
	<-service.done
}

func (service *Service) Close(ctx context.Context) error {
	service.mutex.Lock()
	cancel := service.cancel
	if !service.started && !service.closed {
		service.closed = true
		service.mutex.Unlock()
		service.client.CloseIdleConnections()
		err := service.collector.Close(ctx)
		close(service.done)
		return err
	}
	service.mutex.Unlock()
	if cancel != nil {
		cancel()
	}
	select {
	case <-service.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (service *Service) run(ctx context.Context) {
	defer close(service.done)
	defer service.client.CloseIdleConnections()
	defer service.collector.Close(context.Background())
	defer service.stopProbeServer()
	var reporters sync.WaitGroup
	if service.settings.Role == "master" {
		reporters.Go(func() { service.pullLoop(ctx) })
	} else if len(service.settings.Candidates) > 0 && service.settings.Role != "standalone" {
		reporters.Go(func() { service.reportLoop(ctx) })
	}
	ticker := time.NewTicker(time.Duration(service.settings.SampleSeconds) * time.Second)
	defer ticker.Stop()
	for {
		service.sample(ctx)
		select {
		case <-ctx.Done():
			reporters.Wait()
			return
		case <-ticker.C:
		}
	}
}

func (service *Service) sample(ctx context.Context) {
	metrics, err := service.collector.Collect(ctx)
	if ctx.Err() != nil {
		return
	}
	if err != nil {
		service.logger.Warn("monitor collection incomplete", "error", err)
	}
	service.mutex.Lock()
	report := Report{
		Node: service.settings.node(), BootID: service.bootID, Sequence: service.latest.Sequence + 1,
		CollectedAt: time.Now().UTC(), Metrics: metrics,
	}
	service.latest = report
	service.mutex.Unlock()
	if err := service.registry.Accept(report, time.Now()); err != nil {
		service.logger.Error("local monitor report rejected", "error", err)
	}
}

func (service *Service) reportLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(service.settings.SampleSeconds) * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			service.mutex.RLock()
			report := service.latest
			service.mutex.RUnlock()
			if report.Sequence != 0 {
				service.sendReports(ctx, report)
			}
		}
	}
}

func (service *Service) sendReports(ctx context.Context, report Report) {
	content, err := json.Marshal(report)
	if err != nil {
		service.logger.Error("encode monitor report", "error", err)
		return
	}
	var deliveries sync.WaitGroup
	for _, address := range service.settings.Candidates {
		if service.settings.Role == "candidate" && address == service.settings.AdvertiseURL {
			continue
		}
		deliveries.Go(func() {
			peer, err := service.send(ctx, address, content)
			if err != nil {
				return
			}
			service.mutex.Lock()
			service.peers[address] = candidateHealth{ID: peer.ID, Name: peer.Name, LastSeen: time.Now()}
			service.mutex.Unlock()
		})
	}
	deliveries.Wait()
}

func (service *Service) send(ctx context.Context, address string, content []byte) (peerReply, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, address+"/api/v1/monitor/reports", bytes.NewReader(content))
	if err != nil {
		return peerReply{}, err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+service.settings.ClusterKey)
	request.Header.Set("X-GoPanel-Cluster", service.settings.ClusterID)
	response, err := service.client.Do(request)
	if err != nil {
		return peerReply{}, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return peerReply{}, fmt.Errorf("candidate rejected report: HTTP %d", response.StatusCode)
	}
	var envelope struct {
		Data peerReply `json:"data"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 16<<10)).Decode(&envelope); err != nil {
		return peerReply{}, err
	}
	if envelope.Data.ClusterID != service.settings.ClusterID || len(envelope.Data.ID) < 8 || len(envelope.Data.ID) > 80 || len(envelope.Data.Name) > 80 {
		return peerReply{}, fmt.Errorf("candidate identity does not match the configured cluster")
	}
	return envelope.Data, nil
}

func (service *Service) Cluster(now time.Time) ClusterView {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	view := ClusterView{Role: service.settings.Role, Coordination: "best_effort", CandidateCount: len(service.settings.Candidates)}
	if service.settings.Role == "master" {
		view.Coordination, view.LeaderID, view.LeaderName = "manual", service.settings.NodeID, service.settings.Name
		view.CandidateCount, view.HealthyCandidates, view.PeerCount = 1, 1, len(service.settings.Peers)
		for _, address := range service.settings.Peers {
			peer := service.peers[address]
			if !peer.LastSeen.IsZero() && now.Sub(peer.LastSeen) <= time.Duration(service.settings.OfflineSeconds)*time.Second {
				view.HealthyPeers++
			}
		}
		view.Degraded = view.HealthyPeers < view.PeerCount
		return view
	}
	if service.settings.Role == "probe" && len(service.settings.Candidates) == 0 {
		view.Coordination = "probe"
		return view
	}
	if service.settings.Role == "standalone" {
		view.Coordination = "standalone"
		view.LeaderID = service.settings.NodeID
		view.LeaderName = service.settings.Name
		return view
	}
	for _, address := range service.settings.Candidates {
		peer := service.peers[address]
		healthy := !peer.LastSeen.IsZero() && now.Sub(peer.LastSeen) <= time.Duration(service.settings.OfflineSeconds)*time.Second
		if service.settings.Role == "candidate" && address == service.settings.AdvertiseURL {
			healthy = true
		}
		if healthy {
			view.HealthyCandidates++
			if view.LeaderID == "" {
				view.LeaderID = peer.ID
				view.LeaderName = peer.Name
			}
		}
	}
	view.Degraded = view.HealthyCandidates < view.CandidateCount
	return view
}

func (service *Service) Overview() Overview {
	now := time.Now()
	nodes := service.registry.Views(now)
	view := Overview{Title: service.settings.Title, UpdatedAt: now.UTC(), Total: len(nodes), Nodes: nodes, Cluster: service.Cluster(now)}
	countries := map[string]bool{}
	for _, node := range nodes {
		if node.Country != "" {
			countries[node.Country] = true
		}
		if node.Online {
			view.Online++
			if node.Metrics.RateAvailable {
				view.UploadRate += node.Metrics.UploadRate
				view.DownloadRate += node.Metrics.DownloadRate
			}
		}
		view.UploadBytes = saturatingAdd(view.UploadBytes, node.Metrics.UploadBytes)
		view.DownloadBytes = saturatingAdd(view.DownloadBytes, node.Metrics.DownloadBytes)
	}
	view.Countries = len(countries)
	view.Offline = view.Total - view.Online
	return view
}

func saturatingAdd(first, second uint64) uint64 {
	if math.MaxUint64-first < second {
		return math.MaxUint64
	}
	return first + second
}
