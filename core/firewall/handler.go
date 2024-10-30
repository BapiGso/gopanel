// core/firewall/handler.go
//go:build linux
// +build linux

package firewall

import (
	"encoding/binary"
	"fmt"
	"github.com/google/nftables"
	"github.com/google/nftables/expr"
	"github.com/labstack/echo/v4"
	"math/rand"
	"net/http"
	"time"
)

// 生成随机名称
func generateUniqueName(prefix string, length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	seededRand := rand.New(rand.NewSource(time.Now().UnixNano()))

	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return fmt.Sprintf("%s_%s", prefix, string(b))
}

func Index(c echo.Context) error {
	conn, err := nftables.New()
	if err != nil {
		return err
	}
	tables, err := conn.ListTables()
	if err != nil {
		return err
	}
	chains, err := conn.ListChains()
	if err != nil {
		return err
	}
	req := &struct {
		Network   uint   `form:"network"`
		Transport uint   `form:"transport"`
		Verdict   uint   `form:"verdict"`
		ChainHook uint32 `form:"chainhook"`
		Port      uint   `form:"port"`
		Handle    uint64 `query:"handle"`
		TableName string `query:"tablename"`
		ChainName string `query:"chainname"`
	}{}
	if err := c.Bind(req); err != nil {
		return err
	}

	switch c.Request().Method {
	case "POST":
		// 生成唯一的表名和链名
		tableName := generateUniqueName("table", 8) // 例如: table_a1b2c3d4
		chainName := generateUniqueName("chain", 8) // 例如: chain_e5f6g7h8

		// 创建新表
		table := &nftables.Table{
			Name:   tableName,
			Family: nftables.TableFamily(req.Network),
		}
		table = conn.AddTable(table)

		// 创建新链
		chain := &nftables.Chain{
			Name:     chainName,
			Table:    table,
			Hooknum:  nftables.ChainHookRef(nftables.ChainHook(req.ChainHook)),
			Priority: nftables.ChainPriorityFilter,
			Type:     nftables.ChainTypeFilter,
		}
		chain = conn.AddChain(chain)

		// 创建规则表达式
		exprs := []expr.Any{
			// 匹配协议
			&expr.Meta{
				Key:      expr.MetaKeyL4PROTO,
				Register: 1,
			},
			&expr.Cmp{
				Op:       expr.CmpOpEq,
				Register: 1,
				Data:     []byte{byte(req.Transport)},
			},
			// 匹配目标端口
			&expr.Payload{
				DestRegister: 1,
				Base:         expr.PayloadBaseTransportHeader,
				Offset:       2,
				Len:          2,
			},
			&expr.Cmp{
				Op:       expr.CmpOpEq,
				Register: 1,
				Data:     []byte{byte(req.Port >> 8), byte(req.Port)},
			},
			// 设置动作为接受或拒绝
			&expr.Verdict{
				Kind: expr.VerdictKind(req.Verdict),
			},
		}

		// 添加规则
		rule := &nftables.Rule{
			Table: table,
			Chain: chain,
			Exprs: exprs,
		}
		conn.AddRule(rule)

		if err := conn.Flush(); err != nil {
			return err
		}

		// 返回创建的表名和链名，方便后续管理
		return c.JSON(200, map[string]string{
			"status":     "success",
			"table_name": tableName,
			"chain_name": chainName,
		})

	case "DELETE":
		for _, table := range tables {
			for _, chain := range chains {
				rules, err := conn.GetRules(table, chain)
				if err != nil {
					continue
				}
				for _, rule := range rules {
					if rule.Table.Name == req.TableName && rule.Chain.Name == req.ChainName && rule.Handle == req.Handle {
						if err := conn.DelRule(rule); err != nil {
							return err
						}
						// 删除规则后，检查链中是否还有其他规则
						remainingRules, _ := conn.GetRules(table, chain)
						if len(remainingRules) == 0 {
							// 如果链为空，删除链
							conn.DelChain(chain)
							// 检查表中是否还有其他链
							tableChains, _ := conn.ListChainsOfTableFamily(table.Family)
							hasOtherChains := false
							for _, ch := range tableChains {
								if ch.Table.Name == table.Name && ch.Name != chain.Name {
									hasOtherChains = true
									break
								}
							}
							if !hasOtherChains {
								// 如果表中没有其他链，删除表
								conn.DelTable(table)
							}
						}
						if err := conn.Flush(); err != nil {
							return err
						}
						return c.JSON(200, "success")
					}
				}
			}
		}
		return c.JSON(404, "rule not found")

	case "GET":
		var rulesInfos []RuleInfo
		for _, table := range tables {
			for _, chain := range chains {
				rules, err := conn.GetRules(table, chain)
				if err != nil {
					continue
				}
				for _, rule := range rules {
					rulesInfos = append(rulesInfos, parseRule(rule))
				}
			}
		}
		return c.Render(http.StatusOK, "firewall.template", rulesInfos)
	}
	return echo.ErrMethodNotAllowed
}

type RuleInfo struct {
	nftables.Rule
	Protocol byte
	Port     uint16
	Verdict  uint16
	Hook     uint32
}

func parseRule(rule *nftables.Rule) RuleInfo {
	info := RuleInfo{Rule: *rule}

	// Get Hook value
	if rule.Chain != nil && rule.Chain.Hooknum != nil {
		info.Hook = uint32(*rule.Chain.Hooknum)
	}

	// Parse expressions to extract protocol, port and verdict
	for i, ex := range rule.Exprs {
		switch Expr := ex.(type) {
		case *expr.Cmp:
			// First Cmp usually contains protocol info
			if i == 1 && len(Expr.Data) == 1 {
				info.Protocol = Expr.Data[0]
			}
			// Second Cmp usually contains port info
			if i == 3 && len(Expr.Data) == 2 {
				info.Port = binary.BigEndian.Uint16(Expr.Data)
			}
		case *expr.Verdict:
			// Parse verdict information
			info.Verdict = uint16(Expr.Kind)
		}
	}

	return info
}
