// Cellarium Loan Tracker — hierarchical .env file loading
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
	"path/filepath"

	"github.com/joho/godotenv"
)

// loadEnvFiles loads environment variables from .env files in priority order.
// godotenv does not override already-set variables, so we load highest priority first.
// Priority: app .env.local > app .env > root .env.local > root .env > real env vars (always win).
func loadEnvFiles(rootDir, appDir string) {
	files := []string{
		filepath.Join(appDir, ".env.local"),
		filepath.Join(appDir, ".env"),
		filepath.Join(rootDir, ".env.local"),
		filepath.Join(rootDir, ".env"),
	}
	for _, f := range files {
		_ = godotenv.Load(f)
	}
}
