package firewall

// Rule represents a cross-platform firewall rule.
type Rule struct {
	ID        string // unique identifier (linux: family:chain:handle, windows: rule name, darwin: line number)
	Protocol  string // "tcp" / "udp"
	Port      uint16 // 0 = any
	Direction string // "in" / "out"
	Action    string // "accept" / "drop"
	Network   string // "ipv4" / "ipv6"
}

// Manager is the interface for managing firewall rules across platforms.
type Manager interface {
	List() ([]Rule, error)
	Add(rule Rule) error
	Delete(id string) error
}
