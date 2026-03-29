-- Cellarium Quests — quest queries
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

-- name: CreateQuest :one
INSERT INTO quests.quests (title, description, quest_type, quest_date, quest_line_id, quest_giver, reminder_time, sort_order, recurrence_type, recurrence_n, recurrence_unit)
VALUES (@title, @description, @quest_type, @quest_date, @quest_line_id, @quest_giver, @reminder_time, @sort_order, @recurrence_type, @recurrence_n, @recurrence_unit)
RETURNING id;

-- name: GetQuest :one
SELECT id, title, description, quest_type, quest_date, quest_line_id, quest_giver, reminder_time, reminder_sent_at, sort_order, status, completed_at, failed_at, recurrence_type, recurrence_n, recurrence_unit, created_at
FROM quests.quests WHERE id = @id;

-- name: ListActiveQuests :many
SELECT id, title, description, quest_type, quest_date, quest_line_id, quest_giver, reminder_time, reminder_sent_at, sort_order, status, completed_at, failed_at, recurrence_type, recurrence_n, recurrence_unit, created_at
FROM quests.quests WHERE status = 'active' ORDER BY sort_order ASC, id ASC;

-- name: ListTodayQuests :many
SELECT id, title, description, quest_type, quest_date, quest_line_id, quest_giver, reminder_time, reminder_sent_at, sort_order, status, completed_at, failed_at, recurrence_type, recurrence_n, recurrence_unit, created_at
FROM quests.quests WHERE status = 'active' AND quest_date = @quest_date ORDER BY quest_type ASC, sort_order ASC, id ASC;

-- name: ListQuestLog :many
SELECT id, title, description, quest_type, quest_date, quest_line_id, quest_giver, reminder_time, reminder_sent_at, sort_order, status, completed_at, failed_at, recurrence_type, recurrence_n, recurrence_unit, created_at
FROM quests.quests WHERE status IN ('completed','failed') ORDER BY COALESCE(completed_at, failed_at) DESC NULLS LAST, id DESC;

-- name: UpdateQuest :exec
UPDATE quests.quests SET title = @title, description = @description, quest_type = @quest_type, quest_date = @quest_date, quest_line_id = @quest_line_id, quest_giver = @quest_giver, reminder_time = @reminder_time, sort_order = @sort_order, recurrence_type = @recurrence_type, recurrence_n = @recurrence_n, recurrence_unit = @recurrence_unit WHERE id = @id;

-- name: CompleteQuest :exec
UPDATE quests.quests SET status = 'completed', completed_at = now() WHERE id = @id;

-- name: FailQuest :exec
UPDATE quests.quests SET status = 'failed', failed_at = now() WHERE id = @id;

-- name: FailOverdueQuests :exec
UPDATE quests.quests SET status = 'failed', failed_at = now() WHERE status = 'active' AND quest_date IS NOT NULL AND quest_date < @today;

-- name: DeleteQuest :exec
DELETE FROM quests.quests WHERE id = @id;

-- name: UpdateQuestSortOrder :exec
UPDATE quests.quests SET sort_order = @sort_order WHERE id = @id;

-- name: ListQuestGivers :many
SELECT DISTINCT quest_giver FROM quests.quests WHERE quest_giver IS NOT NULL AND quest_giver <> '' ORDER BY quest_giver ASC;

-- name: ListDueReminders :many
SELECT id, title, description, quest_type, quest_date, quest_line_id, quest_giver, reminder_time, reminder_sent_at, sort_order, status, completed_at, failed_at, recurrence_type, recurrence_n, recurrence_unit, created_at
FROM quests.quests WHERE status = 'active' AND reminder_time IS NOT NULL AND reminder_time <= @now_time AND (reminder_sent_at IS NULL OR reminder_sent_at < @today);

-- name: MarkReminderSent :exec
UPDATE quests.quests SET reminder_sent_at = @today WHERE id = @id;

-- name: UncompleteQuest :exec
UPDATE quests.quests SET status = 'active', completed_at = NULL WHERE id = @id;

-- name: UncompleteQuestAndResetDate :exec
UPDATE quests.quests SET status = 'active', completed_at = NULL, quest_date = @quest_date WHERE id = @id;

-- name: ListActiveAndCompletedQuests :many
SELECT id, title, description, quest_type, quest_date, quest_line_id, quest_giver, reminder_time, reminder_sent_at, sort_order, status, completed_at, failed_at, recurrence_type, recurrence_n, recurrence_unit, created_at
FROM quests.quests WHERE status IN ('active', 'completed') ORDER BY quest_type ASC, sort_order ASC, id ASC;

-- name: UpdateQuestTypeByLine :exec
UPDATE quests.quests SET quest_type = @quest_type WHERE quest_line_id = @quest_line_id;
