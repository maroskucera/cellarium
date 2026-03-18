// Cellarium Pockets — HTTP handler tests
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
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/maroskucera/cellarium/pockets/db/sqlc"
)

func makeNumeric(val string) pgtype.Numeric {
	f, _ := new(big.Float).SetString(val)
	cents, _ := new(big.Float).Mul(f, big.NewFloat(100)).Int(nil)
	return pgtype.Numeric{Int: cents, Exp: -2, Valid: true}
}

func testTemplates(t *testing.T) *template.Template {
	t.Helper()
	tmpl, err := template.ParseGlob("templates/*.html")
	if err != nil {
		t.Fatalf("failed to parse templates: %v", err)
	}
	return tmpl
}

func fixedTime() time.Time {
	return time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
}

func init() {
	timeNow = fixedTime
}

// stubQuerier implements sqlc.Querier for testing.
type stubQuerier struct {
	accounts                 []sqlc.PocketsAccount
	account                  sqlc.PocketsAccount
	balance                  pgtype.Numeric
	firstReserveID           int64
	transactions             []sqlc.PocketsTransaction
	transaction              sqlc.PocketsTransaction
	topupRules               []sqlc.PocketsTopupRule
	topupRule                sqlc.PocketsTopupRule
	autoTopup                sqlc.PocketsTransaction
	autoTopupErr             error
	createdID                int64
	err                      error
	createTxnCalls           []sqlc.CreateTransactionParams
	updateAutoTopupCall      *sqlc.UpdateAutoTopupAmountParams
	updateTxnCall            *sqlc.UpdateTransactionParams
	futureTransactions       []sqlc.PocketsTransaction
	balancesByAccount        map[int64]pgtype.Numeric
	futureTransactionsByAcct map[int64][]sqlc.PocketsTransaction
	topupRulesByAccount      map[int64][]sqlc.PocketsTopupRule
}

func (s *stubQuerier) CreateAccount(_ context.Context, _ sqlc.CreateAccountParams) (int64, error) {
	return s.createdID, s.err
}

func (s *stubQuerier) GetAccount(_ context.Context, _ int64) (sqlc.PocketsAccount, error) {
	if s.err != nil {
		return sqlc.PocketsAccount{}, s.err
	}
	return s.account, nil
}

func (s *stubQuerier) ListAccounts(_ context.Context) ([]sqlc.PocketsAccount, error) {
	return s.accounts, s.err
}

func (s *stubQuerier) UpdateAccount(_ context.Context, _ sqlc.UpdateAccountParams) error {
	return s.err
}

func (s *stubQuerier) GetAccountBalanceAsOfDate(_ context.Context, arg sqlc.GetAccountBalanceAsOfDateParams) (pgtype.Numeric, error) {
	if s.balancesByAccount != nil {
		if b, ok := s.balancesByAccount[arg.AccountID]; ok {
			return b, s.err
		}
	}
	return s.balance, s.err
}

func (s *stubQuerier) ListFutureTransactions(_ context.Context, arg sqlc.ListFutureTransactionsParams) ([]sqlc.PocketsTransaction, error) {
	if s.futureTransactionsByAcct != nil {
		if txns, ok := s.futureTransactionsByAcct[arg.AccountID]; ok {
			return txns, s.err
		}
	}
	return s.futureTransactions, s.err
}

func (s *stubQuerier) GetFirstReserveAccountID(_ context.Context) (int64, error) {
	if s.firstReserveID == 0 {
		return 0, pgx.ErrNoRows
	}
	return s.firstReserveID, nil
}

func (s *stubQuerier) CreateTransaction(_ context.Context, arg sqlc.CreateTransactionParams) (int64, error) {
	s.createTxnCalls = append(s.createTxnCalls, arg)
	return s.createdID, s.err
}

func (s *stubQuerier) GetTransaction(_ context.Context, _ int64) (sqlc.PocketsTransaction, error) {
	if s.err != nil {
		return sqlc.PocketsTransaction{}, s.err
	}
	return s.transaction, nil
}

func (s *stubQuerier) UpdateTransaction(_ context.Context, arg sqlc.UpdateTransactionParams) error {
	s.updateTxnCall = &arg
	return s.err
}

func (s *stubQuerier) ListTransactionsAll(_ context.Context, _ int64) ([]sqlc.PocketsTransaction, error) {
	return s.transactions, s.err
}

func (s *stubQuerier) ListTransactionsTopups(_ context.Context, _ int64) ([]sqlc.PocketsTransaction, error) {
	return s.transactions, s.err
}

func (s *stubQuerier) ListTransactionsAuto(_ context.Context, _ int64) ([]sqlc.PocketsTransaction, error) {
	return s.transactions, s.err
}

func (s *stubQuerier) ListTransactionsWithdrawals(_ context.Context, _ int64) ([]sqlc.PocketsTransaction, error) {
	return s.transactions, s.err
}

func (s *stubQuerier) GetAutoTopupForDate(_ context.Context, _ sqlc.GetAutoTopupForDateParams) (sqlc.PocketsTransaction, error) {
	if s.autoTopupErr != nil {
		return sqlc.PocketsTransaction{}, s.autoTopupErr
	}
	return s.autoTopup, nil
}

