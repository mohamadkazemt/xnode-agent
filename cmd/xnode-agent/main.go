package main

import (
	"context"
	"flag"
	"log"
	"os/signal"
	"syscall"

	"xnode-agent/internal/agent"
	"xnode-agent/internal/config"
)

func main() {
	path := flag.String("config", "/etc/xnode/agent.json", "agent config path")
	flag.Parse()
	cfg, err := config.Load(*path)
	if err != nil {
		log.Fatal(err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if err := agent.New(cfg).Run(ctx); err != nil && err != context.Canceled {
		log.Fatal(err)
	}
}
