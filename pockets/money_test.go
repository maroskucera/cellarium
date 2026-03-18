// Cellarium Pockets — money helper tests
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
	"math/big"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
)

func TestNumericToFloat64(t *testing.T) {
	tests := []struct {
		name string
		n    pgtype.Numeric
		want float64
	}{
		{"zero", pgtype.Numeric{Int: big.NewInt(0), Exp: 0, Valid: true}, 0},
		{"positive cents", pgtype.Numeric{Int: big.NewInt(12345), Exp: -2, Valid: true}, 123.45},
		{"negative cents", pgtype.Numeric{Int: big.NewInt(-50000), Exp: -2, Valid: true}, -500.00},
		{"invalid", pgtype.Numeric{Valid: false}, 0},
		{"nil int", pgtype.Numeric{Int: nil, Valid: true}, 0},
		{"whole number", pgtype.Numeric{Int: big.NewInt(42), Exp: 0, Valid: true}, 42},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := numericToFloat64(tt.n)
			if got != tt.want {
				t.Errorf("numericToFloat64() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFloat64ToNumeric(t *testing.T) {
	tests := []struct {
		name string
		val  float64
		want int64
	}{
		{"zero", 0, 0},
		{"positive", 123.45, 12345},
		{"rounding", 99.999, 10000},
		{"negative", -50.00, -5000},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := float64ToNumeric(tt.val)
			if !got.Valid {
				t.Fatal("expected valid numeric")
			}
			if got.Exp != -2 {
				t.Errorf("expected exp -2, got %d", got.Exp)
			}
			if got.Int.Int64() != tt.want {
				t.Errorf("got int %d, want %d", got.Int.Int64(), tt.want)
			}
		})
	}
}

func TestParseAmount(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    float64
		wantErr bool
	}{
		{"integer", "100", 100, false},
		{"decimal", "123.45", 123.45, false},
		{"negative", "-50.00", -50.00, false},
		{"empty", "", 0, true},
		{"alpha", "abc", 0, true},
		{"leading zero", "0.50", 0.50, false},
		{"comma decimal", "123,45", 123.45, false},
		{"thousands space comma decimal", "1 234,56", 1234.56, false},
		{"thousands space dot decimal", "1 234.56", 1234.56, false},
		{"comma decimal no thousands", "1234,56", 1234.56, false},
		{"thousands space no decimal", "1 000", 1000, false},
		{"multiple thousands groups", "1 234 567,89", 1234567.89, false},
		{"exceeds max", "99999999999.00", 0, true},
		{"exceeds negative max", "-99999999999.00", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseAmount(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseAmount() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("parseAmount() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormatAmount(t *testing.T) {
	tests := []struct {
		name string
		val  float64
		want string
	}{
		{"zero", 0, "0,00"},
		{"integer", 100, "100,00"},
		{"decimal", 123.45, "123,45"},
		{"rounding", 99.999, "100,00"},
		{"thousands", 1234.56, "1 234,56"},
		{"multiple thousands", 1234567.89, "1 234 567,89"},
		{"exact thousands boundary", 1000, "1 000,00"},
		{"negative", -50.00, "-50,00"},
		{"negative thousands", -1234.56, "-1 234,56"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatAmount(tt.val)
			if got != tt.want {
				t.Errorf("formatAmount() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatParseRoundTrip(t *testing.T) {
	values := []float64{0, 1.50, 123.45, 1234.56, 1234567.89, -50.00, -1234.56}
	for _, v := range values {
		formatted := formatAmount(v)
		parsed, err := parseAmount(formatted)
		if err != nil {
			t.Errorf("parseAmount(%q) error: %v", formatted, err)
			continue
		}
		if parsed != v {
			t.Errorf("round-trip failed: %v -> %q -> %v", v, formatted, parsed)
		}
	}
}