func (s *stubQuerier) UpdateAutoTopupAmount(_ context.Context, arg sqlc.UpdateAutoTopupAmountParams) error {
	s.updateAutoTopupCall = &arg
	return s.err
}

func (s *stubQuerier) CreateTopupRule(_ context.Context, _ sqlc.CreateTopupRuleParams) (int64, error) {
	return s.createdID, s.err
}

func (s *stubQuerier) ListTopupRules(_ context.Context, accountID int64) ([]sqlc.PocketsTopupRule, error) {
	if s.topupRulesByAccount != nil {
		if rules, ok := s.topupRulesByAccount[accountID]; ok {
			return rules, s.err
		}
	}
	return s.topupRules, s.err
}

func (s *stubQuerier) DeleteTopupRule(_ context.Context, _ sqlc.DeleteTopupRuleParams) error {
	return s.err
}

func (s *stubQuerier) GetTopupRule(_ context.Context, _ int64) (sqlc.PocketsTopupRule, error) {
	return s.topupRule, s.err
}

// --- Dashboard tests ---

func TestHandleDashboard(t *testing.T) {
	t.Run("empty state", func(t *testing.T) {
		stub := &stubQuerier{autoTopupErr: pgx.ErrNoRows}
		tmpl := testTemplates(t)
		handler := handleDashboard(stub, tmpl)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("got status %d, want %d", w.Code, http.StatusOK)
		}
		body := w.Body.String()
		if !strings.Contains(body, "No accounts yet") {
			t.Error("expected empty state message")
		}
		if !strings.Contains(body, `href="/accounts/new"`) {
			t.Error("expected new account link")
		}
	})

	t.Run("shows accounts with balances", func(t *testing.T) {
		stub := &stubQuerier{
			accounts: []sqlc.PocketsAccount{
				{ID: 1, Name: "Savings", Icon: "💰", Colour: "steel"},
				{ID: 2, Name: "Travel", Icon: "✈️", Colour: "teal", TargetAmount: makeNumeric("5000.00")},
			},
			balance:      makeNumeric("1250.00"),
			autoTopupErr: pgx.ErrNoRows,
		}
		tmpl := testTemplates(t)
		handler := handleDashboard(stub, tmpl)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("got status %d, want %d", w.Code, http.StatusOK)
		}
		body := w.Body.String()
		if !strings.Contains(body, "Savings") {
			t.Error("expected account name Savings")
		}
		if !strings.Contains(body, "1 250,00") {
			t.Error("expected balance 1 250,00")
		}
		if w.Header().Get("Cache-Control") != "no-store" {
			t.Error("expected Cache-Control: no-store")
		}
	})
}

// --- Account handler tests ---

func TestHandleNewAccount(t *testing.T) {
	tmpl := testTemplates(t)
	handler := handleNewAccount(tmpl)

	req := httptest.NewRequest(http.MethodGet, "/accounts/new", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("got status %d, want %d", w.Code, http.StatusOK)
	}
	body := w.Body.String()
	if !strings.Contains(body, `action="/accounts"`) {
		t.Error("expected form action /accounts")
	}
	if !strings.Contains(body, "ruby") {
		t.Error("expected colour options")
	}
}

