-- Cellarium Pockets — sqlc query definitions for accounts
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

-- name: CreateAccount :one
INSERT INTO pockets.accounts (name, icon, colour, target_amount, is_reserve)
VALUES (@name, @icon, @colour, @target_amount, @is_reserve)
RETURNING id;

-- name: GetAccount :one
SELECT id, name, icon, colour, target_amount, is_reserve, sort_order, created_at
FROM pockets.accounts
WHERE id = @id;

-- name: ListAccounts :many
SELECT id, name, icon, colour, target_amount, is_reserve, sort_order, created_at
FROM pockets.accounts
ORDER BY sort_order ASC, id ASC;

-- name: UpdateAccount :exec
UPDATE pockets.accounts
SET name = @name, icon = @icon, colour = @colour, target_amount = @target_amount,
    is_reserve = @is_reserve, sort_order = @sort_order
WHERE id = @id;

-- name: GetAccountBalance :one
SELECT COALESCE(SUM(CASE WHEN is_inflow THEN amount ELSE -amount END), 0)::NUMERIC(12,2) AS balance
FROM pockets.transactions
WHERE account_id = @account_id;

-- name: GetFirstReserveAccountID :one
SELECT id FROM pockets.accounts WHERE is_reserve = TRUE ORDER BY sort_order ASC, id ASC LIMIT 1;
