//go:build darwin

package firewall

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

type darwinManager struct{}

// NewManager creates a new macOS pfctl-based firewall manager using anchor "gopanel".
func NewManager() (Manager, error) {
	return &darwinManager{}, nil
}

func (m *darwinManager) List() ([]Rule, error) {
	out, err := exec.Command("pfctl", "-a", "gopanel", "-sr").CombinedOutput()
	if err != nil {
		// If the anchor doesn't exist yet, return empty list
		if strings.Contains(string(out), "No ALTQ") || len(strings.TrimSpace(string(out))) == 0 {
			return nil, nil
		}
		return nil, fmt.Errorf("pfctl -sr: %s: %w", strings.TrimSpace(string(out)), err)
	}

	var rules []Rule
	for i, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		rule, err := parsePfctlRule(line, i)
		if err != nil {
			continue
		}
		rules = append(rules, rule)
	}
	return rules, nil
}

func (m *darwinManager) Add(rule Rule) error {
	// Read current anchor rules
	existing, _ := m.currentRules()

	// Build the new pf rule line
	pfRule, err := buildPfRule(rule)
	if err != nil {
		return err
	}
	existing = append(existing, pfRule)

	// Reload all rules into the anchor
	return m.loadRules(existing)
}

func (m *darwinManager) Delete(id string) error {
	lineNum, err := strconv.Atoi(id)
	if err != nil {
		return fmt.Errorf("invalid rule id: %s", id)
	}

	existing, err := m.currentRules()
	if err != nil {
		return err
	}

	if lineNum < 0 || lineNum >= len(existing) {
		return fmt.Errorf("rule id out of range: %d", lineNum)
	}

	// Remove the target line
	rules := append(existing[:lineNum], existing[lineNum+1:]...)
	return m.loadRules(rules)
}

// currentRules reads the current anchor rules as raw pf lines.
func (m *darwinManager) currentRules() ([]string, error) {
	out, err := exec.Command("pfctl", "-a", "gopanel", "-sr").CombinedOutput()
	if err != nil {
		return nil, nil // anchor may not exist yet
	}
	var lines []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines, nil
}

// loadRules replaces all rules in the gopanel anchor.
func (m *darwinManager) loadRules(rules []string) error {
	input := strings.Join(rules, "\n") + "\n"
	if len(rules) == 0 {
		input = ""
	}

	cmd := exec.Command("pfctl", "-a", "gopanel", "-f", "-")
	cmd.Stdin = strings.NewReader(input)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("pfctl -f: %s: %w", strings.TrimSpace(string(out)), err)
	}
	return nil
}

// buildPfRule converts a Rule to a pf rule string.
// Example: "block in proto tcp from any to any port 80"
func buildPfRule(rule Rule) (string, error) {
	action := "pass"
	if rule.Action == "drop" {
		action = "block"
	}

	dir := rule.Direction
	proto := rule.Protocol

	line := fmt.Sprintf("%s %s proto %s from any to any", action, dir, proto)
	if rule.Port > 0 {
		line += fmt.Sprintf(" port %d", rule.Port)
	}
	return line, nil
}

// parsePfctlRule parses a pf rule line into a Rule.
// Expected format: "block in proto tcp from any to any port 80"
func parsePfctlRule(line string, lineNum int) (Rule, error) {
	r := Rule{
		ID:      strconv.Itoa(lineNum),
		Network: "ipv4",
	}

	fields := strings.Fields(line)
	if len(fields) < 6 {
		return Rule{}, fmt.Errorf("too few fields in pf rule: %s", line)
	}

	// Parse action
	switch fields[0] {
	case "pass":
		r.Action = "accept"
	case "block":
		r.Action = "drop"
	default:
		return Rule{}, fmt.Errorf("unknown action: %s", fields[0])
	}

	// Parse direction
	switch fields[1] {
	case "in":
		r.Direction = "in"
	case "out":
		r.Direction = "out"
	default:
		return Rule{}, fmt.Errorf("unknown direction: %s", fields[1])
	}

	// Parse protocol (expect "proto" keyword at index 2)
	if fields[2] == "proto" && len(fields) > 3 {
		r.Protocol = fields[3]
	}

	// Parse port (look for "port" keyword)
	for i, f := range fields {
		if f == "port" && i+1 < len(fields) {
			port, err := strconv.ParseUint(fields[i+1], 10, 16)
			if err == nil {
				r.Port = uint16(port)
			}
			break
		}
	}

	// Detect IPv6 from "inet6" keyword
	for _, f := range fields {
		if f == "inet6" {
			r.Network = "ipv6"
			break
		}
	}

	return r, nil
}
