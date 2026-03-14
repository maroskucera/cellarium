// Cellarium Receipt Tracker — HTTP handler for creating receipt entries
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
	"encoding/json"
	"math/big"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/maroskucera/cellarium/receipt-tracker/db/sqlc"
)

type createEntryRequest struct {
	Value     string `json:"value"`
	EntryDate string `json:"entry_date"`
	Note      string `json:"note"`
}

type createEntryResponse struct {
	ID        int64  `json:"id"`
	Value     string `json:"value"`
	EntryDate string `json:"entry_date"`
	Note      string `json:"note"`
	CreatedAt string `json:"created_at"`
}

func handleCreateEntry(q sqlc.Querier) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req createEntryRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON body", http.StatusBadRequest)
			return
		}

		if req.Value == "" {
			http.Error(w, "value is required", http.StatusBadRequest)
			return
		}

		val, ok := new(big.Float).SetString(req.Value)
		if !ok {
			http.Error(w, "value must be a valid decimal number", http.StatusBadRequest)
			return
		}

		// Convert big.Float to pgtype.Numeric via big.Int with 2 decimal places
		// Multiply by 100, truncate to int, set exponent to -2
		val100, _ := new(big.Float).Mul(val, big.NewFloat(100)).Int(nil)
		numericValue := pgtype.Numeric{
			Int:   val100,
			Exp:   -2,
			Valid: true,
		}

		entryDate := time.Now()
		if req.EntryDate != "" {
			parsed, err := time.Parse("2006-01-02", req.EntryDate)
			if err != nil {
				http.Error(w, "entry_date must be in YYYY-MM-DD format", http.StatusBadRequest)
				return
			}
			entryDate = parsed
		}

		params := sqlc.CreateEntryParams{
			Value: numericValue,
			EntryDate: pgtype.Date{
				Time:  entryDate,
				Valid: true,
			},
			Note: req.Note,
		}

		entry, err := q.CreateEntry(r.Context(), params)
		if err != nil {
			http.Error(w, "failed to create entry", http.StatusInternalServerError)
			return
		}

		// Convert pgtype.Numeric back to string
		valueStr := pgtype.Numeric{Int: entry.Value.Int, Exp: entry.Value.Exp, Valid: true}
		valFloat, _ := valueStr.Float64Value()

		resp := createEntryResponse{
			ID:        entry.ID,
			Value:     big.NewFloat(valFloat.Float64).Text('f', 2),
			EntryDate: entry.EntryDate.Time.Format("2006-01-02"),
			Note:      entry.Note,
			CreatedAt: entry.CreatedAt.Time.Format(time.RFC3339),
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(resp)
	})
}
