// Cellarium Loan Tracker — HTTP handlers for loan tracking
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
	"errors"
	"fmt"
	"html/template"
	"math"
	"math/big"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/maroskucera/cellarium/loan-tracker/db/sqlc"
)

const paymentSlots = 5

type setupData struct {
	PaymentSlots []int
}

type paymentDisplay struct {
	Date   string
	Amount string
}

type dashboardData struct {
	LoanAmount        string
	TotalRepaid       string
	AmountRemaining   string
	PercentRepaid     int
	LastPaymentAmount string
	ProjectedDate     string
	TodayStr          string
	PaymentCount      int
	Payments          []paymentDisplay
}

func numericToFloat64(n pgtype.Numeric) float64 {
	if !n.Valid || n.Int == nil {
		return 0
	}
	f := new(big.Float).SetInt(n.Int)
	exp := big.NewFloat(math.Pow(10, float64(n.Exp)))
	f.Mul(f, exp)
	result, _ := f.Float64()
	return result
}

func float64ToNumeric(val float64) pgtype.Numeric {
	cents := int64(math.Round(val * 100))
	return pgtype.Numeric{
		Int:   big.NewInt(cents),
		Exp:   -2,
		Valid: true,
	}
}

func parseAmount(s string) (float64, error) {
	f, ok := new(big.Float).SetString(s)
	if !ok {
		return 0, errors.New("invalid decimal number")
	}
	val, _ := f.Float64()
	return val, nil
}

func formatAmount(val float64) string {
	return fmt.Sprintf("%.2f", val)
}

func handleIndex(q sqlc.Querier, tmpl *template.Template) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		has, err := q.HasEntries(ctx)
		if err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}

		if !has {
			slots := make([]int, paymentSlots)
			for i := range slots {
				slots[i] = i + 1
			}
			tmpl.ExecuteTemplate(w, "setup.html", setupData{PaymentSlots: slots})
			return
		}

		loan, err := q.GetLoanEntry(ctx)
		if err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}

		balance, err := q.GetBalance(ctx)
		if err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}

		totalRepaid, err := q.GetTotalRepaid(ctx)
		if err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}

		loanAmount := numericToFloat64(loan.Amount)
		repaid := numericToFloat64(totalRepaid)
		remaining := numericToFloat64(balance)

		var pct int
		if loanAmount > 0 {
			pct = int(math.Round(repaid / loanAmount * 100))
		}

		var lastAmt string
		lastPayment, err := q.GetLastPayment(ctx)
		if err == nil {
			absAmt := math.Abs(numericToFloat64(lastPayment.Amount))
			lastAmt = formatAmount(absAmt)
		}

		projected := "N/A"
		if lastAmt != "" {
			absLast := math.Abs(numericToFloat64(lastPayment.Amount))
			if absLast > 0 && remaining > 0 {
				months := math.Ceil(remaining / absLast)
				projected = time.Now().AddDate(0, int(months), 0).Format("January 2006")
			}
		}

		payments, err := q.ListPayments(ctx)
		if err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}

		var displayPayments []paymentDisplay
		for _, p := range payments {
			displayPayments = append(displayPayments, paymentDisplay{
				Date:   p.EntryDate.Time.Format("2006-01-02"),
				Amount: formatAmount(math.Abs(numericToFloat64(p.Amount))),
			})
		}

		w.Header().Set("Cache-Control", "no-store")
		tmpl.ExecuteTemplate(w, "dashboard.html", dashboardData{
			LoanAmount:        formatAmount(loanAmount),
			TotalRepaid:       formatAmount(repaid),
			AmountRemaining:   formatAmount(remaining),
			PercentRepaid:     pct,
			LastPaymentAmount: lastAmt,
			ProjectedDate:     projected,
			TodayStr:          time.Now().Format("2006-01-02"),
			PaymentCount:      len(payments),
			Payments:          displayPayments,
		})
	})
}

func handleSetup(q sqlc.Querier) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form data", http.StatusBadRequest)
			return
		}

		loanAmountStr := r.FormValue("loan_amount")
		if loanAmountStr == "" {
			http.Error(w, "loan amount is required", http.StatusBadRequest)
			return
		}

		loanAmount, err := parseAmount(loanAmountStr)
		if err != nil {
			http.Error(w, "loan amount must be a valid decimal number", http.StatusBadRequest)
			return
		}

		loanDateStr := r.FormValue("loan_date")
		if loanDateStr == "" {
			http.Error(w, "loan date is required", http.StatusBadRequest)
			return
		}

		loanDate, err := time.Parse("2006-01-02", loanDateStr)
		if err != nil {
			http.Error(w, "loan date must be in YYYY-MM-DD format", http.StatusBadRequest)
			return
		}

		ctx := r.Context()

		_, err = q.CreateEntry(ctx, sqlc.CreateEntryParams{
			Amount:    float64ToNumeric(loanAmount),
			EntryDate: pgtype.Date{Time: loanDate, Valid: true},
		})
		if err != nil {
			http.Error(w, "failed to create loan entry", http.StatusInternalServerError)
			return
		}

		for i := 1; i <= paymentSlots; i++ {
			amtStr := r.FormValue(fmt.Sprintf("payment_amount_%d", i))
			dateStr := r.FormValue(fmt.Sprintf("payment_date_%d", i))
			if amtStr == "" {
				continue
			}

			amt, err := parseAmount(amtStr)
			if err != nil {
				http.Error(w, fmt.Sprintf("payment #%d: invalid amount", i), http.StatusBadRequest)
				return
			}

			if dateStr == "" {
				http.Error(w, fmt.Sprintf("payment #%d: date is required when amount is provided", i), http.StatusBadRequest)
				return
			}

			pDate, err := time.Parse("2006-01-02", dateStr)
			if err != nil {
				http.Error(w, fmt.Sprintf("payment #%d: invalid date format", i), http.StatusBadRequest)
				return
			}

			_, err = q.CreateEntry(ctx, sqlc.CreateEntryParams{
				Amount:    float64ToNumeric(-amt),
				EntryDate: pgtype.Date{Time: pDate, Valid: true},
			})
			if err != nil {
				http.Error(w, "failed to create payment entry", http.StatusInternalServerError)
				return
			}
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)
	})
}

func handlePayment(q sqlc.Querier) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
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
		if err != nil {
			http.Error(w, "amount must be a valid decimal number", http.StatusBadRequest)
			return
		}

		dateStr := r.FormValue("payment_date")
		if dateStr == "" {
			http.Error(w, "payment date is required", http.StatusBadRequest)
			return
		}

		pDate, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			http.Error(w, "payment date must be in YYYY-MM-DD format", http.StatusBadRequest)
			return
		}

		_, err = q.CreateEntry(r.Context(), sqlc.CreateEntryParams{
			Amount:    float64ToNumeric(-amount),
			EntryDate: pgtype.Date{Time: pDate, Valid: true},
		})
		if err != nil {
			http.Error(w, "failed to create payment entry", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)
	})
}
