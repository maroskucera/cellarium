-- Cellarium Quests — initial schema migration
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

CREATE SCHEMA IF NOT EXISTS quests;

CREATE TABLE quests.quest_lines (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name        TEXT NOT NULL,
    description TEXT,
    sort_order  INT NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE quests.quests (
    id               BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    title            TEXT NOT NULL,
    description      TEXT,
    quest_type       TEXT NOT NULL DEFAULT 'side' CHECK (quest_type IN ('main','side','daily')),
    quest_date       DATE,
    quest_line_id    BIGINT REFERENCES quests.quest_lines(id) ON DELETE SET NULL,
    quest_giver      TEXT,
    reminder_time    TIME,
    reminder_sent_at DATE,
    sort_order       INT NOT NULL DEFAULT 0,
    status           TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active','completed','failed')),
    completed_at     TIMESTAMPTZ,
    failed_at        TIMESTAMPTZ,
    recurrence_type  TEXT CHECK (recurrence_type IN ('every','after_completion')),
    recurrence_n     INT,
    recurrence_unit  TEXT CHECK (recurrence_unit IN ('days','weeks','months')),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_quests_status ON quests.quests (status);
CREATE INDEX idx_quests_date_status ON quests.quests (quest_date, status);
CREATE INDEX idx_quests_type_order ON quests.quests (quest_type, sort_order);
CREATE INDEX idx_quests_line ON quests.quests (quest_line_id);
CREATE INDEX idx_quests_reminder ON quests.quests (reminder_time, status) WHERE reminder_time IS NOT NULL AND status = 'active';