func TestHandleCreateAccount(t *testing.T) {
	t.Run("valid account redirects", func(t *testing.T) {
		stub := &stubQuerier{createdID: 42, autoTopupErr: pgx.ErrNoRows}
		tmpl := testTemplates(t)
		handler := handleCreateAccount(stub, tmpl)

		form := url.Values{}
		form.Set("name", "Savings")
		form.Set("icon", "💰")
		form.Set("colour", "steel")

		req := httptest.NewRequest(http.MethodPost, "/accounts", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusSeeOther {
			t.Fatalf("got status %d, want %d", w.Code, http.StatusSeeOther)
		}
		if loc := w.Header().Get("Location"); loc != "/accounts/42" {
			t.Errorf("got location %q, want /accounts/42", loc)
		}
	})

	t.Run("with initial balance creates transaction", func(t *testing.T) {
		stub := &stubQuerier{createdID: 1, autoTopupErr: pgx.ErrNoRows}
		tmpl := testTemplates(t)
		handler := handleCreateAccount(stub, tmpl)

		form := url.Values{}
		form.Set("name", "Savings")
		form.Set("icon", "💰")
		form.Set("colour", "steel")
		form.Set("initial_balance", "1000.00")

		req := httptest.NewRequest(http.MethodPost, "/accounts", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusSeeOther {
			t.Fatalf("got status %d, want %d", w.Code, http.StatusSeeOther)
		}
		if len(stub.createTxnCalls) != 1 {
			t.Fatalf("expected 1 CreateTransaction call, got %d", len(stub.createTxnCalls))
		}
		call := stub.createTxnCalls[0]
		if !call.IsInflow {
			t.Error("initial balance should be an inflow")
		}
		if !call.IsInitialBalance {
			t.Error("should be marked as initial balance")
		}
	})

	t.Run("missing name returns 400", func(t *testing.T) {
		stub := &stubQuerier{createdID: 1}
		tmpl := testTemplates(t)
		handler := handleCreateAccount(stub, tmpl)

		form := url.Values{}
		form.Set("icon", "💰")
		form.Set("colour", "steel")

		req := httptest.NewRequest(http.MethodPost, "/accounts", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("got status %d, want %d", w.Code, http.StatusBadRequest)
		}
	})

	t.Run("invalid colour returns 400", func(t *testing.T) {
		stub := &stubQuerier{createdID: 1}
		tmpl := testTemplates(t)
		handler := handleCreateAccount(stub, tmpl)

		form := url.Values{}
		form.Set("name", "Test")
		form.Set("icon", "💰")
		form.Set("colour", "nope")

		req := httptest.NewRequest(http.MethodPost, "/accounts", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("got status %d, want %d", w.Code, http.StatusBadRequest)
		}
	})
}

func TestHandleAccountDetail(t *testing.T) {
	t.Run("shows transactions", func(t *testing.T) {
		stub := &stubQuerier{
			account: sqlc.PocketsAccount{ID: 1, Name: "Savings", Icon: "💰", Colour: "steel"},
			balance: makeNumeric("500.00"),
			transactions: []sqlc.PocketsTransaction{
				{
					ID:       1,
					Amount:   makeNumeric("100.00"),
					IsInflow: false,
					TxDate:   pgtype.Date{Time: time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC), Valid: true},
				},
			},
			autoTopupErr: pgx.ErrNoRows,
		}
		tmpl := testTemplates(t)
		handler := handleAccountDetail(stub, tmpl)

		req := httptest.NewRequest(http.MethodGet, "/accounts/1", nil)
		req.SetPathValue("id", "1")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("got status %d, want %d", w.Code, http.StatusOK)
		}
		body := w.Body.String()
		if !strings.Contains(body, "500,00") {
			t.Error("expected balance 500,00")
		}
		if !strings.Contains(body, "-100,00") {
			t.Error("expected outflow -100,00")
		}
	})

	t.Run("filter=withdrawals shows only outflows", func(t *testing.T) {
		stub := &stubQuerier{
			account: sqlc.PocketsAccount{ID: 1, Name: "Savings", Icon: "💰", Colour: "steel"},
			balance: makeNumeric("500.00"),
			transactions: []sqlc.PocketsTransaction{
				{
					ID:       2,
					Amount:   makeNumeric("50.00"),
					IsInflow: false,
					TxDate:   pgtype.Date{Time: time.Date(2026, 3, 5, 0, 0, 0, 0, time.UTC), Valid: true},
				},
			},
			autoTopupErr: pgx.ErrNoRows,
		}
		tmpl := testTemplates(t)
		handler := handleAccountDetail(stub, tmpl)

		req := httptest.NewRequest(http.MethodGet, "/accounts/1?filter=withdrawals", nil)
		req.SetPathValue("id", "1")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("got status %d, want %d", w.Code, http.StatusOK)
		}
		body := w.Body.String()
		if !strings.Contains(body, `aria-current="page"`) {
			t.Error("expected active filter tab")
		}
	})
}

// --- Transaction handler tests ---

func TestHandleNewTransaction(t *testing.T) {
	t.Run("general form shows account select", func(t *testing.T) {
		stub := &stubQuerier{
			accounts: []sqlc.PocketsAccount{
				{ID: 1, Name: "Savings", Icon: "💰", IsReserve: true},
				{ID: 2, Name: "Travel", Icon: "✈️"},
			},
		}
		tmpl := testTemplates(t)
		handler := handleNewTransaction(stub, tmpl)

		req := httptest.NewRequest(http.MethodGet, "/transactions/new", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("got status %d, want %d", w.Code, http.StatusOK)
		}
		body := w.Body.String()
		if !strings.Contains(body, "account_id") {
			t.Error("expected account selector")
		}
		if !strings.Contains(body, `action="/transactions"`) {
			t.Error("expected form action /transactions")
		}
	})

	t.Run("account-specific form hides account select", func(t *testing.T) {
		stub := &stubQuerier{
			account: sqlc.PocketsAccount{ID: 1, Name: "Savings", Icon: "💰"},
		}
		tmpl := testTemplates(t)
		handler := handleNewTransaction(stub, tmpl)

		req := httptest.NewRequest(http.MethodGet, "/accounts/1/transactions/new", nil)
		req.SetPathValue("id", "1")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("got status %d, want %d", w.Code, http.StatusOK)
		}
		body := w.Body.String()
		if strings.Contains(body, "account_id") {
			t.Error("should not show account selector for account-specific form")
		}
	})
}

