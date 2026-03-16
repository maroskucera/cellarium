// Cellarium Receipt Tracker — a mobile-first form for logging receipt values
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
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/maroskucera/cellarium/receipt-tracker/db/sqlc"
)

//go:embed templates/form.html
var formTemplate string

//go:embed static
var staticFS embed.FS

//go:embed db/migrations/*.sql
var migrationsFS embed.FS

func loadEnvFromCwd() {
	appDir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	rootDir := filepath.Dir(appDir)
	loadEnvFiles(rootDir, appDir)
}

func newMigrate(dbURL string) (*migrate.Migrate, error) {
	source, err := iofs.New(migrationsFS, "db/migrations")
	if err != nil {
		return nil, err
	}
	return migrate.NewWithSourceInstance("iofs", source, "pgx5://"+dbURL[len("postgres://"):])
}

func runMigrations(dbURL string, steps int) error {
	m, err := newMigrate(dbURL)
	if err != nil {
		return err
	}
	defer m.Close()
	if steps == 0 {
		err = m.Up()
	} else {
		err = m.Steps(steps)
	}
	if errors.Is(err, migrate.ErrNoChange) {
		return nil
	}
	return err
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "-migrate" {
		args := os.Args[2:]

		// Load env and get DATABASE_URL early for migrate commands.
		loadEnvFromCwd()

		dbURL := os.Getenv("DATABASE_URL")
		if dbURL == "" {
			log.Fatal("DATABASE_URL is required")
		}

		dir := "up"
		if len(args) > 0 {
			dir = args[0]
		}

		switch dir {
		case "up":
			if len(args) > 1 {
				n, err := strconv.Atoi(args[1])
				if err != nil || n <= 0 {
					log.Fatal("usage: -migrate up [N]")
				}
				log.Printf("applying %d migration(s)...", n)
				if err := runMigrations(dbURL, n); err != nil {
					log.Fatalf("migration failed: %v", err)
				}
			} else {
				log.Println("applying all pending migrations...")
				if err := runMigrations(dbURL, 0); err != nil {
					log.Fatalf("migration failed: %v", err)
				}
			}
			log.Println("migrations complete")
		case "down":
			if len(args) < 2 {
				log.Fatal("usage: -migrate down N (number of migrations to roll back is required)")
			}
			n, err := strconv.Atoi(args[1])
			if err != nil || n <= 0 {
				log.Fatal("usage: -migrate down N")
			}
			log.Printf("rolling back %d migration(s)...", n)
			if err := runMigrations(dbURL, -n); err != nil {
				log.Fatalf("rollback failed: %v", err)
			}
			log.Println("rollback complete")
		default:
			fmt.Fprintf(os.Stderr, "unknown migrate direction: %s\nusage: -migrate [up [N] | down N]\n", dir)
			os.Exit(1)
		}
		return
	}

	// Load env files for normal server startup.
	loadEnvFromCwd()

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
	tmpl := template.Must(template.New("form").Parse(formTemplate))

	mux := http.NewServeMux()
	mux.Handle("/", handleRoot(queries, tmpl))
	mux.Handle("/static/", http.FileServerFS(staticFS))

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
