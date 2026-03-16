// Cellarium Receipt Tracker — HTTP handler tests
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
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/maroskucera/cellarium/receipt-tracker/db/sqlc"
)

type stubQuerier struct {
	id  int64
	err error
}

func (s *stubQuerier) CreateEntry(_ context.Context, _ sqlc.CreateEntryParams) (int64, error) {
	if s.err != nil {
		return 0, s.err
	}
	return s.id, nil
}

func newStubQuerier() *stubQuerier {
	return &stubQuerier{id: 1}
}

func TestHandleCreateEntry(t *testing.T) {
	t.Run("valid request with all fields", func(t *testing.T) {
		stub := newStubQuerier()
		handler := handleCreateEntry(stub)

		body := `{"value":"42.50","entry_date":"2026-03-14","note":"groceries"}`
		req := httptest.NewRequest(http.MethodPost, "/api/entries", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("got status %d, want %d", w.Code, http.StatusCreated)
		}

		var resp createEntryResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if resp.ID != 1 {
			t.Errorf("got ID %d, want 1", resp.ID)
		}
	})

	t.Run("valid request with only value defaults date and note", func(t *testing.T) {
		stub := newStubQuerier()
		handler := handleCreateEntry(stub)

		body := `{"value":"10.00"}`
		req := httptest.NewRequest(http.MethodPost, "/api/entries", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("got status %d, want %d", w.Code, http.StatusCreated)
		}

		var resp createEntryResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if resp.ID != 1 {
			t.Errorf("got ID %d, want 1", resp.ID)
		}
	})

	t.Run("missing value returns 400", func(t *testing.T) {
		stub := newStubQuerier()
		handler := handleCreateEntry(stub)

		body := `{"note":"no value"}`
		req := httptest.NewRequest(http.MethodPost, "/api/entries", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("got status %d, want %d", w.Code, http.StatusBadRequest)
		}
	})

	t.Run("non-numeric value returns 400", func(t *testing.T) {
		stub := newStubQuerier()
		handler := handleCreateEntry(stub)

		body := `{"value":"abc"}`
		req := httptest.NewRequest(http.MethodPost, "/api/entries", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("got status %d, want %d", w.Code, http.StatusBadRequest)
		}
	})

	t.Run("invalid date format returns 400", func(t *testing.T) {
		stub := newStubQuerier()
		handler := handleCreateEntry(stub)

		body := `{"value":"10.00","entry_date":"14/03/2026"}`
		req := httptest.NewRequest(http.MethodPost, "/api/entries", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("got status %d, want %d", w.Code, http.StatusBadRequest)
		}
	})

	t.Run("wrong HTTP method returns 405", func(t *testing.T) {
		stub := newStubQuerier()
		handler := handleCreateEntry(stub)

		req := httptest.NewRequest(http.MethodGet, "/api/entries", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("got status %d, want %d", w.Code, http.StatusMethodNotAllowed)
		}
	})
}
