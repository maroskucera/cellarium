-- Cellarium Loan Tracker — sqlc query definitions
-- Copyright (C) 2026 Maroš Kučera
--
-- This program is free software: you can redistribute it and/or modify
-- it under the terms of the GNU General Public License as published by
-- the Free Software Foundation, either version 3 of the License, or
-- (at your option) any later version.
--
-- This program is distributed in the hope that it will be useful,
-- but WITHOUT ANY WARRANTY; without even the implied warranty of
-- MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
-- GNU General Public License for more details.
--
-- You should have received a copy of the GNU General Public License
-- along with this program.  If not, see <https://www.gnu.org/licenses/>.

-- name: HasEntries :one
SELECT EXISTS(SELECT 1 FROM loans.entries) AS has_entries;

-- name: CreateEntry :one
INSERT INTO loans.entries (amount, entry_date) VALUES (@amount, @entry_date) RETURNING id;

-- name: GetLoanEntry :one
SELECT id, amount, entry_date FROM loans.entries ORDER BY id ASC LIMIT 1;

-- name: GetBalance :one
SELECT COALESCE(SUM(amount), 0)::NUMERIC(12,2) AS balance FROM loans.entries;

-- name: GetTotalRepaid :one
SELECT COALESCE(-SUM(amount), 0)::NUMERIC(12,2) AS total FROM loans.entries WHERE amount < 0;

-- name: GetLastPayment :one
SELECT id, amount, entry_date FROM loans.entries WHERE amount < 0 ORDER BY id DESC LIMIT 1;

-- name: ListPayments :many
SELECT id, amount, entry_date FROM loans.entries WHERE amount < 0 ORDER BY entry_date ASC, id ASC;
