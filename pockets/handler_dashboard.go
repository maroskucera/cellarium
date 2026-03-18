// Cellarium Pockets — dashboard handler
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

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/maroskucera/cellarium/pockets/db/sqlc"
)

type accountCard struct {
	ID            int64
	Name          string
	Icon          string
	Colour        string
	ColourHex     string
	Balance       string
	HasTarget     bool
	Target        string
	TargetPercent int
}

type dashboardData struct {
	Accounts []accountCard
}

func handleDashboard(q sqlc.Querier, tmpl *template.Template) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		accounts, err := q.ListAccounts(ctx)
		if err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}

		// Generate auto-topups for all accounts
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
		}

		today := pgtype.Date{Valid: true, Time: timeNow()}

		var cards []accountCard
		for _, a := range accounts {
			bal, err := q.GetAccountBalanceAsOfDate(ctx, sqlc.GetAccountBalanceAsOfDateParams{
				AccountID: a.ID,
				AsOfDate:  today,
			})
			if err != nil {
				http.Error(w, "database error", http.StatusInternalServerError)
				return
			}
			balF := numericToFloat64(bal)
			card := accountCard{
				ID:        a.ID,
				Name:      a.Name,
				Icon:      a.Icon,
				Colour:    a.Colour,
				ColourHex: colourHex(a.Colour),
				Balance:   formatAmount(balF),
				HasTarget: a.TargetAmount.Valid,
			}
			if a.TargetAmount.Valid {
				target := numericToFloat64(a.TargetAmount)
				card.Target = formatAmount(target)
				if target > 0 {
					card.TargetPercent = int(math.Max(0, math.Min(math.Round(balF/target*100), 100)))
				}
			}
			cards = append(cards, card)
		}

		w.Header().Set("Cache-Control", "no-store")
		renderTemplate(w, tmpl, "dashboard", dashboardData{Accounts: cards})
	})
}
