package daemon

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/JLugagne/agach-mcp/internal/daemon/app"
	"github.com/JLugagne/agach-mcp/internal/daemon/config"
	"github.com/sirupsen/logrus"
)

func Run() error {
	initConfig := flag.Bool("init", false, "Create a default config file and exit")
	flag.Parse()

	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	workDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	if *initConfig {
		if err := config.WriteDefault(); err != nil {
			return fmt.Errorf("create config: %w", err)
		}
		path, _ := config.DefaultConfigPath()
		logger.WithField("path", path).Info("Config file created")
		return nil
	}

	cfg, err := config.Load(workDir)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	logger.WithField("server", cfg.BaseURL).Info("Configuration loaded")

	daemon, err := app.New(cfg, logger)
	if err != nil {
		return fmt.Errorf("init daemon: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		logger.WithField("signal", sig).Info("Received shutdown signal")
		cancel()
	}()

	if err := daemon.Run(ctx); err != nil {
		if ctx.Err() != nil {
			logger.Info("Daemon stopped")
			return nil
		}
		return err
	}

	return nil
}
