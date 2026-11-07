package core

import (
	"fmt"
	"net"

	"gopanel/core/config"
	"gopanel/core/frp"
	"gopanel/core/headscale"
	"gopanel/core/monitor"
	"gopanel/core/website"
)

func Initialize(directory string) error {
	if err := config.Init(directory); err != nil {
		return err
	}
	directory = config.Directory()
	settings, err := monitor.LoadConfig(directory)
	if err != nil {
		return fmt.Errorf("initialize monitoring: %w", err)
	}
	if settings.Role == "probe" {
		return nil
	}
	if err := website.Initialize(directory); err != nil {
		return fmt.Errorf("initialize Caddy configuration: %w", err)
	}
	if err := frp.Initialize(directory); err != nil {
		return fmt.Errorf("initialize FRP configuration: %w", err)
	}
	if err := headscale.Initialize(directory); err != nil {
		return fmt.Errorf("initialize Headscale configuration: %w", err)
	}
	return nil
}

func (panel *Core) autostartServices() {
	if err := website.Autostart(); err != nil {
		panel.e.Logger.Error("Caddy auto-start failed", "error", err)
	}
	if err := frp.Autostart(); err != nil {
		panel.e.Logger.Error("FRP auto-start failed", "error", err)
	}
}

func printStartupInfo(settings config.PanelConfig) {
	addresses, err := net.InterfaceAddrs()
	if err == nil {
		for _, address := range addresses {
			network, ok := address.(*net.IPNet)
			if !ok || network.IP.IsLoopback() || !network.IP.IsGlobalUnicast() {
				continue
			}
			host := network.IP.String()
			if network.IP.To4() == nil {
				host = "[" + host + "]"
			}
			fmt.Printf("gopanel started on https://%s%s/%s\n", host, settings.Port, settings.Path)
		}
	}
	fmt.Printf("Panel Port: %s\nPanel Path: %s\nPanel Username: %s\nPanel Password: %s\n", settings.Port, settings.Path, settings.Username, settings.Password)
}
