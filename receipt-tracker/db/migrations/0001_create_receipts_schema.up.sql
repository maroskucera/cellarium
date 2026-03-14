-- Cellarium Receipt Tracker — receipt entry storage
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

CREATE SCHEMA IF NOT EXISTS receipts;

CREATE TABLE receipts.entries (
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    value      NUMERIC(12, 2) NOT NULL,
    entry_date DATE NOT NULL DEFAULT CURRENT_DATE,
    note       TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
