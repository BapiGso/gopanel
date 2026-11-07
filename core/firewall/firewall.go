package firewall

// Rule represents a cross-platform firewall rule.
type Rule struct {
	ID        string `json:"id"`
	Protocol  string `json:"protocol"`
	Port      uint16 `json:"port"`
	Direction string `json:"direction"`
	Action    string `json:"action"`
	Network   string `json:"network"`
}

// Manager is the interface for managing firewall rules across platforms.
type Manager interface {
	List() ([]Rule, error)
	Add(rule Rule) error
	Delete(id string) error
}