func TestHandleCreateTransaction(t *testing.T) {
	t.Run("valid transaction redirects", func(t *testing.T) {
		stub := &stubQuerier{createdID: 1}
		handler := handleCreateTransaction(stub)

		form := url.Values{}
		form.Set("amount", "50.00")
		form.Set("direction", "out")
		form.Set("tx_date", "2026-03-15")

		req := httptest.NewRequest(http.MethodPost, "/accounts/1/transactions", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.SetPathValue("id", "1")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusSeeOther {
			t.Fatalf("got status %d, want %d", w.Code, http.StatusSeeOther)
		}
		if loc := w.Header().Get("Location"); loc != "/accounts/1" {
			t.Errorf("got location %q, want /accounts/1", loc)
		}
		if len(stub.createTxnCalls) != 1 {
			t.Fatalf("expected 1 call, got %d", len(stub.createTxnCalls))
		}
		if stub.createTxnCalls[0].IsInflow {
			t.Error("direction=out should set IsInflow=false")
		}
	})

	t.Run("direction in sets inflow", func(t *testing.T) {
		stub := &stubQuerier{createdID: 1}
		handler := handleCreateTransaction(stub)

		form := url.Values{}
		form.Set("amount", "100.00")
		form.Set("direction", "in")
		form.Set("tx_date", "2026-03-15")

		req := httptest.NewRequest(http.MethodPost, "/accounts/1/transactions", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.SetPathValue("id", "1")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusSeeOther {
			t.Fatalf("got status %d, want %d", w.Code, http.StatusSeeOther)
		}
		if !stub.createTxnCalls[0].IsInflow {
			t.Error("direction=in should set IsInflow=true")
		}
	})

	t.Run("missing amount returns 400", func(t *testing.T) {
		stub := &stubQuerier{createdID: 1}
		handler := handleCreateTransaction(stub)

		form := url.Values{}
		form.Set("direction", "out")
		form.Set("tx_date", "2026-03-15")

		req := httptest.NewRequest(http.MethodPost, "/accounts/1/transactions", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.SetPathValue("id", "1")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("got status %d, want %d", w.Code, http.StatusBadRequest)
		}
	})

	t.Run("general form with account_id", func(t *testing.T) {
		stub := &stubQuerier{createdID: 1}
		handler := handleCreateTransaction(stub)

		form := url.Values{}
		form.Set("account_id", "5")
		form.Set("amount", "75.00")
		form.Set("direction", "out")
		form.Set("tx_date", "2026-03-15")

		req := httptest.NewRequest(http.MethodPost, "/transactions", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusSeeOther {
			t.Fatalf("got status %d, want %d", w.Code, http.StatusSeeOther)
		}
		if stub.createTxnCalls[0].AccountID != 5 {
			t.Errorf("expected account_id 5, got %d", stub.createTxnCalls[0].AccountID)
		}
	})
}

func TestHandleEditTransaction(t *testing.T) {
	t.Run("GET shows edit form", func(t *testing.T) {
		stub := &stubQuerier{
			transaction: sqlc.PocketsTransaction{
				ID:        10,
				AccountID: 1,
				Amount:    makeNumeric("50.00"),
				IsInflow:  false,
				TxDate:    pgtype.Date{Time: time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC), Valid: true},
			},
		}
		tmpl := testTemplates(t)
		handler := handleEditTransaction(stub, tmpl)

		req := httptest.NewRequest(http.MethodGet, "/accounts/1/transactions/10/edit", nil)
		req.SetPathValue("id", "1")
		req.SetPathValue("tid", "10")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("got status %d, want %d", w.Code, http.StatusOK)
		}
		body := w.Body.String()
		if !strings.Contains(body, "50,00") {
			t.Error("expected amount 50,00")
		}
	})

	t.Run("POST updates and redirects", func(t *testing.T) {
		stub := &stubQuerier{
			transaction: sqlc.PocketsTransaction{
				ID:        10,
				AccountID: 1,
				Amount:    makeNumeric("50.00"),
				IsInflow:  false,
				TxDate:    pgtype.Date{Time: time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC), Valid: true},
			},
		}
		tmpl := testTemplates(t)
		handler := handleEditTransaction(stub, tmpl)

		form := url.Values{}
		form.Set("amount", "75.00")
		form.Set("direction", "out")
		form.Set("tx_date", "2026-03-15")

		req := httptest.NewRequest(http.MethodPost, "/accounts/1/transactions/10/edit", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.SetPathValue("id", "1")
		req.SetPathValue("tid", "10")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusSeeOther {
			t.Fatalf("got status %d, want %d", w.Code, http.StatusSeeOther)
		}
	})

	t.Run("editing auto-topup sets user_edited", func(t *testing.T) {
		stub := &stubQuerier{
			transaction: sqlc.PocketsTransaction{
				ID:          10,
				AccountID:   1,
				Amount:      makeNumeric("100.00"),
				IsInflow:    true,
				IsAutoTopup: true,
				TxDate:      pgtype.Date{Time: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC), Valid: true},
			},
		}
		tmpl := testTemplates(t)
		handler := handleEditTransaction(stub, tmpl)

		form := url.Values{}
		form.Set("amount", "150.00")
		form.Set("direction", "in")
		form.Set("tx_date", "2026-03-01")

		req := httptest.NewRequest(http.MethodPost, "/accounts/1/transactions/10/edit", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.SetPathValue("id", "1")
		req.SetPathValue("tid", "10")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusSeeOther {
			t.Fatalf("got status %d, want %d", w.Code, http.StatusSeeOther)
		}
		if stub.updateTxnCall == nil {
			t.Fatal("expected UpdateTransaction call")
		}
		if !stub.updateTxnCall.UserEdited {
			t.Error("editing auto-topup should set user_edited=true")
		}
	})

	t.Run("wrong account returns 404", func(t *testing.T) {
		stub := &stubQuerier{
			transaction: sqlc.PocketsTransaction{
				ID:        10,
				AccountID: 2, // different from path
			},
		}
		tmpl := testTemplates(t)
		handler := handleEditTransaction(stub, tmpl)

		req := httptest.NewRequest(http.MethodGet, "/accounts/1/transactions/10/edit", nil)
		req.SetPathValue("id", "1")
		req.SetPathValue("tid", "10")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("got status %d, want %d", w.Code, http.StatusNotFound)
		}
	})
}

