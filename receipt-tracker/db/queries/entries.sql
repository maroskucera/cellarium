-- Cellarium Receipt Tracker — receipt entry queries
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

-- name: CreateEntry :one
WITH current_batch AS (
    SELECT COALESCE(MAX(batch), 0) AS max_batch FROM receipts.entries
),
next_batch AS (
    SELECT CASE
        WHEN cb.max_batch = 0 THEN 1
        WHEN EXISTS (
            SELECT 1 FROM receipts.entries
            WHERE batch = cb.max_batch AND paid = FALSE
        ) THEN cb.max_batch
        ELSE cb.max_batch + 1
    END AS batch
    FROM current_batch cb
)
INSERT INTO receipts.entries (value, entry_date, note, batch, paid)
SELECT @value, @entry_date, @note, nb.batch, FALSE
FROM next_batch nb
RETURNING id;

-- name: ListUnpaidEntries :many
SELECT id, value, entry_date, note, batch
FROM receipts.entries
WHERE paid = FALSE
ORDER BY batch ASC, entry_date ASC, id ASC;

-- name: MarkEntriesPaid :exec
UPDATE receipts.entries
SET paid = TRUE
WHERE id = ANY(@ids::bigint[]);
