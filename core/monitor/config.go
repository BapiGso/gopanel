package monitor

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/netip"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"

	"github.com/go-playground/validator/v10"

	"gopanel/core/storage"
)

const ConfigFileName = "gopanel_monitor.json"
const DefaultProbePort = "9555"

type Config struct {
	SchemaVersion  int      `json:"schema_version" validate:"eq=1"`
	Role           string   `json:"role" validate:"oneof=standalone master candidate probe"`
	NodeID         string   `json:"node_id" validate:"required,min=8,max=80"`
	Name           string   `json:"name" validate:"required,max=80"`
	Title          string   `json:"title" validate:"required,max=100"`
	Country        string   `json:"country" validate:"omitempty,iso3166_1_alpha2"`
	Region         string   `json:"region" validate:"max=80"`
	Latitude       *float64 `json:"latitude,omitempty" validate:"omitempty,gte=-90,lte=90"`
	Longitude      *float64 `json:"longitude,omitempty" validate:"omitempty,gte=-180,lte=180"`
	Tags           []string `json:"tags" validate:"max=12,dive,min=1,max=32"`
	Price          string   `json:"price,omitempty" validate:"max=80"`
	ExpiresAt      string   `json:"expires_at,omitempty" validate:"omitempty,datetime=2006-01-02"`
	Public         bool     `json:"public"`
	SampleSeconds  int      `json:"sample_seconds" validate:"min=1,max=60"`
	OfflineSeconds int      `json:"offline_seconds" validate:"min=5,max=300"`
	Candidates     []string `json:"candidates" validate:"max=8,dive,required,max=300"`
	AdvertiseURL   string   `json:"advertise_url,omitempty" validate:"max=300"`
	ClusterID      string   `json:"cluster_id,omitempty" validate:"max=80"`
	ClusterKey     string   `json:"cluster_key,omitempty" validate:"omitempty,min=32,max=200"`
	CAFile         string   `json:"ca_file,omitempty" validate:"max=500"`
	AllowHTTP      bool     `json:"allow_http"`
	Interfaces     []string `json:"interfaces" validate:"max=32,dive,min=1,max=80"`
	HistorySamples int      `json:"history_samples" validate:"min=10,max=1800"`
	MaximumNodes   int      `json:"maximum_nodes" validate:"min=1,max=2000"`
	Peers          []string `json:"peers,omitempty" validate:"max=500,dive,required,max=300"`
	Key            string   `json:"key,omitempty" validate:"omitempty,min=32,max=200"`
	Listen         string   `json:"listen,omitempty" validate:"max=200"`
}

func DefaultConfig() Config {
	return Config{
		SchemaVersion: 1, Role: "standalone", NodeID: rand.Text(),
		Name: "Local node", Title: "GoPanel Observatory", Public: true,
		SampleSeconds: 2, OfflineSeconds: 15, HistorySamples: 300, MaximumNodes: 500,
		Tags: []string{}, Candidates: []string{}, Interfaces: []string{},
	}
}

func LoadConfig(directory string) (Config, error) {
	if directory == "" {
		return Config{}, fmt.Errorf("monitor configuration directory is required")
	}
	file := &storage.File{Path: filepath.Join(directory, ConfigFileName)}
	initial, err := json.MarshalIndent(map[string]string{"node_id": rand.Text()}, "", "  ")
	if err != nil {
		return Config{}, err
	}
	if err := file.Ensure(append(initial, '\n')); err != nil {
		return Config{}, err
	}
	for range 5 {
		document, err := file.Read()
		if err != nil {
			return Config{}, err
		}
		configuration, err := parseConfig([]byte(document.Content))
		if err != nil {
			return Config{}, fmt.Errorf("read %s: %w", file.Path, err)
		}
		var fields map[string]json.RawMessage
		if err := json.Unmarshal([]byte(document.Content), &fields); err != nil {
			return Config{}, err
		}
		if _, exists := fields["node_id"]; !exists {
			fields["node_id"], err = json.Marshal(configuration.NodeID)
			if err != nil {
				return Config{}, err
			}
			content, err := json.MarshalIndent(fields, "", "  ")
			if err != nil {
				return Config{}, err
			}
			if _, err := file.Save(string(append(content, '\n')), document.Revision); err != nil {
				if errors.Is(err, storage.ErrRevisionConflict) {
					continue
				}
				return Config{}, err
			}
		}
		if configuration.CAFile != "" && !filepath.IsAbs(configuration.CAFile) {
			configuration.CAFile = filepath.Join(directory, configuration.CAFile)
		}
		return configuration, nil
	}
	return Config{}, storage.ErrRevisionConflict
}

func parseConfig(content []byte) (Config, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(content, &fields); err != nil || fields == nil {
		return Config{}, fmt.Errorf("monitor configuration must be a JSON object")
	}
	for name, value := range fields {
		if name != "latitude" && name != "longitude" && bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return Config{}, fmt.Errorf("monitor configuration field %s cannot be null", name)
		}
	}
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	configuration := DefaultConfig()
	if err := decoder.Decode(&configuration); err != nil {
		return Config{}, fmt.Errorf("invalid monitor configuration: %w", err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return Config{}, fmt.Errorf("monitor configuration must contain exactly one JSON object")
	}
	if _, exists := fields["role"]; !exists {
		if len(configuration.Peers) > 0 {
			configuration.Role = "master"
		} else if configuration.Key != "" {
			configuration.Role = "probe"
		}
	}
	if _, exists := fields["name"]; !exists && len(configuration.NodeID) >= 8 {
		configuration.Name = "Node-" + configuration.NodeID[:8]
	}
	if err := configuration.Validate(); err != nil {
		return Config{}, err
	}
	return configuration, nil
}

