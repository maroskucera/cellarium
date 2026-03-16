// Cellarium Receipt Tracker — HTTP handler for the server-rendered form
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
	"math/big"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/maroskucera/cellarium/receipt-tracker/db/sqlc"
)

type pageData struct {
	Error     string
	Success   bool
	Value     string
	EntryDate string
	Note      string
	Today     string
}

func handleRoot(q sqlc.Querier, tmpl *template.Template) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		today := time.Now().Format("2006-01-02")

		switch r.Method {
		case http.MethodGet:
			data := pageData{
				Today:   today,
				Success: r.URL.Query().Get("saved") == "1",
			}
			tmpl.Execute(w, data)

		case http.MethodPost:
			value := r.FormValue("value")
			entryDate := r.FormValue("entry_date")
			note := r.FormValue("note")

			data := pageData{
				Value:     value,
				EntryDate: entryDate,
				Note:      note,
				Today:     today,
			}

			if value == "" {
				data.Error = "value is required"
				tmpl.Execute(w, data)
				return
			}

			val, ok := new(big.Float).SetString(value)
			if !ok {
				data.Error = "value must be a valid decimal number"
				tmpl.Execute(w, data)
				return
			}

			val100, _ := new(big.Float).Mul(val, big.NewFloat(100)).Int(nil)
			numericValue := pgtype.Numeric{
				Int:   val100,
				Exp:   -2,
				Valid: true,
			}

			entryTime := time.Now()
			if entryDate != "" {
				parsed, err := time.Parse("2006-01-02", entryDate)
				if err != nil {
					data.Error = "date must be in YYYY-MM-DD format"
					tmpl.Execute(w, data)
					return
				}
				entryTime = parsed
			}

			params := sqlc.CreateEntryParams{
				Value: numericValue,
				EntryDate: pgtype.Date{
					Time:  entryTime,
					Valid: true,
				},
				Note: note,
			}

			_, err := q.CreateEntry(r.Context(), params)
			if err != nil {
				data.Error = "failed to create entry"
				http.Error(w, data.Error, http.StatusInternalServerError)
				return
			}

			http.Redirect(w, r, "/?saved=1", http.StatusSeeOther)

		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
}
