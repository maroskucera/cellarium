// Cellarium Pockets — account handlers
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
	"math"
	"net/http"
	"strconv"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/maroskucera/cellarium/pockets/db/sqlc"
)

type accountNavData struct {
	AccountID int64
	Active    string
}

type accountFormData struct {
	Colours []colour
}

type accountEditData struct {
	Account struct {
		ID           int64
		Name         string
		Icon         string
		Colour       string
		TargetAmount string
		IsReserve    bool
	}
	Colours []colour
	Nav     accountNavData
}

type txnDisplay struct {
	ID            int64
	Date          string
	DisplayAmount string
	IsInflow      bool
	Note          string
}

type accountDetailData struct {
	Account struct {
		ID           int64
		Name         string
		Icon         string
		ColourHex    string
		HasTarget    bool
		TargetAmount string
	}
	Balance       string
	TargetPercent int
	Filter        string
	Transactions  []txnDisplay
	Nav           accountNavData
}

func handleNewAccount(tmpl *template.Template) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		renderTemplate(w, tmpl, "account_new", accountFormData{
			Colours: allColours(),
		})
	})
}

func handleCreateAccount(q sqlc.Querier, tmpl *template.Template) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form data", http.StatusBadRequest)
			return
		}

		name := r.FormValue("name")
		if name == "" {
			http.Error(w, "name is required", http.StatusBadRequest)
			return
		}

		icon := r.FormValue("icon")
		if icon == "" {
			http.Error(w, "icon is required", http.StatusBadRequest)
			return
		}

		colourKey := r.FormValue("colour")
		if !validColour(colourKey) {
			http.Error(w, "invalid colour", http.StatusBadRequest)
			return
		}

		var targetAmount pgtype.Numeric
		if s := r.FormValue("target_amount"); s != "" {
			v, err := parseAmount(s)
			if err != nil {
				http.Error(w, "invalid target amount", http.StatusBadRequest)
				return
			}
			targetAmount = float64ToNumeric(v)
		}

		isReserve := r.FormValue("is_reserve") == "true"

		ctx := r.Context()
		accountID, err := q.CreateAccount(ctx, sqlc.CreateAccountParams{
			Name:         name,
			Icon:         icon,
			Colour:       colourKey,
			TargetAmount: targetAmount,
			IsReserve:    isReserve,
		})
		if err != nil {
			http.Error(w, "failed to create account", http.StatusInternalServerError)
			return
		}

		// Create initial balance transaction if provided
		if s := r.FormValue("initial_balance"); s != "" {
			v, err := parseAmount(s)
			if err != nil {
				http.Error(w, "invalid initial balance", http.StatusBadRequest)
				return
			}
			if v > 0 {
				_, err = q.CreateTransaction(ctx, sqlc.CreateTransactionParams{
					AccountID:        accountID,
					Amount:           float64ToNumeric(v),
					IsInflow:         true,
					TxDate:           pgtype.Date{Valid: true, Time: timeNow()},
					IsInitialBalance: true,
				})
				if err != nil {
					http.Error(w, "failed to create initial balance", http.StatusInternalServerError)
					return
				}
			}
		}

		http.Redirect(w, r, fmt.Sprintf("/accounts/%d", accountID), http.StatusSeeOther)
	})
}

