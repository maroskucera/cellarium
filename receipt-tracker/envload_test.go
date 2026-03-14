// Cellarium Receipt Tracker — environment file loading tests
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
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadEnvFiles(t *testing.T) {
	t.Run("app .env overrides root .env", func(t *testing.T) {
		rootDir := t.TempDir()
		appDir := filepath.Join(rootDir, "app")
		if err := os.MkdirAll(appDir, 0o755); err != nil {
			t.Fatal(err)
		}

		writeFile(t, filepath.Join(rootDir, ".env"), "MY_VAR=root\n")
		writeFile(t, filepath.Join(appDir, ".env"), "MY_VAR=app\n")

		os.Unsetenv("MY_VAR")
		defer os.Unsetenv("MY_VAR")

		loadEnvFiles(rootDir, appDir)

		got := os.Getenv("MY_VAR")
		if got != "app" {
			t.Errorf("got %q, want %q", got, "app")
		}
	})

	t.Run(".env.local overrides .env", func(t *testing.T) {
		rootDir := t.TempDir()
		appDir := filepath.Join(rootDir, "app")
		if err := os.MkdirAll(appDir, 0o755); err != nil {
			t.Fatal(err)
		}

		writeFile(t, filepath.Join(appDir, ".env"), "MY_VAR=app\n")
		writeFile(t, filepath.Join(appDir, ".env.local"), "MY_VAR=app-local\n")

		os.Unsetenv("MY_VAR")
		defer os.Unsetenv("MY_VAR")

		loadEnvFiles(rootDir, appDir)

		got := os.Getenv("MY_VAR")
		if got != "app-local" {
			t.Errorf("got %q, want %q", got, "app-local")
		}
	})

	t.Run("real env var overrides everything", func(t *testing.T) {
		rootDir := t.TempDir()
		appDir := filepath.Join(rootDir, "app")
		if err := os.MkdirAll(appDir, 0o755); err != nil {
			t.Fatal(err)
		}

		writeFile(t, filepath.Join(appDir, ".env.local"), "MY_VAR=app-local\n")

		os.Setenv("MY_VAR", "real")
		defer os.Unsetenv("MY_VAR")

		loadEnvFiles(rootDir, appDir)

		got := os.Getenv("MY_VAR")
		if got != "real" {
			t.Errorf("got %q, want %q", got, "real")
		}
	})

	t.Run("missing files are silently skipped", func(t *testing.T) {
		rootDir := t.TempDir()
		appDir := filepath.Join(rootDir, "app")
		if err := os.MkdirAll(appDir, 0o755); err != nil {
			t.Fatal(err)
		}

		writeFile(t, filepath.Join(rootDir, ".env"), "MY_VAR=root\n")

		os.Unsetenv("MY_VAR")
		defer os.Unsetenv("MY_VAR")

		loadEnvFiles(rootDir, appDir)

		got := os.Getenv("MY_VAR")
		if got != "root" {
			t.Errorf("got %q, want %q", got, "root")
		}
	})

	t.Run("root .env.local overrides root .env", func(t *testing.T) {
		rootDir := t.TempDir()
		appDir := filepath.Join(rootDir, "app")
		if err := os.MkdirAll(appDir, 0o755); err != nil {
			t.Fatal(err)
		}

		writeFile(t, filepath.Join(rootDir, ".env"), "MY_VAR=root\n")
		writeFile(t, filepath.Join(rootDir, ".env.local"), "MY_VAR=root-local\n")

		os.Unsetenv("MY_VAR")
		defer os.Unsetenv("MY_VAR")

		loadEnvFiles(rootDir, appDir)

		got := os.Getenv("MY_VAR")
		if got != "root-local" {
			t.Errorf("got %q, want %q", got, "root-local")
		}
	})
}
