-- Cellarium Pockets — sqlc query definitions for transactions
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

-- name: CreateTransaction :one
INSERT INTO pockets.transactions (account_id, amount, is_inflow, tx_date, note, is_auto_topup, user_edited, is_initial_balance, topup_rule_id)
VALUES (@account_id, @amount, @is_inflow, @tx_date, @note, @is_auto_topup, @user_edited, @is_initial_balance, @topup_rule_id)
RETURNING id;

-- name: GetTransaction :one
SELECT id, account_id, amount, is_inflow, tx_date, note, is_auto_topup, user_edited, is_initial_balance, topup_rule_id, created_at
FROM pockets.transactions
WHERE id = @id;

-- name: UpdateTransaction :exec
UPDATE pockets.transactions
SET amount = @amount, is_inflow = @is_inflow, tx_date = @tx_date, note = @note, user_edited = @user_edited
WHERE id = @id;

-- name: ListTransactionsAll :many
SELECT id, account_id, amount, is_inflow, tx_date, note, is_auto_topup, user_edited, is_initial_balance, topup_rule_id, created_at
FROM pockets.transactions
WHERE account_id = @account_id
ORDER BY tx_date DESC, id DESC;

-- name: ListTransactionsTopups :many
SELECT id, account_id, amount, is_inflow, tx_date, note, is_auto_topup, user_edited, is_initial_balance, topup_rule_id, created_at
FROM pockets.transactions
WHERE account_id = @account_id AND is_inflow = TRUE AND is_initial_balance = FALSE
ORDER BY tx_date DESC, id DESC;

-- name: ListTransactionsAuto :many
SELECT id, account_id, amount, is_inflow, tx_date, note, is_auto_topup, user_edited, is_initial_balance, topup_rule_id, created_at
FROM pockets.transactions
WHERE account_id = @account_id AND is_auto_topup = TRUE
ORDER BY tx_date DESC, id DESC;

-- name: ListTransactionsWithdrawals :many
SELECT id, account_id, amount, is_inflow, tx_date, note, is_auto_topup, user_edited, is_initial_balance, topup_rule_id, created_at
FROM pockets.transactions
WHERE account_id = @account_id AND is_inflow = FALSE
ORDER BY tx_date DESC, id DESC;

-- name: GetAutoTopupForDate :one
SELECT id, account_id, amount, is_inflow, tx_date, note, is_auto_topup, user_edited, is_initial_balance, topup_rule_id, created_at
FROM pockets.transactions
WHERE account_id = @account_id AND is_auto_topup = TRUE AND tx_date = @tx_date
LIMIT 1;

-- name: ListFutureTransactions :many
SELECT id, account_id, amount, is_inflow, tx_date, note, is_auto_topup, user_edited, is_initial_balance, topup_rule_id, created_at
FROM pockets.transactions
WHERE account_id = @account_id AND tx_date > @after_date AND is_auto_topup = FALSE
ORDER BY tx_date ASC, id ASC;

-- name: UpdateAutoTopupAmount :exec
UPDATE pockets.transactions
SET amount = @amount, topup_rule_id = @topup_rule_id
WHERE id = @id AND user_edited = FALSE;
