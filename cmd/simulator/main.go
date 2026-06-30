package main

import (
	"context"
	"errors"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/trueleafmarket-dg/dg-print/printer-simulator/internal"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	loadDotenv()

	cfg, err := internal.LoadConfig()
	if err != nil {
		slog.Error("config error", "err", err)
		os.Exit(2)
	}

	logConfig(cfg)

	state := internal.NewPrinterState()
	retention := internal.NewOutputRetention(cfg.OutputDir, cfg.MaxOutputFiles)
	renderer := internal.NewRenderer(internal.RendererOptions{
		OutputDir:  cfg.OutputDir,
		LabelSize:  cfg.LabelSize,
		Dpmm:       cfg.Dpmm,
		PrintDelay: cfg.PrintDelay,
		Retention:  retention,
	})
	webhook := internal.NewWebhook(cfg.WebhookURL)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	tcpServer := internal.NewTCPServer(internal.TCPServerOptions{
		Addr:     cfg.TCPAddr(),
		State:    state,
		Renderer: renderer,
		Webhook:  webhook,
	})
	go func() {
		if err := tcpServer.Start(ctx); err != nil {
			slog.Error("TCP server error", "err", err)
			os.Exit(1)
		}
	}()

	controlAPI := internal.NewControlAPI(internal.ControlAPIOptions{
		State:         state,
		Renderer:      renderer,
		OutputDir:     cfg.OutputDir,
		BasicAuthUser: cfg.BasicAuthUser,
		BasicAuthPass: cfg.BasicAuthPass,
	})
	httpServer := &http.Server{
		Addr:    cfg.HTTPAddr(),
		Handler: controlAPI.Handler(),
	}

	go func() {
		slog.Info("control API listening", "addr", cfg.HTTPAddr())
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("HTTP server error", "err", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down simulator")
	httpServer.Close()
}

func loadDotenv() {
	path := os.Getenv("ENV_FILE")
	if path == "" {
		path = ".env"
	}
	if err := godotenv.Load(path); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return
		}
		slog.Warn("failed to load env file", "path", path, "err", err)
		return
	}
	slog.Info("loaded env file", "path", path)
}

func logConfig(cfg internal.Config) {
	slog.Info("simulator config",
		"tcp_addr", cfg.TCPAddr(),
		"http_addr", cfg.HTTPAddr(),
		"output_dir", cfg.OutputDir,
		"dpmm", cfg.Dpmm,
		"basic_auth", cfg.BasicAuthEnabled(),
		"print_delay", cfg.PrintDelay,
		"max_output_files", cfg.MaxOutputFiles,
		"webhook", cfg.WebhookURL != "",
	)
}