// --- Top-up rule tests ---

func TestHandleTopupRules(t *testing.T) {
	stub := &stubQuerier{
		account: sqlc.PocketsAccount{ID: 1, Name: "Savings"},
		topupRules: []sqlc.PocketsTopupRule{
			{ID: 1, AccountID: 1, Amount: makeNumeric("200.00"), EffectiveDate: pgtype.Date{Time: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), Valid: true}},
		},
	}
	tmpl := testTemplates(t)
	handler := handleTopupRules(stub, tmpl)

	req := httptest.NewRequest(http.MethodGet, "/accounts/1/topups", nil)
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("got status %d, want %d", w.Code, http.StatusOK)
	}
	body := w.Body.String()
	if !strings.Contains(body, "200,00") {
		t.Error("expected rule amount 200,00")
	}
}

func TestHandleCreateTopupRule(t *testing.T) {
	t.Run("valid rule redirects", func(t *testing.T) {
		stub := &stubQuerier{createdID: 1}
		handler := handleCreateTopupRule(stub)

		form := url.Values{}
		form.Set("amount", "150.00")
		form.Set("effective_date", "2026-04-01")

		req := httptest.NewRequest(http.MethodPost, "/accounts/1/topups", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.SetPathValue("id", "1")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusSeeOther {
			t.Fatalf("got status %d, want %d", w.Code, http.StatusSeeOther)
		}
	})

	t.Run("missing amount returns 400", func(t *testing.T) {
		stub := &stubQuerier{createdID: 1}
		handler := handleCreateTopupRule(stub)

		form := url.Values{}
		form.Set("effective_date", "2026-04-01")

		req := httptest.NewRequest(http.MethodPost, "/accounts/1/topups", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.SetPathValue("id", "1")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("got status %d, want %d", w.Code, http.StatusBadRequest)
		}
	})
}

func TestHandleDeleteTopupRule(t *testing.T) {
	stub := &stubQuerier{}
	handler := handleDeleteTopupRule(stub)

	req := httptest.NewRequest(http.MethodPost, "/accounts/1/topups/5/delete", nil)
	req.SetPathValue("id", "1")
	req.SetPathValue("rid", "5")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("got status %d, want %d", w.Code, http.StatusSeeOther)
	}
	if loc := w.Header().Get("Location"); loc != "/accounts/1/topups" {
		t.Errorf("got location %q, want /accounts/1/topups", loc)
	}
}

// --- ensureAutoTopups tests ---

