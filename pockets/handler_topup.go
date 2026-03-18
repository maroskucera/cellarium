// Cellarium Pockets — top-up rule handlers and auto-generation
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
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/maroskucera/cellarium/pockets/db/sqlc"
)

type ruleDisplay struct {
	ID            int64
	Amount        string
	EffectiveDate string
}

type topupRulesData struct {
	Account struct {
		ID   int64
		Name string
	}
	Rules    []ruleDisplay
	TodayStr string
	Nav      accountNavData
}

func handleTopupRules(q sqlc.Querier, tmpl *template.Template) http.Handler {
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

		rules, err := q.ListTopupRules(ctx, id)
		if err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}

		var displayRules []ruleDisplay
		for _, ru := range rules {
			displayRules = append(displayRules, ruleDisplay{
				ID:            ru.ID,
				Amount:        formatAmount(numericToFloat64(ru.Amount)),
				EffectiveDate: ru.EffectiveDate.Time.Format("2006-01-02"),
			})
		}

		data := topupRulesData{
			Rules:    displayRules,
			TodayStr: timeNow().Format("2006-01-02"),
			Nav:      accountNavData{AccountID: id, Active: "topups"},
		}
		data.Account.ID = account.ID
		data.Account.Name = account.Name

		w.Header().Set("Cache-Control", "no-store")
		renderTemplate(w, tmpl, "topup_rules", data)
	})
}

func handleCreateTopupRule(q sqlc.Querier) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			http.Error(w, "invalid account id", http.StatusBadRequest)
			return
		}

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

		dateStr := r.FormValue("effective_date")
		if dateStr == "" {
			http.Error(w, "effective date is required", http.StatusBadRequest)
			return
		}
		effDate, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			http.Error(w, "invalid date format", http.StatusBadRequest)
			return
		}

		_, err = q.CreateTopupRule(r.Context(), sqlc.CreateTopupRuleParams{
			AccountID:     id,
			Amount:        float64ToNumeric(amount),
			EffectiveDate: pgtype.Date{Time: effDate, Valid: true},
		})
		if err != nil {
			http.Error(w, "failed to create rule", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, fmt.Sprintf("/accounts/%d/topups", id), http.StatusSeeOther)
	})
}

func handleDeleteTopupRule(q sqlc.Querier) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		accountID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			http.Error(w, "invalid account id", http.StatusBadRequest)
			return
		}

		ruleID, err := strconv.ParseInt(r.PathValue("rid"), 10, 64)
		if err != nil {
			http.Error(w, "invalid rule id", http.StatusBadRequest)
			return
		}

		err = q.DeleteTopupRule(r.Context(), sqlc.DeleteTopupRuleParams{
			ID:        ruleID,
			AccountID: accountID,
		})
		if err != nil {
			http.Error(w, "failed to delete rule", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, fmt.Sprintf("/accounts/%d/topups", accountID), http.StatusSeeOther)
	})
}

// ensureAutoTopups generates auto-topup transactions for all past and current months
// based on the account's topup rules. It never creates future transactions.
// rules must be sorted by effective_date ascending (as returned by ListTopupRules).
func ensureAutoTopups(ctx context.Context, q sqlc.Querier, accountID int64, rules []sqlc.PocketsTopupRule) error {
	if len(rules) == 0 {
		return nil
	}

	now := timeNow()
	currentFirst := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	// Find earliest rule date
	earliest := rules[0].EffectiveDate.Time
	startMonth := time.Date(earliest.Year(), earliest.Month(), 1, 0, 0, 0, 0, time.UTC)

	for m := startMonth; !m.After(currentFirst); m = m.AddDate(0, 1, 0) {
		// Find applicable rule: latest rule with effective_date <= m
		var applicableRule *sqlc.PocketsTopupRule
		for i := len(rules) - 1; i >= 0; i-- {
			ruleDate := time.Date(rules[i].EffectiveDate.Time.Year(), rules[i].EffectiveDate.Time.Month(), 1, 0, 0, 0, 0, time.UTC)
			if !ruleDate.After(m) {
				applicableRule = &rules[i]
				break
			}
		}
		if applicableRule == nil {
			continue
		}

		txDate := pgtype.Date{Time: m, Valid: true}
		existing, err := q.GetAutoTopupForDate(ctx, sqlc.GetAutoTopupForDateParams{
			AccountID: accountID,
			TxDate:    txDate,
		})

		if err == pgx.ErrNoRows {
			// Create new auto-topup
			_, err = q.CreateTransaction(ctx, sqlc.CreateTransactionParams{
				AccountID:   accountID,
				Amount:      applicableRule.Amount,
				IsInflow:    true,
				TxDate:      txDate,
				IsAutoTopup: true,
				TopupRuleID: pgtype.Int8{Int64: applicableRule.ID, Valid: true},
			})
			if err != nil {
				return err
			}
		} else if err != nil {
			return err
		} else {
			// Exists — update if not user-edited and amount differs
			if !existing.UserEdited {
				existingStr := formatAmount(numericToFloat64(existing.Amount))
				ruleStr := formatAmount(numericToFloat64(applicableRule.Amount))
				if existingStr != ruleStr {
					err = q.UpdateAutoTopupAmount(ctx, sqlc.UpdateAutoTopupAmountParams{
						ID:          existing.ID,
						Amount:      applicableRule.Amount,
						TopupRuleID: pgtype.Int8{Int64: applicableRule.ID, Valid: true},
					})
					if err != nil {
						return err
					}
				}
			}
		}
	}

	return nil
}
