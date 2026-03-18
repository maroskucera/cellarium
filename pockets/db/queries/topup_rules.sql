-- Cellarium Pockets — sqlc query definitions for top-up rules
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

-- name: CreateTopupRule :one
INSERT INTO pockets.topup_rules (account_id, amount, effective_date)
VALUES (@account_id, @amount, @effective_date)
RETURNING id;

-- name: ListTopupRules :many
SELECT id, account_id, amount, effective_date, created_at
FROM pockets.topup_rules
WHERE account_id = @account_id
ORDER BY effective_date ASC, id ASC;

-- name: DeleteTopupRule :exec
DELETE FROM pockets.topup_rules WHERE id = @id AND account_id = @account_id;

-- name: GetTopupRule :one
SELECT id, account_id, amount, effective_date, created_at
FROM pockets.topup_rules
WHERE id = @id;
