package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/merrkry/tele2don/internal/service"
)

func main() {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := service.LoadTempDebugConfig()
	service, err := service.NewBridgeService(cfg)
	if err != nil {
		slog.Error("Failed to create bridge service", "err", err)
		os.Exit(1)
	}

	err = service.Start(ctx)
	if err != nil {
		slog.Error("Failed to start bridge service", "err", err)
	}

	<-stop
	slog.Info("Received shutdown signal, stopping the service.")
}