func TestEnsureAutoTopups(t *testing.T) {
	t.Run("no rules creates nothing", func(t *testing.T) {
		stub := &stubQuerier{}
		err := ensureAutoTopups(context.Background(), stub, 1, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(stub.createTxnCalls) != 0 {
			t.Errorf("expected 0 calls, got %d", len(stub.createTxnCalls))
		}
	})

	t.Run("future rule creates nothing", func(t *testing.T) {
		stub := &stubQuerier{autoTopupErr: pgx.ErrNoRows}
		rules := []sqlc.PocketsTopupRule{
			{ID: 1, AccountID: 1, Amount: makeNumeric("100.00"), EffectiveDate: pgtype.Date{Time: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC), Valid: true}},
		}
		err := ensureAutoTopups(context.Background(), stub, 1, rules)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(stub.createTxnCalls) != 0 {
			t.Errorf("expected 0 calls, got %d", len(stub.createTxnCalls))
		}
	})

	t.Run("past rule creates transactions for each month", func(t *testing.T) {
		stub := &stubQuerier{autoTopupErr: pgx.ErrNoRows, createdID: 1}
		rules := []sqlc.PocketsTopupRule{
			{ID: 1, AccountID: 1, Amount: makeNumeric("100.00"), EffectiveDate: pgtype.Date{Time: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), Valid: true}},
		}
		// timeNow returns 2026-03-15, so should create for Jan, Feb, Mar
		err := ensureAutoTopups(context.Background(), stub, 1, rules)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(stub.createTxnCalls) != 3 {
			t.Fatalf("expected 3 calls (Jan, Feb, Mar), got %d", len(stub.createTxnCalls))
		}
		// Verify dates
		expected := []time.Time{
			time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
			time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		}
		for i, call := range stub.createTxnCalls {
			if !call.TxDate.Time.Equal(expected[i]) {
				t.Errorf("call %d: got date %v, want %v", i, call.TxDate.Time, expected[i])
			}
			if !call.IsInflow {
				t.Errorf("call %d: expected IsInflow=true", i)
			}
			if !call.IsAutoTopup {
				t.Errorf("call %d: expected IsAutoTopup=true", i)
			}
		}
	})

	t.Run("existing not user-edited with different amount gets updated", func(t *testing.T) {
		stub := &stubQuerier{
			autoTopup: sqlc.PocketsTransaction{
				ID:          5,
				AccountID:   1,
				Amount:      makeNumeric("80.00"),
				IsAutoTopup: true,
				UserEdited:  false,
			},
		}
		rules := []sqlc.PocketsTopupRule{
			{ID: 1, AccountID: 1, Amount: makeNumeric("100.00"), EffectiveDate: pgtype.Date{Time: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC), Valid: true}},
		}
		err := ensureAutoTopups(context.Background(), stub, 1, rules)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if stub.updateAutoTopupCall == nil {
			t.Fatal("expected UpdateAutoTopupAmount call")
		}
		if stub.updateAutoTopupCall.ID != 5 {
			t.Errorf("expected ID 5, got %d", stub.updateAutoTopupCall.ID)
		}
	})

	t.Run("existing user-edited is not updated", func(t *testing.T) {
		stub := &stubQuerier{
			autoTopup: sqlc.PocketsTransaction{
				ID:          5,
				AccountID:   1,
				Amount:      makeNumeric("80.00"),
				IsAutoTopup: true,
				UserEdited:  true,
			},
		}
		rules := []sqlc.PocketsTopupRule{
			{ID: 1, AccountID: 1, Amount: makeNumeric("100.00"), EffectiveDate: pgtype.Date{Time: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC), Valid: true}},
		}
		err := ensureAutoTopups(context.Background(), stub, 1, rules)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if stub.updateAutoTopupCall != nil {
			t.Error("should NOT update user-edited auto-topup")
		}
	})

	t.Run("multiple rules use correct amounts per period", func(t *testing.T) {
		stub := &stubQuerier{autoTopupErr: pgx.ErrNoRows, createdID: 1}
		rules := []sqlc.PocketsTopupRule{
			{ID: 1, AccountID: 1, Amount: makeNumeric("100.00"), EffectiveDate: pgtype.Date{Time: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), Valid: true}},
			{ID: 2, AccountID: 1, Amount: makeNumeric("200.00"), EffectiveDate: pgtype.Date{Time: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC), Valid: true}},
		}
		err := ensureAutoTopups(context.Background(), stub, 1, rules)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(stub.createTxnCalls) != 3 {
			t.Fatalf("expected 3 calls, got %d", len(stub.createTxnCalls))
		}
		// Jan and Feb should use rule 1 (100), Mar should use rule 2 (200)
		janAmt := numericToFloat64(stub.createTxnCalls[0].Amount)
		if janAmt != 100.00 {
			t.Errorf("Jan amount = %.2f, want 100.00", janAmt)
		}
		febAmt := numericToFloat64(stub.createTxnCalls[1].Amount)
		if febAmt != 100.00 {
			t.Errorf("Feb amount = %.2f, want 100.00", febAmt)
		}
		marAmt := numericToFloat64(stub.createTxnCalls[2].Amount)
		if marAmt != 200.00 {
			t.Errorf("Mar amount = %.2f, want 200.00", marAmt)
		}
	})
}

// --- Forecast tests ---

