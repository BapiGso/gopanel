//go:build linux

package firewall

import (
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/nftables"
	"github.com/google/nftables/expr"
	"golang.org/x/sys/unix"
)

const nftTableName = "gopanel"

type nftManager struct{}

// NewManager creates a new nftables-based firewall manager.
func NewManager() (Manager, error) {
	return &nftManager{}, nil
}

// ensureChains creates the gopanel table and input/output chains if they don't exist.
// nftables AddTable/AddChain are idempotent — safe to call on every operation.
func (m *nftManager) ensureChains(conn *nftables.Conn, family nftables.TableFamily) (*nftables.Table, map[string]*nftables.Chain) {
	table := conn.AddTable(&nftables.Table{
		Name:   nftTableName,
		Family: family,
	})
	chains := map[string]*nftables.Chain{
		"gopanel-input": conn.AddChain(&nftables.Chain{
			Name:     "gopanel-input",
			Table:    table,
			Hooknum:  nftables.ChainHookInput,
			Priority: nftables.ChainPriorityFilter,
			Type:     nftables.ChainTypeFilter,
		}),
		"gopanel-output": conn.AddChain(&nftables.Chain{
			Name:     "gopanel-output",
			Table:    table,
			Hooknum:  nftables.ChainHookOutput,
			Priority: nftables.ChainPriorityFilter,
			Type:     nftables.ChainTypeFilter,
		}),
	}
	return table, chains
}

func (m *nftManager) List() ([]Rule, error) {
	conn, err := nftables.New()
	if err != nil {
		return nil, err
	}

	families := []nftables.TableFamily{nftables.TableFamilyIPv4, nftables.TableFamilyIPv6}
	for _, f := range families {
		m.ensureChains(conn, f)
	}
	if err := conn.Flush(); err != nil {
		return nil, err
	}

	var rules []Rule
	chainNames := []string{"gopanel-input", "gopanel-output"}
	for _, family := range families {
		table := &nftables.Table{Name: nftTableName, Family: family}
		for _, chainName := range chainNames {
			chain := &nftables.Chain{Name: chainName, Table: table}
			nfRules, err := conn.GetRules(table, chain)
			if err != nil {
				continue
			}
			for _, r := range nfRules {
				parsed := parseNftRule(r)
				if parsed.Protocol != "" {
					rules = append(rules, parsed)
				}
			}
		}
	}
	return rules, nil
}

func (m *nftManager) Add(rule Rule) error {
	conn, err := nftables.New()
	if err != nil {
		return err
	}

	var family nftables.TableFamily
	switch rule.Network {
	case "ipv4":
		family = nftables.TableFamilyIPv4
	case "ipv6":
		family = nftables.TableFamilyIPv6
	default:
		return fmt.Errorf("unsupported network: %s", rule.Network)
	}

	var chainName string
	switch rule.Direction {
	case "in":
		chainName = "gopanel-input"
	case "out":
		chainName = "gopanel-output"
	default:
		return fmt.Errorf("unsupported direction: %s", rule.Direction)
	}

	table, chains := m.ensureChains(conn, family)
	chain := chains[chainName]

	var proto byte
	switch rule.Protocol {
	case "tcp":
		proto = unix.IPPROTO_TCP
	case "udp":
		proto = unix.IPPROTO_UDP
	default:
		return fmt.Errorf("unsupported protocol: %s", rule.Protocol)
	}

	var verdict expr.VerdictKind
	switch rule.Action {
	case "accept":
		verdict = expr.VerdictAccept
	case "drop":
		verdict = expr.VerdictDrop
	default:
		return fmt.Errorf("unsupported action: %s", rule.Action)
	}

	exprs := []expr.Any{
		&expr.Meta{Key: expr.MetaKeyL4PROTO, Register: 1},
		&expr.Cmp{Op: expr.CmpOpEq, Register: 1, Data: []byte{proto}},
	}

	if rule.Port > 0 {
		portBytes := make([]byte, 2)
		binary.BigEndian.PutUint16(portBytes, rule.Port)
		exprs = append(exprs,
			&expr.Payload{
				DestRegister: 1,
				Base:         expr.PayloadBaseTransportHeader,
				Offset:       2,
				Len:          2,
			},
			&expr.Cmp{Op: expr.CmpOpEq, Register: 1, Data: portBytes},
		)
	}

	exprs = append(exprs, &expr.Verdict{Kind: verdict})

	conn.AddRule(&nftables.Rule{
		Table: table,
		Chain: chain,
		Exprs: exprs,
	})

	return conn.Flush()
}

func (m *nftManager) Delete(id string) error {
	// ID format: family:chain:handle
	parts := strings.SplitN(id, ":", 3)
	if len(parts) != 3 {
		return fmt.Errorf("invalid rule id: %s", id)
	}

	familyNum, err := strconv.Atoi(parts[0])
	if err != nil {
		return fmt.Errorf("invalid family in id: %s", id)
	}
	chainName := parts[1]
	handle, err := strconv.ParseUint(parts[2], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid handle in id: %s", id)
	}

	conn, err := nftables.New()
	if err != nil {
		return err
	}

	table := &nftables.Table{Name: nftTableName, Family: nftables.TableFamily(familyNum)}
	chain := &nftables.Chain{Name: chainName, Table: table}

	if err := conn.DelRule(&nftables.Rule{
		Table:  table,
		Chain:  chain,
		Handle: handle,
	}); err != nil {
		return err
	}

	return conn.Flush()
}

// parseNftRule extracts Rule fields from an nftables rule by tracking expression
// types rather than relying on fixed indices.
func parseNftRule(rule *nftables.Rule) Rule {
	r := Rule{}

	switch rule.Table.Family {
	case nftables.TableFamilyIPv4:
		r.Network = "ipv4"
	case nftables.TableFamilyIPv6:
		r.Network = "ipv6"
	}

	if rule.Chain != nil {
		switch rule.Chain.Name {
		case "gopanel-input":
			r.Direction = "in"
		case "gopanel-output":
			r.Direction = "out"
		}
	}

	r.ID = fmt.Sprintf("%d:%s:%d", rule.Table.Family, rule.Chain.Name, rule.Handle)

	// Track previous expression type to determine what the next Cmp refers to
	var lastMeta expr.MetaKey
	var lastIsPort bool

	for _, ex := range rule.Exprs {
		switch e := ex.(type) {
		case *expr.Meta:
			lastMeta = e.Key
			lastIsPort = false
		case *expr.Payload:
			if e.Base == expr.PayloadBaseTransportHeader && e.Offset == 2 && e.Len == 2 {
				lastIsPort = true
			}
			lastMeta = 0
		case *expr.Cmp:
			if lastIsPort && len(e.Data) == 2 {
				r.Port = binary.BigEndian.Uint16(e.Data)
			} else if lastMeta == expr.MetaKeyL4PROTO && len(e.Data) == 1 {
				switch e.Data[0] {
				case unix.IPPROTO_TCP:
					r.Protocol = "tcp"
				case unix.IPPROTO_UDP:
					r.Protocol = "udp"
				}
			}
			lastIsPort = false
			lastMeta = 0
		case *expr.Verdict:
			switch e.Kind {
			case expr.VerdictAccept:
				r.Action = "accept"
			case expr.VerdictDrop:
				r.Action = "drop"
			}
		}
	}

	return r
}