func (configuration Config) Validate() error {
	if err := validator.New().Struct(configuration); err != nil {
		return fmt.Errorf("invalid monitor configuration: %w", err)
	}
	if (configuration.Latitude == nil) != (configuration.Longitude == nil) {
		return fmt.Errorf("latitude and longitude must be supplied together")
	}
	if configuration.OfflineSeconds < configuration.SampleSeconds*2 {
		return fmt.Errorf("offline_seconds must be at least twice sample_seconds")
	}
	if strings.ContainsAny(configuration.NodeID, "/\\?#% \t\r\n") {
		return fmt.Errorf("node_id must be a stable URL-safe identifier")
	}
	if strings.IndexFunc(configuration.ClusterID, unicode.IsControl) >= 0 || strings.IndexFunc(configuration.ClusterKey, unicode.IsControl) >= 0 || strings.IndexFunc(configuration.Key, unicode.IsControl) >= 0 {
		return fmt.Errorf("monitor identities and keys cannot contain control characters")
	}
	if (configuration.Role == "candidate" || configuration.Role == "probe" && len(configuration.Candidates) > 0) && (configuration.ClusterID == "" || configuration.ClusterKey == "" || len(configuration.Candidates) == 0) {
		return fmt.Errorf("candidate and probe roles require cluster_id, cluster_key and candidates")
	}
	if configuration.Role == "master" && (configuration.Key == "" || len(configuration.Peers) == 0) {
		return fmt.Errorf("master monitoring requires peers and a shared monitor key")
	}
	if configuration.Role == "probe" && len(configuration.Candidates) == 0 && configuration.Key == "" {
		return fmt.Errorf("a read-only probe requires a shared monitor key")
	}
	if len(configuration.Peers) > 0 && configuration.Role != "master" {
		return fmt.Errorf("only the manually selected master can configure peers")
	}
	if len(configuration.Peers)+1 > configuration.MaximumNodes {
		return fmt.Errorf("maximum_nodes must include every configured peer and the local node")
	}
	if configuration.Key != "" && (len(configuration.Candidates) > 0 || configuration.ClusterKey != "" || configuration.ClusterID != "" || configuration.AdvertiseURL != "") {
		return fmt.Errorf("manual peers/key monitoring cannot be mixed with legacy candidate configuration")
	}
	if configuration.Listen != "" {
		_, port, err := net.SplitHostPort(configuration.Listen)
		number, parseError := strconv.Atoi(port)
		if err != nil || parseError != nil || number < 1 || number > 65535 || configuration.Key == "" || configuration.Role == "master" {
			return fmt.Errorf("listen must be a host:port address for a keyed read-only probe")
		}
	}
	peerAddresses := map[string]bool{}
	for _, peer := range configuration.Peers {
		address, err := peerAddress(peer)
		if err != nil {
			return err
		}
		if peerAddresses[address] {
			return fmt.Errorf("peer IPs must not repeat")
		}
		peerAddresses[address] = true
	}
	seen := map[string]bool{}
	for _, address := range configuration.Candidates {
		if err := validateAddress(address, configuration.AllowHTTP); err != nil {
			return err
		}
		if seen[address] {
			return fmt.Errorf("candidate URLs must not repeat")
		}
		seen[address] = true
	}
	if configuration.Role == "candidate" {
		if !seen[configuration.AdvertiseURL] {
			return fmt.Errorf("advertise_url must appear in the ordered candidates list")
		}
	}
	return nil
}

func peerAddress(peer string) (string, error) {
	if address, err := netip.ParseAddr(strings.TrimPrefix(strings.TrimSuffix(peer, "]"), "[")); err == nil && address.Zone() == "" {
		return "http://" + net.JoinHostPort(address.String(), DefaultProbePort), nil
	}
	if !strings.Contains(peer, "://") {
		host, port, err := net.SplitHostPort(peer)
		address, addressError := netip.ParseAddr(host)
		number, portError := strconv.Atoi(port)
		if err != nil || addressError != nil || address.Zone() != "" || portError != nil || number < 1 || number > 65535 {
			return "", fmt.Errorf("peers must be IPs, IP:port addresses or plain HTTP(S) origins")
		}
		return "http://" + net.JoinHostPort(address.String(), strconv.Itoa(number)), nil
	}
	if err := validateAddress(peer, true); err != nil {
		return "", err
	}
	parsed, _ := url.Parse(peer)
	if port := parsed.Port(); port != "" {
		number, err := strconv.Atoi(port)
		if err != nil || number < 1 || number > 65535 {
			return "", fmt.Errorf("peer port must be between 1 and 65535")
		}
	}
	return peer, nil
}

func (configuration Config) probeListenAddress() string {
	if configuration.Listen != "" {
		return configuration.Listen
	}
	return ":" + DefaultProbePort
}

func validateAddress(address string, allowHTTP bool) error {
	parsed, err := url.Parse(address)
	if err != nil || parsed.Hostname() == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.Path != "" {
		return fmt.Errorf("candidate URLs must be plain HTTP(S) origins without paths or credentials")
	}
	if parsed.Scheme != "https" && parsed.Scheme != "http" {
		return fmt.Errorf("candidate URLs must use HTTP or HTTPS")
	}
	if parsed.Scheme == "http" && !allowHTTP {
		address := net.ParseIP(parsed.Hostname())
		if address == nil || !address.IsLoopback() {
			return fmt.Errorf("HTTP candidates require explicit allow_http; use HTTPS outside local testing")
		}
	}
	return nil
}
