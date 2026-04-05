// Cellarium Quests — gamified todo list where quests fail if not completed by their date
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
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/maroskucera/cellarium/quests/db/sqlc"
)

//go:embed static/*
var staticFS embed.FS

//go:embed db/migrations/*.sql
var migrationsFS embed.FS

//go:embed templates/*
var templatesFS embed.FS

// timeNow is a function that returns the current time. Override in tests.
var timeNow = time.Now

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
	stripped := strings.TrimPrefix(dbURL, "postgresql://")
	stripped = strings.TrimPrefix(stripped, "postgres://")
	migrateURL := "pgx5://" + stripped
	if strings.Contains(migrateURL, "?") {
		migrateURL += "&x-migrations-table=quests_schema_migrations"
	} else {
		migrateURL += "?x-migrations-table=quests_schema_migrations"
	}
	return migrate.NewWithSourceInstance("iofs", source, migrateURL)
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

	loadEnvFromCwd()
	initLocation()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	listenAddr := os.Getenv("LISTEN_ADDR")
	if listenAddr == "" {
		listenAddr = ":8085"
	}

	vapidPrivKey := os.Getenv("VAPID_PRIVATE_KEY")
	vapidPubKey := os.Getenv("VAPID_PUBLIC_KEY")
	vapidSubject := os.Getenv("VAPID_SUBJECT")

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("unable to connect to database: %v", err)
	}
	defer pool.Close()

	queries := sqlc.New(pool)

	tmpl, err := template.ParseFS(templatesFS, "templates/*.html")
	if err != nil {
		log.Fatalf("failed to parse templates: %v", err)
	}

	mux := http.NewServeMux()

	// Today
	mux.Handle("GET /{$}", handleToday(queries, tmpl))

	// Quest CRUD
	mux.Handle("GET /quests/new", handleNewQuest(queries, tmpl))
	mux.Handle("POST /quests/new", handleNewQuest(queries, tmpl))
	mux.Handle("GET /quests/{id}/edit", handleEditQuest(queries, tmpl))
	mux.Handle("POST /quests/{id}/edit", handleEditQuest(queries, tmpl))
	mux.Handle("POST /quests/{id}/delete", handleDeleteQuest(queries))
	mux.Handle("POST /quests/{id}/complete", handleCompleteQuest(queries))
	mux.Handle("POST /quests/{id}/uncomplete", handleUncompleteQuest(queries))
	mux.Handle("GET /quests/{id}/retry", handleRetryQuest(queries, tmpl))
	mux.Handle("POST /quests/{id}/retry", handleRetryQuest(queries, tmpl))
	mux.Handle("GET /quests/{id}", handleViewQuest(queries, tmpl))
	mux.Handle("POST /quests/reorder", handleReorderQuest(queries))

	// All quests
	mux.Handle("GET /quests", handleAllQuests(queries, tmpl))

	// Quest log
	mux.Handle("GET /log", handleQuestLog(queries, tmpl))

	// Quest lines
	mux.Handle("GET /quest-lines", handleQuestLines(queries, tmpl))
	mux.Handle("GET /quest-lines/new", handleNewQuestLine(queries, tmpl))
	mux.Handle("POST /quest-lines/new", handleNewQuestLine(queries, tmpl))
	mux.Handle("GET /quest-lines/{id}/edit", handleEditQuestLine(queries, tmpl))
	mux.Handle("POST /quest-lines/{id}/edit", handleEditQuestLine(queries, tmpl))
	mux.Handle("POST /quest-lines/{id}/delete", handleDeleteQuestLine(queries))
	mux.Handle("POST /quest-lines/reorder", handleReorderQuestLine(queries))

	// API
	mux.Handle("GET /api/quest-givers", handleQuestGivers(queries))

	// Push notifications
	mux.Handle("POST /api/push/subscribe", handlePushSubscribe(queries))
	mux.Handle("POST /api/push/unsubscribe", handlePushUnsubscribe(queries))
	mux.Handle("GET /api/push/vapid-public-key", handlePushVapidKey(vapidPubKey))

	// Serve service worker and manifest at root scope (required for full PWA)
	mux.Handle("GET /service-worker.js", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript")
		f, err := staticFS.Open("static/service-worker.js")
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		defer f.Close()
		io.Copy(w, f)
	}))
	mux.Handle("GET /manifest.json", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/manifest+json")
		f, err := staticFS.Open("static/manifest.json")
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		defer f.Close()
		io.Copy(w, f)
	}))

	// Static files
	staticSub, err := fs.Sub(staticFS, "static")
	if err != nil {
		log.Fatal(err)
	}
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServerFS(staticSub)))

	// Start background ticker for reminders + overdue failure detection
	cfg := notifyConfig{
		VAPIDPrivateKey: vapidPrivKey,
		VAPIDPublicKey:  vapidPubKey,
		VAPIDSubject:    vapidSubject,
	}
	go startTicker(ctx, queries, cfg)

	srv := &http.Server{
		Addr:    listenAddr,
		Handler: mux,
	}

	go func() {
		<-ctx.Done()
		log.Println("shutting down...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		srv.Shutdown(shutdownCtx)
	}()

	log.Printf("listening on %s", listenAddr)
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
