// Cellarium Pockets — forecast handlers
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
	"html/template"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/maroskucera/cellarium/pockets/db/sqlc"
)

type forecastRow struct {
	Month         string
	Balance       string
	TargetPercent int
}

type forecastData struct {
	Account struct {
		ID        int64
		Name      string
		HasTarget bool
	}
	Months int
	Rows   []forecastRow
	Nav    accountNavData
}

type forecastAllAccount struct {
	ID   int64
	Name string
	Icon string
}

type forecastAllRow struct {
	Month    string
	Balances []string
}

type forecastAllData struct {
	Accounts []forecastAllAccount
	Months   int
	Rows     []forecastAllRow
}

// computeForecast projects balances for future months based on current balance and topup rules.
// rules must be sorted by effective_date ascending (as returned by ListTopupRules).
func computeForecast(currentBalance float64, targetAmount float64, rules []sqlc.PocketsTopupRule, months int) []forecastRow {
	now := timeNow()
	balance := currentBalance
	var rows []forecastRow

	for i := 1; i <= months; i++ {
		m := time.Date(now.Year(), now.Month()+time.Month(i), 1, 0, 0, 0, 0, time.UTC)

		// Find applicable rule for this month
		var ruleAmount float64
		for j := len(rules) - 1; j >= 0; j-- {
			ruleDate := time.Date(rules[j].EffectiveDate.Time.Year(), rules[j].EffectiveDate.Time.Month(), 1, 0, 0, 0, 0, time.UTC)
			if !ruleDate.After(m) {
				ruleAmount = numericToFloat64(rules[j].Amount)
				break
			}
		}

		balance += ruleAmount

		row := forecastRow{
			Month:   m.Format("January 2006"),
			Balance: formatAmount(balance),
		}
		if targetAmount > 0 {
			row.TargetPercent = int(math.Min(math.Round(balance/targetAmount*100), 999))
		}
		rows = append(rows, row)
	}

	return rows
}

func parseMonths(r *http.Request) int {
	months := 6
	if m := r.URL.Query().Get("months"); m != "" {
		if n, err := strconv.Atoi(m); err == nil && (n == 6 || n == 12) {
			months = n
		}
	}
	return months
}

func handleAccountForecast(q sqlc.Querier, tmpl *template.Template) http.Handler {
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
		if err := ensureAutoTopups(ctx, q, id, rules); err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}

		bal, err := q.GetAccountBalance(ctx, id)
		if err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}

		months := parseMonths(r)
		var target float64
		if account.TargetAmount.Valid {
			target = numericToFloat64(account.TargetAmount)
		}
		rows := computeForecast(numericToFloat64(bal), target, rules, months)

		data := forecastData{
			Months: months,
			Rows:   rows,
			Nav:    accountNavData{AccountID: id, Active: "forecast"},
		}
		data.Account.ID = account.ID
		data.Account.Name = account.Name
		data.Account.HasTarget = account.TargetAmount.Valid

		w.Header().Set("Cache-Control", "no-store")
		renderTemplate(w, tmpl, "forecast", data)
	})
}

func handleAllForecast(q sqlc.Querier, tmpl *template.Template) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		accounts, err := q.ListAccounts(ctx)
		if err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}

		months := parseMonths(r)

		type accountForecast struct {
			info forecastAllAccount
			rows []forecastRow
		}

		var forecasts []accountForecast
		var displayAccounts []forecastAllAccount

		for _, a := range accounts {
			rules, err := q.ListTopupRules(ctx, a.ID)
			if err != nil {
				http.Error(w, "database error", http.StatusInternalServerError)
				return
			}
			if err := ensureAutoTopups(ctx, q, a.ID, rules); err != nil {
				http.Error(w, "database error", http.StatusInternalServerError)
				return
			}
			bal, err := q.GetAccountBalance(ctx, a.ID)
			if err != nil {
				http.Error(w, "database error", http.StatusInternalServerError)
				return
			}

			var target float64
			if a.TargetAmount.Valid {
				target = numericToFloat64(a.TargetAmount)
			}

			info := forecastAllAccount{ID: a.ID, Name: a.Name, Icon: a.Icon}
			rows := computeForecast(numericToFloat64(bal), target, rules, months)
			forecasts = append(forecasts, accountForecast{info: info, rows: rows})
			displayAccounts = append(displayAccounts, info)
		}

		// Build rows: one per month, with balances across accounts
		var allRows []forecastAllRow
		if len(forecasts) > 0 {
			for i := 0; i < months; i++ {
				row := forecastAllRow{
					Month: forecasts[0].rows[i].Month,
				}
				for _, f := range forecasts {
					row.Balances = append(row.Balances, f.rows[i].Balance)
				}
				allRows = append(allRows, row)
			}
		}

		w.Header().Set("Cache-Control", "no-store")
		renderTemplate(w, tmpl, "forecast_all", forecastAllData{
			Accounts: displayAccounts,
			Months:   months,
			Rows:     allRows,
		})
	})
}
