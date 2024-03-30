package website

import (
	_ "github.com/caddyserver/caddy/v2/modules/standard"
)

func init() {
	//
	//fmt.Println(caddy.AppDataDir(), caddy.HomeDir())
	//err := caddy.Load([]byte("{\n\t\"apps\": {\n\t\t\"http\": {\n\t\t\t\"servers\": {\n\t\t\t\t\"example\": {\n\t\t\t\t\t\"listen\": [\":2015\"],\n\t\t\t\t\t\"routes\": [\n\t\t\t\t\t\t{\n\t\t\t\t\t\t\t\"handle\": [{\n\t\t\t\t\t\t\t\t\"handler\": \"static_response\",\n\t\t\t\t\t\t\t\t\"body\": \"Hello, orld!\"\n\t\t\t\t\t\t\t}]\n\t\t\t\t\t\t}\n\t\t\t\t\t]\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t}\n}"), true)
	//fmt.Println(caddy.Exiting(), err)
	//caddycmd.Main()
	//caddycmd.RegisterCommand(caddycmd.Command{
	//	Name:      "",
	//	Usage:     "",
	//	Short:     "",
	//	Long:      "",
	//	Flags:     nil,
	//	Func:      nil,
	//	CobraFunc: nil,
	//})
	//caddycmd.Main()
	//fmt.Println(caddycmd.LoadConfig("C:\\Users\\lishunsheng\\Documents\\workspace\\gopanel\\Caddyfile", ""))

	//caddy.Load([]byte(""), true)
	//fmt.Println(caddy.Stop())
	//caddy.RegisterModule(nil)
	//fmt.Println(caddy.HomeDir(), caddy.Modules(), caddy.App(), 123)
	//caddy.Run(&caddy.Config{
	//	Admin: &caddy.AdminConfig{
	//		Disabled:      false,
	//		Listen:        ":2015",
	//		EnforceOrigin: false,
	//		Origins:       nil,
	//		Config:        &caddy.ConfigSettings{
	//			Persist:   nil,
	//			LoadRaw:   nil,
	//			LoadDelay: 0,
	//		},
	//		Identity:      nil,
	//		Remote:        nil,
	//	},
	//	Logging:    nil,
	//	StorageRaw: nil,
	//	AppsRaw:    nil,
	//})
	//caddy.Run(&caddy.Config{})
}
