-- Cellarium Quests — quest line queries
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

-- name: CreateQuestLine :one
INSERT INTO quests.quest_lines (name, description, sort_order, quest_type)
VALUES (@name, @description, @sort_order, @quest_type)
RETURNING id;

-- name: GetQuestLine :one
SELECT id, name, description, sort_order, quest_type, created_at FROM quests.quest_lines WHERE id = @id;

-- name: ListQuestLines :many
SELECT id, name, description, sort_order, quest_type, created_at FROM quests.quest_lines ORDER BY sort_order ASC, id ASC;

-- name: UpdateQuestLine :exec
UPDATE quests.quest_lines SET name = @name, description = @description, sort_order = @sort_order, quest_type = @quest_type WHERE id = @id;

-- name: UpdateQuestLineSortOrder :exec
UPDATE quests.quest_lines SET sort_order = @sort_order WHERE id = @id;

-- name: DeleteQuestLine :exec
DELETE FROM quests.quest_lines WHERE id = @id;
