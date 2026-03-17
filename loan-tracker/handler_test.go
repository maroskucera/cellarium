// Cellarium Loan Tracker — HTTP handler tests
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
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/maroskucera/cellarium/loan-tracker/db/sqlc"
)

// stubQuerier implements sqlc.Querier for testing without a real database.
type stubQuerier struct {
	hasEntries     bool
	loanEntry      sqlc.GetLoanEntryRow
	balance        pgtype.Numeric
	totalRepaid    pgtype.Numeric
	lastPayment    sqlc.GetLastPaymentRow
	hasLastPayment bool
	payments       []sqlc.ListPaymentsRow
	createdID      int64
	err            error
}

func makeNumeric(val string) pgtype.Numeric {
	f, _ := new(big.Float).SetString(val)
	cents, _ := new(big.Float).Mul(f, big.NewFloat(100)).Int(nil)
	return pgtype.Numeric{Int: cents, Exp: -2, Valid: true}
}

func (s *stubQuerier) HasEntries(_ context.Context) (bool, error) {
	return s.hasEntries, s.err
}

func (s *stubQuerier) CreateEntry(_ context.Context, _ sqlc.CreateEntryParams) (int64, error) {
	if s.err != nil {
		return 0, s.err
	}
	return s.createdID, nil
}

func (s *stubQuerier) GetLoanEntry(_ context.Context) (sqlc.GetLoanEntryRow, error) {
	return s.loanEntry, s.err
}

func (s *stubQuerier) GetBalance(_ context.Context) (pgtype.Numeric, error) {
	return s.balance, s.err
}

func (s *stubQuerier) GetTotalRepaid(_ context.Context) (pgtype.Numeric, error) {
	return s.totalRepaid, s.err
}

func (s *stubQuerier) GetLastPayment(_ context.Context) (sqlc.GetLastPaymentRow, error) {
	if !s.hasLastPayment {
		return sqlc.GetLastPaymentRow{}, pgx.ErrNoRows
	}
	return s.lastPayment, nil
}

func (s *stubQuerier) ListPayments(_ context.Context) ([]sqlc.ListPaymentsRow, error) {
	return s.payments, s.err
}

func testTemplates(t *testing.T) *template.Template {
	t.Helper()
	tmpl, err := template.ParseGlob("templates/*.html")
	if err != nil {
		t.Fatalf("failed to parse templates: %v", err)
	}
	return tmpl
}

// --- handleIndex tests ---

func TestHandleIndex(t *testing.T) {
	t.Run("no entries shows setup form", func(t *testing.T) {
		stub := &stubQuerier{hasEntries: false}
		tmpl := testTemplates(t)
		handler := handleIndex(stub, tmpl)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("got status %d, want %d", w.Code, http.StatusOK)
		}
		body := w.Body.String()
		if !strings.Contains(body, `action="/setup"`) {
			t.Error("expected setup form with action=/setup")
		}
	})

	t.Run("loan exists no payments shows dashboard with 0 percent", func(t *testing.T) {
		stub := &stubQuerier{
			hasEntries:     true,
			loanEntry:      sqlc.GetLoanEntryRow{ID: 1, Amount: makeNumeric("10000.00")},
			balance:        makeNumeric("10000.00"),
			totalRepaid:    makeNumeric("0.00"),
			hasLastPayment: false,
			payments:       nil,
		}
		tmpl := testTemplates(t)
		handler := handleIndex(stub, tmpl)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("got status %d, want %d", w.Code, http.StatusOK)
		}
		body := w.Body.String()
		if !strings.Contains(body, `action="/payment"`) {
			t.Error("expected payment form with action=/payment")
		}
		if !strings.Contains(body, "0%") {
			t.Error("expected 0% repaid")
		}
		if !strings.Contains(body, "10000.00") {
			t.Error("expected loan amount 10000.00")
		}
		if w.Header().Get("Cache-Control") != "no-store" {
			t.Error("expected Cache-Control: no-store header")
		}
	})

	t.Run("loan exists with payments shows dashboard with stats", func(t *testing.T) {
		stub := &stubQuerier{
			hasEntries:     true,
			loanEntry:      sqlc.GetLoanEntryRow{ID: 1, Amount: makeNumeric("10000.00")},
			balance:        makeNumeric("8500.00"),
			totalRepaid:    makeNumeric("1500.00"),
			hasLastPayment: true,
			lastPayment: sqlc.GetLastPaymentRow{
				ID:     3,
				Amount: makeNumeric("-500.00"),
			},
			payments: []sqlc.ListPaymentsRow{
				{ID: 2, Amount: makeNumeric("-1000.00")},
				{ID: 3, Amount: makeNumeric("-500.00")},
			},
		}
		tmpl := testTemplates(t)
		handler := handleIndex(stub, tmpl)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("got status %d, want %d", w.Code, http.StatusOK)
		}
		body := w.Body.String()
		if !strings.Contains(body, `value="500.00"`) {
			t.Errorf("expected pre-filled amount 500.00, body: %s", body)
		}
		if !strings.Contains(body, "1500.00") {
			t.Error("expected total repaid 1500.00")
		}
		if !strings.Contains(body, "8500.00") {
			t.Error("expected amount remaining 8500.00")
		}
		if !strings.Contains(body, "15%") {
			t.Error("expected 15% repaid")
		}
		if !strings.Contains(body, "Payment history (2)") {
			t.Error("expected payment count 2")
		}
	})
}

