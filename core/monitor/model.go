package monitor

import "time"

type Node struct {
	ID        string   `json:"id" validate:"required,min=8,max=80"`
	Name      string   `json:"name" validate:"required,max=80"`
	Country   string   `json:"country" validate:"omitempty,iso3166_1_alpha2"`
	Region    string   `json:"region" validate:"max=80"`
	Latitude  *float64 `json:"latitude,omitempty" validate:"omitempty,gte=-90,lte=90"`
	Longitude *float64 `json:"longitude,omitempty" validate:"omitempty,gte=-180,lte=180"`
	Tags      []string `json:"tags" validate:"max=12,dive,min=1,max=32"`
	Price     string   `json:"price,omitempty" validate:"max=80"`
	ExpiresAt string   `json:"expires_at,omitempty" validate:"omitempty,datetime=2006-01-02"`
}

type Metrics struct {
	CPUPercent        float64  `json:"cpu_percent" validate:"gte=0,lte=100"`
	MemoryPercent     float64  `json:"memory_percent" validate:"gte=0,lte=100"`
	MemoryTotal       uint64   `json:"memory_total"`
	MemoryUsed        uint64   `json:"memory_used"`
	DiskPercent       float64  `json:"disk_percent" validate:"gte=0,lte=100"`
	DiskTotal         uint64   `json:"disk_total"`
	DiskUsed          uint64   `json:"disk_used"`
	UploadBytes       uint64   `json:"upload_bytes"`
	DownloadBytes     uint64   `json:"download_bytes"`
	UploadRate        float64  `json:"upload_rate" validate:"gte=0,lte=1000000000000"`
	DownloadRate      float64  `json:"download_rate" validate:"gte=0,lte=1000000000000"`
	UptimeSeconds     uint64   `json:"uptime_seconds"`
	Platform          string   `json:"platform" validate:"max=80"`
	Architecture      string   `json:"architecture" validate:"max=80"`
	CPUCores          int      `json:"cpu_cores" validate:"gte=0,lte=65536"`
	Available         bool     `json:"available"`
	RateAvailable     bool     `json:"rate_available"`
	UnavailableFields []string `json:"unavailable_fields" validate:"max=8,dive,max=40"`
}

type Report struct {
	Node        Node      `json:"node" validate:"required"`
	BootID      string    `json:"boot_id" validate:"required,min=8,max=80"`
	Sequence    uint64    `json:"sequence" validate:"gt=0"`
	CollectedAt time.Time `json:"collected_at"`
	Metrics     Metrics   `json:"metrics"`
}

type NodeView struct {
	Node
	Metrics     Metrics   `json:"metrics"`
	Online      bool      `json:"online"`
	CollectedAt time.Time `json:"collected_at"`
	LastSeen    time.Time `json:"last_seen"`
}

type Sample struct {
	Time          time.Time `json:"time"`
	CPUPercent    float64   `json:"cpu_percent"`
	MemoryPercent float64   `json:"memory_percent"`
	UploadRate    float64   `json:"upload_rate"`
	DownloadRate  float64   `json:"download_rate"`
}

type Overview struct {
	Title         string      `json:"title"`
	UpdatedAt     time.Time   `json:"updated_at"`
	Total         int         `json:"total"`
	Online        int         `json:"online"`
	Offline       int         `json:"offline"`
	Countries     int         `json:"countries"`
	UploadRate    float64     `json:"upload_rate"`
	DownloadRate  float64     `json:"download_rate"`
	UploadBytes   uint64      `json:"upload_bytes"`
	DownloadBytes uint64      `json:"download_bytes"`
	Nodes         []NodeView  `json:"nodes"`
	Cluster       ClusterView `json:"cluster"`
}

type ClusterView struct {
	Role              string `json:"role"`
	Coordination      string `json:"coordination"`
	LeaderName        string `json:"leader_name,omitempty"`
	LeaderID          string `json:"leader_id,omitempty"`
	HealthyCandidates int    `json:"healthy_candidates"`
	CandidateCount    int    `json:"candidate_count"`
	Degraded          bool   `json:"degraded"`
	PeerCount         int    `json:"peer_count,omitempty"`
	HealthyPeers      int    `json:"healthy_peers,omitempty"`
}

func (configuration Config) node() Node {
	return Node{
		ID: configuration.NodeID, Name: configuration.Name, Country: configuration.Country,
		Region: configuration.Region, Latitude: configuration.Latitude, Longitude: configuration.Longitude,
		Tags: append([]string{}, configuration.Tags...), Price: configuration.Price, ExpiresAt: configuration.ExpiresAt,
	}
}

func cloneNode(node Node) Node {
	node.Tags = append([]string{}, node.Tags...)
	if node.Latitude != nil {
		latitude := *node.Latitude
		longitude := *node.Longitude
		node.Latitude = &latitude
		node.Longitude = &longitude
	}
	return node
}
