package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/aman/ais140-socket/internal/config"
	"github.com/aman/ais140-socket/internal/logger"
	"github.com/aman/ais140-socket/internal/server"
)

func main() {
	cfg := config.Load()
	log := logger.New()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	srv := server.New(cfg, log)
	if err := srv.StartServer(ctx); err != nil {
		log.Error("server exited with error", "error", err)
		os.Exit(1)
	}
}
