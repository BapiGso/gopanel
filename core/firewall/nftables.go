package firewall

import (
	"fmt"
	"github.com/google/nftables"
	"golang.org/x/sys/unix"
)

func createChain(conn *nftables.Conn, table *nftables.Table, hooknum nftables.ChainHook) {
	ipv4Table := conn.AddTable(&nftables.Table{
		Name:   "ipv4table",
		Family: nftables.TableFamilyIPv4,
	})

	// 创建 IPv6 表
	ipv6Table := conn.AddTable(&nftables.Table{
		Name:   "ipv6table",
		Family: nftables.TableFamilyIPv6,
	})
	fmt.Println(ipv4Table, ipv6Table)
	chain := conn.AddChain(&nftables.Chain{
		Name:     "",
		Table:    table,
		Hooknum:  nftables.ChainHookInput,
		Priority: nftables.ChainPriorityFilter,
		Type:     nftables.ChainTypeFilter,
	})

	// 添加规则来阻止 TCP 流量
	addDropRule(conn, table, chain, unix.IPPROTO_TCP)

	// 添加规则来阻止 UDP 流量
	addDropRule(conn, table, chain, unix.IPPROTO_UDP)

}

func addDropRule(conn *nftables.Conn, table *nftables.Table, chain *nftables.Chain, tcp int) {

}