func TestComputeForecast(t *testing.T) {
	t.Run("no rules flat balance", func(t *testing.T) {
		rows := computeForecast(1000, 0, nil, 6, nil)
		if len(rows) != 6 {
			t.Fatalf("expected 6 rows, got %d", len(rows))
		}
		for _, r := range rows {
			if r.Balance != "1 000,00" {
				t.Errorf("expected 1 000,00, got %s", r.Balance)
			}
		}
	})

	t.Run("with rule adds monthly", func(t *testing.T) {
		rules := []sqlc.PocketsTopupRule{
			{ID: 1, Amount: makeNumeric("100.00"), EffectiveDate: pgtype.Date{Time: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), Valid: true}},
		}
		rows := computeForecast(500, 2000, rules, 3, nil)
		if len(rows) != 3 {
			t.Fatalf("expected 3 rows, got %d", len(rows))
		}
		// Month 1: 500+100=600, Month 2: 700, Month 3: 800
		expected := []string{"600,00", "700,00", "800,00"}
		for i, r := range rows {
			if r.Balance != expected[i] {
				t.Errorf("row %d: got %s, want %s", i, r.Balance, expected[i])
			}
		}
		// Check target percent for month 1: 600/2000 = 30%
		if rows[0].TargetPercent != 30 {
			t.Errorf("row 0 target percent = %d, want 30", rows[0].TargetPercent)
		}
	})

	t.Run("rule change mid-forecast", func(t *testing.T) {
		// Current month is March 2026, forecast months 4-6 (Apr, May, Jun)
		rules := []sqlc.PocketsTopupRule{
			{ID: 1, Amount: makeNumeric("100.00"), EffectiveDate: pgtype.Date{Time: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), Valid: true}},
			{ID: 2, Amount: makeNumeric("200.00"), EffectiveDate: pgtype.Date{Time: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC), Valid: true}},
		}
		rows := computeForecast(0, 0, rules, 6, nil)
		// Months: Apr(+100=100), May(+100=200), Jun(+200=400), Jul(+200=600), Aug(+200=800), Sep(+200=1000)
		// Wait - need to check: forecast starts from "next month" relative to now (March 2026)
		// Month 1 = April, Month 2 = May, Month 3 = June (rule 2 applies), etc.
		if rows[0].Balance != "100,00" {
			t.Errorf("Apr: got %s, want 100,00", rows[0].Balance)
		}
		if rows[1].Balance != "200,00" {
			t.Errorf("May: got %s, want 200,00", rows[1].Balance)
		}
		if rows[2].Balance != "400,00" {
			t.Errorf("Jun: got %s, want 400,00", rows[2].Balance)
		}
	})

	t.Run("future withdrawal reduces forecast balance", func(t *testing.T) {
		// Now is March 15 2026. Balance=1000, topup 100/month, future outflow 300 in May.
		rules := []sqlc.PocketsTopupRule{
			{ID: 1, Amount: makeNumeric("100.00"), EffectiveDate: pgtype.Date{Time: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), Valid: true}},
		}
		futureTxns := []sqlc.PocketsTransaction{
			{ID: 1, Amount: makeNumeric("300.00"), IsInflow: false, TxDate: pgtype.Date{Time: time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC), Valid: true}},
		}
		rows := computeForecast(1000, 0, rules, 6, futureTxns)
		// Apr: 1000+100=1100, May: 1100+100-300=900, Jun: 900+100=1000,
		// Jul: 1100, Aug: 1200, Sep: 1300
		expected := []string{"1 100,00", "900,00", "1 000,00", "1 100,00", "1 200,00", "1 300,00"}
		for i, r := range rows {
			if r.Balance != expected[i] {
				t.Errorf("row %d: got %s, want %s", i, r.Balance, expected[i])
			}
		}
	})

	t.Run("future inflow in forecast", func(t *testing.T) {
		// Balance=500, no rules, future inflow 200 in June (month 3).
		futureTxns := []sqlc.PocketsTransaction{
			{ID: 1, Amount: makeNumeric("200.00"), IsInflow: true, TxDate: pgtype.Date{Time: time.Date(2026, 6, 5, 0, 0, 0, 0, time.UTC), Valid: true}},
		}
		rows := computeForecast(500, 0, nil, 6, futureTxns)
		// Apr=500, May=500, Jun=700, Jul=700, Aug=700, Sep=700
		expected := []string{"500,00", "500,00", "700,00", "700,00", "700,00", "700,00"}
		for i, r := range rows {
			if r.Balance != expected[i] {
				t.Errorf("row %d: got %s, want %s", i, r.Balance, expected[i])
			}
		}
	})

	t.Run("multiple future txns in same month", func(t *testing.T) {
		futureTxns := []sqlc.PocketsTransaction{
			{ID: 1, Amount: makeNumeric("100.00"), IsInflow: true, TxDate: pgtype.Date{Time: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC), Valid: true}},
			{ID: 2, Amount: makeNumeric("50.00"), IsInflow: false, TxDate: pgtype.Date{Time: time.Date(2026, 5, 15, 0, 0, 0, 0, time.UTC), Valid: true}},
		}
		rows := computeForecast(1000, 0, nil, 3, futureTxns)
		// Apr=1000, May=1000+100-50=1050, Jun=1050
		expected := []string{"1 000,00", "1 050,00", "1 050,00"}
		for i, r := range rows {
			if r.Balance != expected[i] {
				t.Errorf("row %d: got %s, want %s", i, r.Balance, expected[i])
			}
		}
	})

	t.Run("current-month future txn adjusts starting balance", func(t *testing.T) {
		// Now is March 15. A txn dated March 20 is future but in current month.
		// It should be applied to the starting balance before the month loop.
		futureTxns := []sqlc.PocketsTransaction{
			{ID: 1, Amount: makeNumeric("200.00"), IsInflow: false, TxDate: pgtype.Date{Time: time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC), Valid: true}},
		}
		rows := computeForecast(1000, 0, nil, 3, futureTxns)
		// Starting balance adjusted: 1000-200=800
		// Apr=800, May=800, Jun=800
		expected := []string{"800,00", "800,00", "800,00"}
		for i, r := range rows {
			if r.Balance != expected[i] {
				t.Errorf("row %d: got %s, want %s", i, r.Balance, expected[i])
			}
		}
	})
}