// --- handleSetup tests ---

func TestHandleSetup(t *testing.T) {
	t.Run("valid loan only redirects", func(t *testing.T) {
		stub := &stubQuerier{createdID: 1}
		handler := handleSetup(stub)

		form := url.Values{}
		form.Set("loan_amount", "10000.00")
		form.Set("loan_date", "2025-01-15")

		req := httptest.NewRequest(http.MethodPost, "/setup", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusSeeOther {
			t.Fatalf("got status %d, want %d", w.Code, http.StatusSeeOther)
		}
		if loc := w.Header().Get("Location"); loc != "/" {
			t.Errorf("got location %q, want /", loc)
		}
	})

	t.Run("valid loan with initial payments redirects", func(t *testing.T) {
		callCount := 0
		stub := &stubQuerier{createdID: 1}
		// Override CreateEntry to count calls
		countingStub := &countingQuerier{stubQuerier: stub}
		handler := handleSetup(countingStub)

		form := url.Values{}
		form.Set("loan_amount", "10000.00")
		form.Set("loan_date", "2025-01-15")
		form.Set("payment_amount_1", "500.00")
		form.Set("payment_date_1", "2025-02-15")
		// row 2 empty
		form.Set("payment_amount_3", "500.00")
		form.Set("payment_date_3", "2025-04-15")

		req := httptest.NewRequest(http.MethodPost, "/setup", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusSeeOther {
			t.Fatalf("got status %d, want %d", w.Code, http.StatusSeeOther)
		}
		callCount = countingStub.createCount
		// 1 loan + 2 payments = 3 calls
		if callCount != 3 {
			t.Errorf("got %d CreateEntry calls, want 3", callCount)
		}
	})

	t.Run("missing loan amount returns 400", func(t *testing.T) {
		stub := &stubQuerier{createdID: 1}
		handler := handleSetup(stub)

		form := url.Values{}
		form.Set("loan_date", "2025-01-15")

		req := httptest.NewRequest(http.MethodPost, "/setup", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("got status %d, want %d", w.Code, http.StatusBadRequest)
		}
	})

	t.Run("missing loan date returns 400", func(t *testing.T) {
		stub := &stubQuerier{createdID: 1}
		handler := handleSetup(stub)

		form := url.Values{}
		form.Set("loan_amount", "10000.00")

		req := httptest.NewRequest(http.MethodPost, "/setup", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("got status %d, want %d", w.Code, http.StatusBadRequest)
		}
	})

	t.Run("invalid amount returns 400", func(t *testing.T) {
		stub := &stubQuerier{createdID: 1}
		handler := handleSetup(stub)

		form := url.Values{}
		form.Set("loan_amount", "abc")
		form.Set("loan_date", "2025-01-15")

		req := httptest.NewRequest(http.MethodPost, "/setup", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("got status %d, want %d", w.Code, http.StatusBadRequest)
		}
	})

	t.Run("GET method returns 405", func(t *testing.T) {
		stub := &stubQuerier{createdID: 1}
		handler := handleSetup(stub)

		req := httptest.NewRequest(http.MethodGet, "/setup", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("got status %d, want %d", w.Code, http.StatusMethodNotAllowed)
		}
	})
}

// --- handlePayment tests ---

func TestHandlePayment(t *testing.T) {
	t.Run("valid payment redirects", func(t *testing.T) {
		stub := &stubQuerier{createdID: 1}
		handler := handlePayment(stub)

		form := url.Values{}
		form.Set("amount", "500.00")
		form.Set("payment_date", "2026-03-16")

		req := httptest.NewRequest(http.MethodPost, "/payment", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusSeeOther {
			t.Fatalf("got status %d, want %d", w.Code, http.StatusSeeOther)
		}
		if loc := w.Header().Get("Location"); loc != "/" {
			t.Errorf("got location %q, want /", loc)
		}
	})

	t.Run("missing amount returns 400", func(t *testing.T) {
		stub := &stubQuerier{createdID: 1}
		handler := handlePayment(stub)

		form := url.Values{}
		form.Set("payment_date", "2026-03-16")

		req := httptest.NewRequest(http.MethodPost, "/payment", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("got status %d, want %d", w.Code, http.StatusBadRequest)
		}
	})

	t.Run("missing date returns 400", func(t *testing.T) {
		stub := &stubQuerier{createdID: 1}
		handler := handlePayment(stub)

		form := url.Values{}
		form.Set("amount", "500.00")

		req := httptest.NewRequest(http.MethodPost, "/payment", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("got status %d, want %d", w.Code, http.StatusBadRequest)
		}
	})

	t.Run("GET method returns 405", func(t *testing.T) {
		stub := &stubQuerier{createdID: 1}
		handler := handlePayment(stub)

		req := httptest.NewRequest(http.MethodGet, "/payment", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("got status %d, want %d", w.Code, http.StatusMethodNotAllowed)
		}
	})
}

// countingQuerier wraps stubQuerier but counts CreateEntry calls.
type countingQuerier struct {
	*stubQuerier
	createCount int
}

func (c *countingQuerier) CreateEntry(ctx context.Context, arg sqlc.CreateEntryParams) (int64, error) {
	c.createCount++
	return c.stubQuerier.CreateEntry(ctx, arg)
}
