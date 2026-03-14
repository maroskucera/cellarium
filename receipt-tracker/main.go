// Cellarium Receipt Tracker — a mobile-first PWA for logging receipt values
// Copyright (C) 2026 Maroš Kučera
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package main

import (
	"context"
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/maroskucera/cellarium/receipt-tracker/db/sqlc"
)

//go:embed frontend/*
var frontendFS embed.FS

func main() {
	// Load env files: app dir is the executable's directory, root is one level up.
	exe, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	appDir := filepath.Dir(exe)

	// When running with `go run`, use the working directory instead.
	if wd, err := os.Getwd(); err == nil {
		appDir = wd
	}
	rootDir := filepath.Dir(appDir)
	loadEnvFiles(rootDir, appDir)

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	listenAddr := os.Getenv("LISTEN_ADDR")
	if listenAddr == "" {
		listenAddr = ":8080"
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("unable to connect to database: %v", err)
	}
	defer pool.Close()

	queries := sqlc.New(pool)

	mux := http.NewServeMux()
	mux.Handle("/api/entries", handleCreateEntry(queries))

	frontendSub, err := fs.Sub(frontendFS, "frontend")
	if err != nil {
		log.Fatal(err)
	}
	mux.Handle("/", http.FileServerFS(frontendSub))

	srv := &http.Server{
		Addr:    listenAddr,
		Handler: mux,
	}

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Println("shutting down...")
		srv.Shutdown(context.Background())
	}()

	log.Printf("listening on %s", listenAddr)
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
