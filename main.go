package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"gopanel/core"
	"gopanel/core/config"
	"gopanel/core/monitor"
)

func main() {
	directory := os.Getenv("GOPANEL_CONFIG_DIR")
	if directory == "" {
		var err error
		directory, err = os.Getwd()
		if err != nil {
			fmt.Fprintln(os.Stderr, "resolve configuration directory:", err)
			os.Exit(1)
		}
	}
	if err := core.Initialize(directory); err != nil {
		fmt.Fprintln(os.Stderr, "initialize panel:", err)
		os.Exit(1)
	}
	settings, err := monitor.LoadConfig(config.Directory())
	if err != nil {
		fmt.Fprintln(os.Stderr, "initialize monitor:", err)
		os.Exit(1)
	}
	if settings.Role == "probe" {
		service, err := monitor.NewService(settings, nil, slog.Default())
		if err != nil {
			fmt.Fprintln(os.Stderr, "initialize probe:", err)
			os.Exit(1)
		}
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		if err := service.Start(ctx); err != nil {
			service.Close(context.Background())
			fmt.Fprintln(os.Stderr, "start probe:", err)
			os.Exit(1)
		}
		service.Wait()
		return
	}
	c := core.New()
	c.Route()
}
