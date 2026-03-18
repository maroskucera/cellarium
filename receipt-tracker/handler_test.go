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
	"errors"
	"html/template"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/maroskucera/cellarium/receipt-tracker/db/sqlc"
)

type stubQuerier struct {
	id            int64
	err           error
	unpaidEntries []sqlc.ListUnpaidEntriesRow
	listUnpaidErr error
	markPaidIDs   []int64
	markPaidErr   error
}

func (s *stubQuerier) CreateEntry(_ context.Context, _ sqlc.CreateEntryParams) (int64, error) {
	if s.err != nil {
		return 0, s.err
	}
	return s.id, nil
}

func (s *stubQuerier) ListUnpaidEntries(_ context.Context) ([]sqlc.ListUnpaidEntriesRow, error) {
	return s.unpaidEntries, s.listUnpaidErr
}

func (s *stubQuerier) MarkEntriesPaid(_ context.Context, ids []int64) error {
	s.markPaidIDs = ids
	return s.markPaidErr
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

func numericFromString(s string) pgtype.Numeric {
	val, _ := new(big.Float).SetString(s)
	val100, _ := new(big.Float).Mul(val, big.NewFloat(100)).Int(nil)
	return pgtype.Numeric{Int: val100, Exp: -2, Valid: true}
}

func dateFromString(s string) pgtype.Date {
	t, _ := time.Parse("2006-01-02", s)
	return pgtype.Date{Time: t, Valid: true}
}

const testPaidTemplate = `{{if .Success}}<div class="success">Entries marked as paid</div>{{end}}{{if .Error}}<div class="error">{{.Error}}</div>{{end}}{{if .Batches}}<table>{{range .Batches}}<tr class="batch-header"><td></td><td>Batch {{.Batch}}</td><td></td><td><input type="checkbox" class="batch-toggle" data-batch="{{.Batch}}"></td></tr>{{range .Entries}}<tr><td>{{.Date}}</td><td>{{.Amount}}</td><td>{{.Batch}}</td><td><input type="checkbox" name="ids" value="{{.ID}}" data-batch="{{.Batch}}"></td></tr>{{end}}{{end}}</table>{{else}}<p class="empty">No outstanding receipts</p>{{end}}`

func newTestPaidHandler(stub *stubQuerier) http.Handler {
	tmpl := template.Must(template.New("paid").Parse(testPaidTemplate))
	return handlePaid(stub, tmpl)
}

func TestHandlePaid(t *testing.T) {
	t.Run("GET renders table with entries grouped by batch", func(t *testing.T) {
		stub := newStubQuerier()
		stub.unpaidEntries = []sqlc.ListUnpaidEntriesRow{
			{ID: 1, Value: numericFromString("10.50"), EntryDate: dateFromString("2026-03-10"), Batch: 1},
			{ID: 2, Value: numericFromString("20.00"), EntryDate: dateFromString("2026-03-11"), Batch: 1},
			{ID: 3, Value: numericFromString("5.75"), EntryDate: dateFromString("2026-03-15"), Batch: 2},
		}
		handler := newTestPaidHandler(stub)

		req := httptest.NewRequest(http.MethodGet, "/paid", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("got status %d, want %d", w.Code, http.StatusOK)
		}

		body := w.Body.String()
		if !strings.Contains(body, "Batch 1") {
			t.Error("response does not contain Batch 1 header")
		}
		if !strings.Contains(body, "Batch 2") {
			t.Error("response does not contain Batch 2 header")
		}
		if !strings.Contains(body, "10.50") {
			t.Error("response does not contain amount 10.50")
		}
		if !strings.Contains(body, "20.00") {
			t.Error("response does not contain amount 20.00")
		}
		if !strings.Contains(body, "5.75") {
			t.Error("response does not contain amount 5.75")
		}
		if !strings.Contains(body, "2026-03-10") {
			t.Error("response does not contain date 2026-03-10")
		}
	})

	t.Run("GET with no entries shows empty message", func(t *testing.T) {
		stub := newStubQuerier()
		stub.unpaidEntries = nil
		handler := newTestPaidHandler(stub)

		req := httptest.NewRequest(http.MethodGet, "/paid", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("got status %d, want %d", w.Code, http.StatusOK)
		}

		if !strings.Contains(w.Body.String(), "No outstanding receipts") {
			t.Error("response does not contain empty state message")
		}
	})

	t.Run("GET renders batch header checkboxes with data-batch", func(t *testing.T) {
		stub := newStubQuerier()
		stub.unpaidEntries = []sqlc.ListUnpaidEntriesRow{
			{ID: 1, Value: numericFromString("10.00"), EntryDate: dateFromString("2026-03-10"), Batch: 3},
		}
		handler := newTestPaidHandler(stub)

		req := httptest.NewRequest(http.MethodGet, "/paid", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		body := w.Body.String()
		if !strings.Contains(body, `data-batch="3"`) {
			t.Error("response does not contain data-batch attribute for batch 3")
		}
		if !strings.Contains(body, `class="batch-toggle"`) {
			t.Error("response does not contain batch-toggle checkbox")
		}
	})

	t.Run("GET with db error returns 500", func(t *testing.T) {
		stub := newStubQuerier()
		stub.listUnpaidErr = errors.New("db error")
		handler := newTestPaidHandler(stub)

		req := httptest.NewRequest(http.MethodGet, "/paid", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("got status %d, want %d", w.Code, http.StatusInternalServerError)
		}
	})

	t.Run("GET with saved=1 shows success message", func(t *testing.T) {
		stub := newStubQuerier()
		handler := newTestPaidHandler(stub)

		req := httptest.NewRequest(http.MethodGet, "/paid?saved=1", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if !strings.Contains(w.Body.String(), "Entries marked as paid") {
			t.Error("response does not contain success message")
		}
	})

	t.Run("POST with selected IDs marks as paid and redirects", func(t *testing.T) {
		stub := newStubQuerier()
		handler := newTestPaidHandler(stub)

		body := "ids=1&ids=3&ids=5"
		req := httptest.NewRequest(http.MethodPost, "/paid", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusSeeOther {
			t.Fatalf("got status %d, want %d", w.Code, http.StatusSeeOther)
		}

		loc := w.Header().Get("Location")
		if loc != "/paid?saved=1" {
			t.Errorf("got Location %q, want %q", loc, "/paid?saved=1")
		}

		if len(stub.markPaidIDs) != 3 {
			t.Fatalf("got %d IDs, want 3", len(stub.markPaidIDs))
		}
		want := []int64{1, 3, 5}
		for i, id := range stub.markPaidIDs {
			if id != want[i] {
				t.Errorf("markPaidIDs[%d] = %d, want %d", i, id, want[i])
			}
		}
	})

	t.Run("POST with no IDs redirects without marking", func(t *testing.T) {
		stub := newStubQuerier()
		handler := newTestPaidHandler(stub)

		req := httptest.NewRequest(http.MethodPost, "/paid", strings.NewReader(""))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusSeeOther {
			t.Fatalf("got status %d, want %d", w.Code, http.StatusSeeOther)
		}

		if stub.markPaidIDs != nil {
			t.Errorf("expected no IDs marked, got %v", stub.markPaidIDs)
		}
	})

	t.Run("POST with invalid ID returns 400", func(t *testing.T) {
		stub := newStubQuerier()
		handler := newTestPaidHandler(stub)

		body := "ids=abc"
		req := httptest.NewRequest(http.MethodPost, "/paid", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("got status %d, want %d", w.Code, http.StatusBadRequest)
		}
	})

	t.Run("PUT returns 405", func(t *testing.T) {
		stub := newStubQuerier()
		handler := newTestPaidHandler(stub)

		req := httptest.NewRequest(http.MethodPut, "/paid", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("got status %d, want %d", w.Code, http.StatusMethodNotAllowed)
		}
	})
}
