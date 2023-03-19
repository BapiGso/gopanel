package core

import "flag"

type BindFlag struct {
	Domain string
	Port   string
	Reset  bool
}

func (c *Core) BindFlag() {
	c.CommandLineArgs = BindFlag{
		Domain: *flag.String("d", "", "绑定域名，用于申请ssl证书"),
		Port:   *flag.String("p", "8848", "运行端口，默认8488"),
		Reset:  *flag.Bool("r", false, "重设账户名密码"),
	}
	flag.Parse()
}
