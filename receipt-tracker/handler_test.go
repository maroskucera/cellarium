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
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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

const testTemplate = `<form>{{if .Error}}<p class="error">{{.Error}}</p>{{end}}{{if .Success}}<p class="success">Entry saved</p>{{end}}<input name="value" value="{{.Value}}"><input name="entry_date" value="{{if .EntryDate}}{{.EntryDate}}{{else}}{{.Today}}{{end}}"><input name="note" value="{{.Note}}"></form>`

func newTestHandler(stub *stubQuerier) http.Handler {
	tmpl := template.Must(template.New("form").Parse(testTemplate))
	return handleRoot(stub, tmpl)
}

func TestHandleRoot(t *testing.T) {
	t.Run("GET renders form with today's date", func(t *testing.T) {
		handler := newTestHandler(newStubQuerier())

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("got status %d, want %d", w.Code, http.StatusOK)
		}

		body := w.Body.String()
		if !strings.Contains(body, "<form>") {
			t.Error("response does not contain <form>")
		}

		today := time.Now().Format("2006-01-02")
		if !strings.Contains(body, today) {
			t.Errorf("response does not contain today's date %s", today)
		}
	})

	t.Run("GET with saved=1 shows success message", func(t *testing.T) {
		handler := newTestHandler(newStubQuerier())

		req := httptest.NewRequest(http.MethodGet, "/?saved=1", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("got status %d, want %d", w.Code, http.StatusOK)
		}

		body := w.Body.String()
		if !strings.Contains(body, "Entry saved") {
			t.Error("response does not contain success message")
		}
	})

	t.Run("POST valid form redirects with saved=1", func(t *testing.T) {
		handler := newTestHandler(newStubQuerier())

		body := "value=42.50&entry_date=2026-03-14&note=groceries"
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusSeeOther {
			t.Fatalf("got status %d, want %d", w.Code, http.StatusSeeOther)
		}

		loc := w.Header().Get("Location")
		if loc != "/?saved=1" {
			t.Errorf("got Location %q, want %q", loc, "/?saved=1")
		}
	})

	t.Run("POST with only value defaults date and redirects", func(t *testing.T) {
		handler := newTestHandler(newStubQuerier())

		body := "value=10.00"
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusSeeOther {
			t.Fatalf("got status %d, want %d", w.Code, http.StatusSeeOther)
		}
	})

	t.Run("POST missing value shows error", func(t *testing.T) {
		handler := newTestHandler(newStubQuerier())

		body := "note=novalue"
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("got status %d, want %d", w.Code, http.StatusOK)
		}

		respBody := w.Body.String()
		if !strings.Contains(respBody, "value is required") {
			t.Error("response does not contain error about missing value")
		}
		if !strings.Contains(respBody, "novalue") {
			t.Error("response does not preserve note field")
		}
	})

	t.Run("POST non-numeric value shows error", func(t *testing.T) {
		handler := newTestHandler(newStubQuerier())

		body := "value=abc"
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("got status %d, want %d", w.Code, http.StatusOK)
		}

		if !strings.Contains(w.Body.String(), "valid decimal number") {
			t.Error("response does not contain error about invalid number")
		}
	})

	t.Run("POST invalid date shows error", func(t *testing.T) {
		handler := newTestHandler(newStubQuerier())

		body := "value=10.00&entry_date=14/03/2026"
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("got status %d, want %d", w.Code, http.StatusOK)
		}

		if !strings.Contains(w.Body.String(), "YYYY-MM-DD") {
			t.Error("response does not contain date format error")
		}
	})

	t.Run("PUT returns 405", func(t *testing.T) {
		handler := newTestHandler(newStubQuerier())

		req := httptest.NewRequest(http.MethodPut, "/", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("got status %d, want %d", w.Code, http.StatusMethodNotAllowed)
		}
	})
}
