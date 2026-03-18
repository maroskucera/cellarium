// Cellarium Pockets — money type conversion and formatting helpers
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
	"math"
	"math/big"

	"github.com/jackc/pgx/v5/pgtype"
)

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

// maxAmount is the largest value that fits in NUMERIC(12,2): 9999999999.99
const maxAmount = 9999999999.99

func parseAmount(s string) (float64, error) {
	f, ok := new(big.Float).SetString(s)
	if !ok {
		return 0, errors.New("invalid decimal number")
	}
	val, _ := f.Float64()
	if val > maxAmount || val < -maxAmount {
		return 0, errors.New("amount exceeds maximum precision")
	}
	return val, nil
}

func formatAmount(val float64) string {
	return fmt.Sprintf("%.2f", val)
}
