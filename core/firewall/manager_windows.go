//go:build windows

package firewall

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

type windowsManager struct{}

// NewManager creates a new Windows Firewall manager using netsh.
func NewManager() (Manager, error) {
	return &windowsManager{}, nil
}

// ruleName builds a deterministic rule name encoding all parameters.
// Format: GoPanelFW-{direction}-{network}-{protocol}-{port}-{action}
func ruleName(rule Rule) string {
	return fmt.Sprintf("GoPanelFW-%s-%s-%s-%d-%s",
		rule.Direction, rule.Network, rule.Protocol, rule.Port, rule.Action)
}

func (m *windowsManager) List() ([]Rule, error) {
	out, err := exec.Command("netsh", "advfirewall", "firewall", "show", "rule", "name=all").Output()
	if err != nil {
		return nil, fmt.Errorf("netsh show rule: %w", err)
	}

	var rules []Rule
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		idx := strings.Index(line, "GoPanelFW-")
		if idx < 0 {
			continue
		}
		name := strings.TrimSpace(line[idx:])
		rule, err := parseWindowsRuleName(name)
		if err != nil {
			continue
		}
		rules = append(rules, rule)
	}
	return rules, nil
}

func (m *windowsManager) Add(rule Rule) error {
	name := ruleName(rule)

	dir := rule.Direction
	action := "allow"
	if rule.Action == "drop" {
		action = "block"
	}

	args := []string{
		"advfirewall", "firewall", "add", "rule",
		"name=" + name,
		"dir=" + dir,
		"action=" + action,
		"protocol=" + rule.Protocol,
	}

	if rule.Port > 0 {
		args = append(args, "localport="+strconv.Itoa(int(rule.Port)))
	}

	if out, err := exec.Command("netsh", args...).CombinedOutput(); err != nil {
		return fmt.Errorf("netsh add rule: %s: %w", strings.TrimSpace(string(out)), err)
	}
	return nil
}

func (m *windowsManager) Delete(id string) error {
	if out, err := exec.Command("netsh", "advfirewall", "firewall", "delete", "rule", "name="+id).CombinedOutput(); err != nil {
		return fmt.Errorf("netsh delete rule: %s: %w", strings.TrimSpace(string(out)), err)
	}
	return nil
}

// parseWindowsRuleName extracts Rule fields from a GoPanelFW-* rule name.
// Expected format: GoPanelFW-{direction}-{network}-{protocol}-{port}-{action}
func parseWindowsRuleName(name string) (Rule, error) {
	name = strings.TrimSpace(name)
	if !strings.HasPrefix(name, "GoPanelFW-") {
		return Rule{}, fmt.Errorf("not a GoPanelFW rule: %s", name)
	}

	rest := strings.TrimPrefix(name, "GoPanelFW-")
	parts := strings.SplitN(rest, "-", 5)
	if len(parts) != 5 {
		return Rule{}, fmt.Errorf("invalid GoPanelFW rule name: %s", name)
	}

	port, err := strconv.ParseUint(parts[3], 10, 16)
	if err != nil {
		return Rule{}, fmt.Errorf("invalid port in rule name: %s", name)
	}

	return Rule{
		ID:        name,
		Direction: parts[0],
		Network:   parts[1],
		Protocol:  parts[2],
		Port:      uint16(port),
		Action:    parts[4],
	}, nil
}