func handleAccountDetail(q sqlc.Querier, tmpl *template.Template) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			http.Error(w, "invalid account id", http.StatusBadRequest)
			return
		}

		ctx := r.Context()
		account, err := q.GetAccount(ctx, id)
		if err != nil {
			http.Error(w, "account not found", http.StatusNotFound)
			return
		}

		// Ensure auto-topups
		rules, err := q.ListTopupRules(ctx, id)
		if err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}
		if err := ensureAutoTopups(ctx, q, id, rules); err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}

		bal, err := q.GetAccountBalance(ctx, id)
		if err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}

		filter := r.URL.Query().Get("filter")
		if filter == "" {
			filter = "all"
		}

		var txns []sqlc.PocketsTransaction
		switch filter {
		case "topups":
			txns, err = q.ListTransactionsTopups(ctx, id)
		case "auto":
			txns, err = q.ListTransactionsAuto(ctx, id)
		case "withdrawals":
			txns, err = q.ListTransactionsWithdrawals(ctx, id)
		default:
			filter = "all"
			txns, err = q.ListTransactionsAll(ctx, id)
		}
		if err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}

		var displayTxns []txnDisplay
		for _, tx := range txns {
			amt := numericToFloat64(tx.Amount)
			display := formatAmount(amt)
			if tx.IsInflow {
				display = "+" + display
			} else {
				display = "-" + display
			}
			note := ""
			if tx.Note.Valid {
				note = tx.Note.String
			}
			if tx.IsInitialBalance {
				if note == "" {
					note = "Initial balance"
				}
			} else if tx.IsAutoTopup {
				if note == "" {
					note = "Auto top-up"
				}
			}
			displayTxns = append(displayTxns, txnDisplay{
				ID:            tx.ID,
				Date:          tx.TxDate.Time.Format("2006-01-02"),
				DisplayAmount: display,
				IsInflow:      tx.IsInflow,
				Note:          note,
			})
		}

		balF := numericToFloat64(bal)
		data := accountDetailData{
			Balance:      formatAmount(balF),
			Filter:       filter,
			Transactions: displayTxns,
			Nav:          accountNavData{AccountID: id, Active: "transactions"},
		}
		data.Account.ID = account.ID
		data.Account.Name = account.Name
		data.Account.Icon = account.Icon
		data.Account.ColourHex = colourHex(account.Colour)
		data.Account.HasTarget = account.TargetAmount.Valid
		if account.TargetAmount.Valid {
			target := numericToFloat64(account.TargetAmount)
			data.Account.TargetAmount = formatAmount(target)
			if target > 0 {
				data.TargetPercent = int(math.Max(0, math.Min(math.Round(balF/target*100), 100)))
			}
		}

		w.Header().Set("Cache-Control", "no-store")
		renderTemplate(w, tmpl, "account_detail", data)
	})
}

func handleEditAccount(q sqlc.Querier, tmpl *template.Template) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			http.Error(w, "invalid account id", http.StatusBadRequest)
			return
		}

		ctx := r.Context()
		account, err := q.GetAccount(ctx, id)
		if err != nil {
			http.Error(w, "account not found", http.StatusNotFound)
			return
		}

		if r.Method == http.MethodPost {
			if err := r.ParseForm(); err != nil {
				http.Error(w, "invalid form data", http.StatusBadRequest)
				return
			}

			name := r.FormValue("name")
			if name == "" {
				http.Error(w, "name is required", http.StatusBadRequest)
				return
			}

			icon := r.FormValue("icon")
			if icon == "" {
				http.Error(w, "icon is required", http.StatusBadRequest)
				return
			}

			colourKey := r.FormValue("colour")
			if !validColour(colourKey) {
				http.Error(w, "invalid colour", http.StatusBadRequest)
				return
			}

			var targetAmount pgtype.Numeric
			if s := r.FormValue("target_amount"); s != "" {
				v, err := parseAmount(s)
				if err != nil {
					http.Error(w, "invalid target amount", http.StatusBadRequest)
					return
				}
				targetAmount = float64ToNumeric(v)
			}

			isReserve := r.FormValue("is_reserve") == "true"

			err = q.UpdateAccount(ctx, sqlc.UpdateAccountParams{
				ID:           id,
				Name:         name,
				Icon:         icon,
				Colour:       colourKey,
				TargetAmount: targetAmount,
				IsReserve:    isReserve,
				SortOrder:    account.SortOrder,
			})
			if err != nil {
				http.Error(w, "failed to update account", http.StatusInternalServerError)
				return
			}

			http.Redirect(w, r, fmt.Sprintf("/accounts/%d", id), http.StatusSeeOther)
			return
		}

		// GET: show edit form
		data := accountEditData{
			Colours: allColours(),
			Nav:     accountNavData{AccountID: id, Active: "edit"},
		}
		data.Account.ID = account.ID
		data.Account.Name = account.Name
		data.Account.Icon = account.Icon
		data.Account.Colour = account.Colour
		data.Account.IsReserve = account.IsReserve
		if account.TargetAmount.Valid {
			data.Account.TargetAmount = formatAmount(numericToFloat64(account.TargetAmount))
		}

		w.Header().Set("Cache-Control", "no-store")
		renderTemplate(w, tmpl, "account_edit", data)
	})
}
