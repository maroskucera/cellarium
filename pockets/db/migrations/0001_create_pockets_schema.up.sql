-- Cellarium Pockets — initial schema for virtual bank account tracking
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

CREATE SCHEMA IF NOT EXISTS pockets;

CREATE TABLE pockets.accounts (
    id            BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name          TEXT NOT NULL,
    icon          TEXT NOT NULL,
    colour        TEXT NOT NULL,
    target_amount NUMERIC(12, 2),
    is_reserve    BOOLEAN NOT NULL DEFAULT FALSE,
    sort_order    INT NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE pockets.topup_rules (
    id             BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    account_id     BIGINT NOT NULL REFERENCES pockets.accounts(id),
    amount         NUMERIC(12, 2) NOT NULL,
    effective_date DATE NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_topup_rules_account_effective ON pockets.topup_rules (account_id, effective_date);

CREATE TABLE pockets.transactions (
    id                 BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    account_id         BIGINT NOT NULL REFERENCES pockets.accounts(id),
    amount             NUMERIC(12, 2) NOT NULL,
    is_inflow          BOOLEAN NOT NULL,
    tx_date            DATE NOT NULL DEFAULT CURRENT_DATE,
    note               TEXT,
    is_auto_topup      BOOLEAN NOT NULL DEFAULT FALSE,
    user_edited        BOOLEAN NOT NULL DEFAULT FALSE,
    is_initial_balance BOOLEAN NOT NULL DEFAULT FALSE,
    topup_rule_id      BIGINT REFERENCES pockets.topup_rules(id),
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_transactions_account_date ON pockets.transactions (account_id, tx_date DESC);
CREATE INDEX idx_transactions_account_autotopup_date ON pockets.transactions (account_id, is_auto_topup, tx_date);
CREATE UNIQUE INDEX idx_transactions_one_autotopup_per_month ON pockets.transactions (account_id, tx_date) WHERE is_auto_topup = TRUE;
