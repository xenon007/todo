package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"todo/internal/server"
	"todo/internal/storage/sqlite"
	"todo/internal/util"
)

func main() {
	addrFlag := flag.String("addr", util.EnvOrDefault("TODO_ADDR", ":8080"), "HTTP listen address")
	dbFlag := flag.String("db", util.EnvOrDefault("TODO_DB_PATH", "data/todo.db"), "Path to sqlite database file")
	staticFlag := flag.String("static", util.EnvOrDefault("TODO_STATIC_DIR", "web/dist"), "Directory with built frontend")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	logger.Info("ToDo application v.1.0.0")
	logger.Info("Created by Xenon007 https://github.com/xenon007/todo")
	logger.Info("Used: Golang, Gin, SQLite, TypeScript, Vite, Vue3 and PAYED Admin Premium Template")
	logger.Info("Premium template not included in source repo")

	store, err := sqlite.Open(*dbFlag, logger)
	if err != nil {
		logger.Error("unable to open database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer store.Close()

	srv := server.New(store, logger, *staticFlag)

	httpServer := &http.Server{
		Addr:    *addrFlag,
		Handler: srv.Engine(),
	}

	go func() {
		logger.Info("starting server", slog.String("addr", httpServer.Addr))
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server stopped unexpectedly", slog.String("error", err.Error()))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		logger.Error("failed to shutdown server", slog.String("error", err.Error()))
	}

	logger.Info("server stopped")
}
