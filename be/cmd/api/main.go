// Package main is the entrypoint for the Steins API server.
//
// Usage:
//
//	api [-addr :8080] [-data ./data/manga] [-pretty] [-cors http://localhost:5173,...]
package main

import (
	"context"
	"errors"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"steins/internal/api"
	"steins/internal/api/handler"
	"steins/internal/chapter"
	"steins/internal/integrity"
	"steins/internal/manga"
	"steins/internal/storage/fs"
	"steins/pkg/logger"
)

func main() {
	addr := flag.String("addr", ":8080", "http listen address")
	dataDir := flag.String("data", "./data/manga", "catalog root directory")
	pretty := flag.Bool("pretty", false, "pretty (console) log output")
	cors := flag.String("cors", "", "comma-separated allowed CORS origins (empty = reflect any origin)")
	flag.Parse()

	log := logger.New(*pretty)

	store, err := fs.New(*dataDir)
	if err != nil {
		log.WithError(err).Fatal().Str("data_dir", *dataDir).Msg("failed to load catalog")
	}

	mangaSvc := manga.NewService(store)
	chapterSvc := chapter.NewService(store, store, integrity.NewCalculator())
	handlers := handler.NewHandlers(mangaSvc, chapterSvc)

	cfg := api.Config{AllowedOrigins: parseOrigins(*cors)}
	router := api.NewRouter(cfg, log, handlers)

	srv := &http.Server{
		Addr:              *addr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		log.Info().Str("addr", *addr).Str("data_dir", *dataDir).Msg("api server starting")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.WithError(err).Fatal().Msg("server failed")
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	log.Info().Msg("shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.WithError(err).Error().Msg("graceful shutdown failed")
	}
	log.Info().Msg("api server stopped")
}

func parseOrigins(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