func TestHandleAccountForecast(t *testing.T) {
	t.Run("basic forecast", func(t *testing.T) {
		stub := &stubQuerier{
			account: sqlc.PocketsAccount{ID: 1, Name: "Savings", Icon: "💰", Colour: "steel"},
			balance: makeNumeric("1000.00"),
		}
		tmpl := testTemplates(t)
		handler := handleAccountForecast(stub, tmpl)

		req := httptest.NewRequest(http.MethodGet, "/accounts/1/forecast?months=6", nil)
		req.SetPathValue("id", "1")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("got status %d, want %d", w.Code, http.StatusOK)
		}
		body := w.Body.String()
		if !strings.Contains(body, "Forecast") {
			t.Error("expected forecast heading")
		}
		if !strings.Contains(body, "1 000,00") {
			t.Error("expected balance in forecast")
		}
	})

	t.Run("future transactions affect forecast", func(t *testing.T) {
		stub := &stubQuerier{
			account: sqlc.PocketsAccount{ID: 1, Name: "Savings", Icon: "💰", Colour: "steel"},
			balance: makeNumeric("1000.00"),
			topupRules: []sqlc.PocketsTopupRule{
				{ID: 1, Amount: makeNumeric("100.00"), EffectiveDate: pgtype.Date{Time: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), Valid: true}},
			},
			futureTransactions: []sqlc.PocketsTransaction{
				{ID: 1, Amount: makeNumeric("500.00"), IsInflow: false, TxDate: pgtype.Date{Time: time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC), Valid: true}},
			},
			autoTopupErr: pgx.ErrNoRows,
		}
		tmpl := testTemplates(t)
		handler := handleAccountForecast(stub, tmpl)

		req := httptest.NewRequest(http.MethodGet, "/accounts/1/forecast?months=6", nil)
		req.SetPathValue("id", "1")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("got status %d, want %d", w.Code, http.StatusOK)
		}
		body := w.Body.String()
		// May should show 1000+100+100-500=700, not 1200
		if !strings.Contains(body, "700,00") {
			t.Error("expected forecast to reflect future withdrawal (700,00)")
		}
	})
}

func TestHandleAllForecast(t *testing.T) {
	t.Run("basic all-accounts forecast", func(t *testing.T) {
		stub := &stubQuerier{
			accounts: []sqlc.PocketsAccount{
				{ID: 1, Name: "Savings", Icon: "💰", Colour: "steel"},
			},
			balance: makeNumeric("500.00"),
		}
		tmpl := testTemplates(t)
		handler := handleAllForecast(stub, tmpl)

		req := httptest.NewRequest(http.MethodGet, "/forecast?months=6", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("got status %d, want %d", w.Code, http.StatusOK)
		}
		body := w.Body.String()
		if !strings.Contains(body, "All Accounts") {
			t.Error("expected all accounts heading")
		}
	})

	t.Run("multiple accounts with different balances and future txns", func(t *testing.T) {
		stub := &stubQuerier{
			accounts: []sqlc.PocketsAccount{
				{ID: 1, Name: "Savings", Icon: "💰", Colour: "steel"},
				{ID: 2, Name: "Travel", Icon: "✈️", Colour: "teal"},
			},
			balancesByAccount: map[int64]pgtype.Numeric{
				1: makeNumeric("1000.00"),
				2: makeNumeric("500.00"),
			},
			topupRulesByAccount: map[int64][]sqlc.PocketsTopupRule{
				1: {{ID: 1, Amount: makeNumeric("100.00"), EffectiveDate: pgtype.Date{Time: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), Valid: true}}},
				2: {},
			},
			futureTransactionsByAcct: map[int64][]sqlc.PocketsTransaction{
				1: {},
				2: {{ID: 1, Amount: makeNumeric("200.00"), IsInflow: false, TxDate: pgtype.Date{Time: time.Date(2026, 4, 5, 0, 0, 0, 0, time.UTC), Valid: true}}},
			},
			autoTopupErr: pgx.ErrNoRows,
		}
		tmpl := testTemplates(t)
		handler := handleAllForecast(stub, tmpl)

		req := httptest.NewRequest(http.MethodGet, "/forecast?months=6", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("got status %d, want %d", w.Code, http.StatusOK)
		}
		body := w.Body.String()
		// Account 1 Apr: 1000+100=1100
		if !strings.Contains(body, "1 100,00") {
			t.Error("expected Savings forecast with topup (1 100,00)")
		}
		// Account 2 Apr: 500-200=300
		if !strings.Contains(body, "300,00") {
			t.Error("expected Travel forecast with future withdrawal (300,00)")
		}
	})
}

// --- Edit account tests ---

func TestHandleEditAccount(t *testing.T) {
	t.Run("GET shows edit form", func(t *testing.T) {
		stub := &stubQuerier{
			account: sqlc.PocketsAccount{ID: 1, Name: "Savings", Icon: "💰", Colour: "steel", IsReserve: true},
		}
		tmpl := testTemplates(t)
		handler := handleEditAccount(stub, tmpl)

		req := httptest.NewRequest(http.MethodGet, "/accounts/1/edit", nil)
		req.SetPathValue("id", "1")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("got status %d, want %d", w.Code, http.StatusOK)
		}
		body := w.Body.String()
		if !strings.Contains(body, "Savings") {
			t.Error("expected account name")
		}
	})

	t.Run("POST updates and redirects", func(t *testing.T) {
		stub := &stubQuerier{
			account: sqlc.PocketsAccount{ID: 1, Name: "Savings", Icon: "💰", Colour: "steel"},
		}
		tmpl := testTemplates(t)
		handler := handleEditAccount(stub, tmpl)

		form := url.Values{}
		form.Set("name", "New Name")
		form.Set("icon", "🏦")
		form.Set("colour", "ruby")

		req := httptest.NewRequest(http.MethodPost, "/accounts/1/edit", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.SetPathValue("id", "1")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusSeeOther {
			t.Fatalf("got status %d, want %d", w.Code, http.StatusSeeOther)
		}
		if loc := w.Header().Get("Location"); loc != "/accounts/1" {
			t.Errorf("got location %q, want /accounts/1", loc)
		}
	})
}
