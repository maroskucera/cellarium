-- Cellarium Quests — push subscription queries
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

-- name: CreatePushSubscription :one
INSERT INTO quests.push_subscriptions (endpoint, p256dh, auth)
VALUES (@endpoint, @p256dh, @auth)
ON CONFLICT (endpoint) DO UPDATE SET p256dh = EXCLUDED.p256dh, auth = EXCLUDED.auth
RETURNING id;

-- name: DeletePushSubscription :exec
DELETE FROM quests.push_subscriptions WHERE endpoint = @endpoint;

-- name: ListPushSubscriptions :many
SELECT id, endpoint, p256dh, auth, created_at FROM quests.push_subscriptions;
