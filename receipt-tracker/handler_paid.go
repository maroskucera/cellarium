// Cellarium Receipt Tracker — HTTP handler for marking receipts as paid
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
	"math/big"
	"net/http"
	"strconv"

	"github.com/maroskucera/cellarium/receipt-tracker/db/sqlc"
)

type paidEntry struct {
	ID     int64
	Date   string
	Note   string
	Amount string
	Batch  int32
}

type batchGroup struct {
	Batch      int32
	Total      string
	Entries    []paidEntry
	totalCents int64
}

type paidPageData struct {
	Batches []batchGroup
	Error   string
	Success bool
}

// numericFromPgtypeCents converts a NUMERIC(_, 2) value to integer cents
// exactly. Cents = Int * 10^(Exp+2). For values already stored at Exp=-2
// this is a zero-cost pass-through; other exponents are handled via big.Int
// scaling with half-away-from-zero rounding for the lossy case.
func numericFromPgtypeCents(v sqlc.ListUnpaidEntriesRow) int64 {
	if !v.Value.Valid || v.Value.NaN || v.Value.Int == nil {
		return 0
	}
	shift := int(v.Value.Exp) + 2
	n := new(big.Int).Set(v.Value.Int)
	switch {
	case shift > 0:
		mul := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(shift)), nil)
		n.Mul(n, mul)
	case shift < 0:
		div := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(-shift)), nil)
		half := new(big.Int).Rsh(div, 1)
		sign := n.Sign()
		abs := new(big.Int).Abs(n)
		abs.Add(abs, half)
		abs.Quo(abs, div)
		if sign < 0 {
			n.Neg(abs)
		} else {
			n.Set(abs)
		}
	}
	return n.Int64()
}

func formatCents(c int64) string {
	neg := ""
	if c < 0 {
		neg = "-"
		c = -c
	}
	return fmt.Sprintf("%s%d.%02d", neg, c/100, c%100)
}

func formatNumericFromPgtype(v sqlc.ListUnpaidEntriesRow) string {
	return formatCents(numericFromPgtypeCents(v))
}

func handlePaid(q sqlc.Querier, tmpl *template.Template) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			entries, err := q.ListUnpaidEntries(r.Context())
			if err != nil {
				http.Error(w, "failed to load entries", http.StatusInternalServerError)
				return
			}

			var batches []batchGroup
			batchMap := make(map[int32]int) // batch number -> index in batches slice

			for _, e := range entries {
				date := ""
				if e.EntryDate.Valid {
					date = e.EntryDate.Time.Format("2006-01-02")
				}

				pe := paidEntry{
					ID:     e.ID,
					Date:   date,
					Note:   e.Note,
					Amount: formatNumericFromPgtype(e),
					Batch:  e.Batch,
				}

				idx, exists := batchMap[e.Batch]
				if !exists {
					idx = len(batches)
					batchMap[e.Batch] = idx
					batches = append(batches, batchGroup{Batch: e.Batch})
				}
				batches[idx].Entries = append(batches[idx].Entries, pe)
				batches[idx].totalCents += numericFromPgtypeCents(e)
			}
			for i := range batches {
				batches[i].Total = formatCents(batches[i].totalCents)
			}

			data := paidPageData{
				Batches: batches,
				Success: r.URL.Query().Get("saved") == "1",
			}

			tmpl.Execute(w, data)

		case http.MethodPost:
			if err := r.ParseForm(); err != nil {
				http.Error(w, "invalid form data", http.StatusBadRequest)
				return
			}

			idStrings := r.Form["ids"]
			if len(idStrings) == 0 {
				http.Redirect(w, r, "/paid", http.StatusSeeOther)
				return
			}

			ids := make([]int64, 0, len(idStrings))
			for _, s := range idStrings {
				id, err := strconv.ParseInt(s, 10, 64)
				if err != nil {
					http.Error(w, "invalid entry ID", http.StatusBadRequest)
					return
				}
				ids = append(ids, id)
			}

			if err := q.MarkEntriesPaid(r.Context(), ids); err != nil {
				http.Error(w, "failed to mark entries as paid", http.StatusInternalServerError)
				return
			}

			http.Redirect(w, r, "/paid?saved=1", http.StatusSeeOther)

		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
}
