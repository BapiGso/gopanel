// core/firewall/handler.go
//go:build linux
// +build linux

package firewall

import (
	"encoding/binary"
	"github.com/google/nftables"
	"github.com/google/nftables/expr"
	"github.com/labstack/echo/v4"
	"net/http"
)

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
		Network   uint   `form:"network"        json:"network"`
		Transport uint   `form:"transport"      json:"transport"`
		Verdict   uint   `form:"verdict"        json:"verdict"`
		ChainHook uint32 `form:"chainhook"      json:"chainhook"`
		Port      uint   `form:"port"           json:"port"`
		Handle    uint64 `query:"handle"        json:"handle"`
		TableName string `query:"tablename"        json:"tablename"`
		ChainName string `query:"chainname"        json:"chainnname"`
	}{}
	if err := c.Bind(req); err != nil {
		return err
	}
	switch c.Request().Method {
	case "POST":
		table := conn.AddTable(&nftables.Table{
			Name:   "gotable",
			Family: nftables.TableFamily(req.Network),
		})
		chain := conn.AddChain(&nftables.Chain{
			Name:     "gochain",
			Table:    table,
			Hooknum:  nftables.ChainHookRef(nftables.ChainHook(req.ChainHook)), //当数据包首次进入网络栈时触发，此时还未进行任何路由决策。
			Priority: nftables.ChainPriorityFilter,                             //过滤链的优先级，值为 0，表示默认优先级。
			Type:     nftables.ChainTypeFilter,                                 //过滤链，用于过滤数据包，决定是否允许数据包通过。
		})
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
				Offset:       2, // 目标端口在传输层头部的偏移量
				Len:          2, // 端口号长度为2字节
			},
			&expr.Cmp{
				Op:       expr.CmpOpEq,
				Register: 1,
				Data:     []byte{byte(req.Port >> 8), byte(req.Port)}, // 转换端口为网络字节序
			},
			// 设置动作为接受或拒接
			&expr.Verdict{
				Kind: expr.VerdictKind(req.Verdict),
			},
		}
		conn.AddRule(&nftables.Rule{
			Table: table,
			Chain: chain,
			Exprs: exprs,
		})
		if err := conn.Flush(); err != nil {
			return err
		}
		return c.JSON(200, "success")
	case "DELETE":
		for _, table := range tables {
			for _, chain := range chains {
				rules, err := conn.GetRules(table, chain)
				if err != nil {
					continue
				}
				for _, rule := range rules {
					if rule.Table.Name == req.TableName && rule.Chain.Name == req.ChainName && rule.Handle == req.Handle {
						//fmt.Println("匹配到rule")
						if err := conn.DelRule(rule); err != nil {
							return err
						}
						if err := conn.Flush(); err != nil {
							return err
						}
					}
				}
			}
		}

		return c.JSON(200, "success")
	case "GET":
		rulesMap := make(map[uint64]RuleInfo)

		for _, table := range tables {
			for _, chain := range chains {
				rules, err := conn.GetRules(table, chain)
				if err != nil {
					continue
				}
				for _, rule := range rules {
					// 使用Handle作为唯一标识符
					ruleInfo := parseRule(rule)
					rulesMap[rule.Handle] = ruleInfo
				}
			}
		}
		return c.Render(http.StatusOK, "firewall.template", rulesMap)
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
