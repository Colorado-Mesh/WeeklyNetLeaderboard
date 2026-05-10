package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"weeklynet/internal/config"
	"weeklynet/internal/ingest"
	"weeklynet/internal/logging"
	"weeklynet/internal/mqtt"
	"weeklynet/internal/storage"
	"weeklynet/internal/web"
)

const (
	readHeaderTimeout = 5 * time.Second
	readTimeout       = 15 * time.Second
	writeTimeout      = 20 * time.Second
	idleTimeout       = 60 * time.Second
	maxHeaderBytes    = 1 << 20
)

func main() {
	_ = config.LoadEnvFile(".env.local")
	logger := logging.NewJSONLogger()

	cfg, err := config.Load()
	if err != nil {
		logger.Error("load config", "error", err.Error())
		os.Exit(1)
	}

	store, err := storage.OpenSQLite(cfg.SQLitePath)
	if err != nil {
		logger.Error("open sqlite", "error", err.Error())
		os.Exit(1)
	}
	defer func() {
		if err := store.Close(); err != nil {
			logger.Warn("close sqlite", "error", err.Error())
		}
	}()

	webServer, err := web.NewServer(cfg, store, logger)
	if err != nil {
		logger.Error("create web server", "error", err.Error())
		os.Exit(1)
	}

	httpServer := newHTTPServer(cfg.HTTPAddr, webServer.Routes())

	ingestService := ingest.NewService(cfg, store, logger)
	mqttConsumer := mqtt.NewConsumer(
		logger,
		cfg.MQTTBrokerURL,
		cfg.MQTTClientID,
		cfg.MQTTUsername,
		cfg.MQTTPassword,
		cfg.MQTTMaxPayloadBytes,
	)
	topic := cfg.TopicForIATA(cfg.IATADefault)
	if err := mqttConsumer.Subscribe(topic, func(ctx context.Context, msg mqtt.Message) {
		ingestService.HandleMessage(ctx, msg.Topic, msg.PayloadHex, msg.ObservedAt)
	}); err != nil {
		logger.Warn("mqtt subscribe registration failed", "error", err.Error(), "topic", topic)
	}
	defer mqttConsumer.Close()

	go connectMQTTWithRetry(logger, mqttConsumer)

	go func() {
		logger.Info("http listening", "addr", cfg.HTTPAddr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("http server error", "error", err.Error())
			os.Exit(1)
		}
	}()

	waitForShutdown(logger, httpServer)
}

func newHTTPServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
		MaxHeaderBytes:    maxHeaderBytes,
	}
}

func connectMQTTWithRetry(logger *slog.Logger, consumer *mqtt.Consumer) {
	const retryDelay = 10 * time.Second
	for {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		err := consumer.Connect(ctx)
		cancel()
		if err == nil {
			return
		}
		logger.Warn("mqtt connect failed; app still serving http", "error", err.Error())
		time.Sleep(retryDelay)
	}
}

func waitForShutdown(logger *slog.Logger, httpServer *http.Server) {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(ctx); err != nil {
		logger.Warn("http shutdown failed", "error", err.Error())
	}
}
