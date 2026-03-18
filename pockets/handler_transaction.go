// Cellarium Pockets — transaction handlers
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
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/maroskucera/cellarium/pockets/db/sqlc"
)

type accountOption struct {
	ID       int64
	Name     string
	Icon     string
	Selected bool
}

type newTxnData struct {
	FormAction        string
	ShowAccountSelect bool
	Accounts          []accountOption
	TodayStr          string
}

type editTxnData struct {
	AccountID   int64
	Transaction struct {
		ID       int64
		Amount   string
		IsInflow bool
		Date     string
		Note     string
	}
	Nav accountNavData
}

func handleNewTransaction(q sqlc.Querier, tmpl *template.Template) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		idStr := r.PathValue("id")
		ctx := r.Context()

		data := newTxnData{
			TodayStr: timeNow().Format("2006-01-02"),
		}

		if idStr != "" {
			// Account-specific: /accounts/{id}/transactions/new
			id, err := strconv.ParseInt(idStr, 10, 64)
			if err != nil {
				http.Error(w, "invalid account id", http.StatusBadRequest)
				return
			}
			_, err = q.GetAccount(ctx, id)
			if err != nil {
				http.Error(w, "account not found", http.StatusNotFound)
				return
			}
			data.FormAction = fmt.Sprintf("/accounts/%d/transactions", id)
			data.ShowAccountSelect = false
		} else {
			// General: /transactions/new
			accounts, err := q.ListAccounts(ctx)
			if err != nil {
				http.Error(w, "database error", http.StatusInternalServerError)
				return
			}

			// Find first reserve account for pre-selection
			var firstReserveID int64
			for _, a := range accounts {
				if a.IsReserve {
					firstReserveID = a.ID
					break
				}
			}
			if firstReserveID == 0 && len(accounts) > 0 {
				firstReserveID = accounts[0].ID
			}

			var options []accountOption
			for _, a := range accounts {
				options = append(options, accountOption{
					ID:       a.ID,
					Name:     a.Name,
					Icon:     a.Icon,
					Selected: a.ID == firstReserveID,
				})
			}
			data.FormAction = "/transactions"
			data.ShowAccountSelect = true
			data.Accounts = options
		}

		w.Header().Set("Cache-Control", "no-store")
		renderTemplate(w, tmpl, "transaction_new", data)
	})
}

func createTransaction(q sqlc.Querier, w http.ResponseWriter, r *http.Request, accountID int64) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form data", http.StatusBadRequest)
		return
	}

	amountStr := r.FormValue("amount")
	if amountStr == "" {
		http.Error(w, "amount is required", http.StatusBadRequest)
		return
	}
	amount, err := parseAmount(amountStr)
	if err != nil || amount <= 0 {
		http.Error(w, "amount must be a positive number", http.StatusBadRequest)
		return
	}

	direction := r.FormValue("direction")
	isInflow := direction == "in"

	dateStr := r.FormValue("tx_date")
	if dateStr == "" {
		http.Error(w, "date is required", http.StatusBadRequest)
		return
	}
	txDate, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		http.Error(w, "invalid date format", http.StatusBadRequest)
		return
	}

	var note pgtype.Text
	if n := r.FormValue("note"); n != "" {
		note = pgtype.Text{String: n, Valid: true}
	}

	_, err = q.CreateTransaction(r.Context(), sqlc.CreateTransactionParams{
		AccountID: accountID,
		Amount:    float64ToNumeric(amount),
		IsInflow:  isInflow,
		TxDate:    pgtype.Date{Time: txDate, Valid: true},
		Note:      note,
	})
	if err != nil {
		http.Error(w, "failed to create transaction", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/accounts/%d", accountID), http.StatusSeeOther)
}

func handleCreateTransaction(q sqlc.Querier) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		idStr := r.PathValue("id")
		if idStr != "" {
			// Account-specific: /accounts/{id}/transactions
			id, err := strconv.ParseInt(idStr, 10, 64)
			if err != nil {
				http.Error(w, "invalid account id", http.StatusBadRequest)
				return
			}
			createTransaction(q, w, r, id)
			return
		}

		// General: /transactions — get account_id from form
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form data", http.StatusBadRequest)
			return
		}
		accountIDStr := r.FormValue("account_id")
		if accountIDStr == "" {
			http.Error(w, "account is required", http.StatusBadRequest)
			return
		}
		accountID, err := strconv.ParseInt(accountIDStr, 10, 64)
		if err != nil {
			http.Error(w, "invalid account id", http.StatusBadRequest)
			return
		}
		createTransaction(q, w, r, accountID)
	})
}

func handleEditTransaction(q sqlc.Querier, tmpl *template.Template) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		accountID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			http.Error(w, "invalid account id", http.StatusBadRequest)
			return
		}

		txnID, err := strconv.ParseInt(r.PathValue("tid"), 10, 64)
		if err != nil {
			http.Error(w, "invalid transaction id", http.StatusBadRequest)
			return
		}

		ctx := r.Context()
		txn, err := q.GetTransaction(ctx, txnID)
		if err != nil {
			http.Error(w, "transaction not found", http.StatusNotFound)
			return
		}

		if txn.AccountID != accountID {
			http.Error(w, "transaction not found", http.StatusNotFound)
			return
		}

		if r.Method == http.MethodPost {
			if err := r.ParseForm(); err != nil {
				http.Error(w, "invalid form data", http.StatusBadRequest)
				return
			}

			amountStr := r.FormValue("amount")
			if amountStr == "" {
				http.Error(w, "amount is required", http.StatusBadRequest)
				return
			}
			amount, err := parseAmount(amountStr)
			if err != nil || amount <= 0 {
				http.Error(w, "amount must be a positive number", http.StatusBadRequest)
				return
			}

			direction := r.FormValue("direction")
			isInflow := direction == "in"

			dateStr := r.FormValue("tx_date")
			if dateStr == "" {
				http.Error(w, "date is required", http.StatusBadRequest)
				return
			}
			txDate, err := time.Parse("2006-01-02", dateStr)
			if err != nil {
				http.Error(w, "invalid date format", http.StatusBadRequest)
				return
			}

			var note pgtype.Text
			if n := r.FormValue("note"); n != "" {
				note = pgtype.Text{String: n, Valid: true}
			}

			userEdited := txn.UserEdited || txn.IsAutoTopup
			err = q.UpdateTransaction(ctx, sqlc.UpdateTransactionParams{
				ID:         txnID,
				Amount:     float64ToNumeric(amount),
				IsInflow:   isInflow,
				TxDate:     pgtype.Date{Time: txDate, Valid: true},
				Note:       note,
				UserEdited: userEdited,
			})
			if err != nil {
				http.Error(w, "failed to update transaction", http.StatusInternalServerError)
				return
			}

			http.Redirect(w, r, fmt.Sprintf("/accounts/%d", accountID), http.StatusSeeOther)
			return
		}

		// GET: show edit form
		data := editTxnData{
			AccountID: accountID,
			Nav:       accountNavData{AccountID: accountID, Active: "transactions"},
		}
		data.Transaction.ID = txn.ID
		data.Transaction.Amount = formatAmount(numericToFloat64(txn.Amount))
		data.Transaction.IsInflow = txn.IsInflow
		data.Transaction.Date = txn.TxDate.Time.Format("2006-01-02")
		if txn.Note.Valid {
			data.Transaction.Note = txn.Note.String
		}

		w.Header().Set("Cache-Control", "no-store")
		renderTemplate(w, tmpl, "transaction_edit", data)
	})
}
