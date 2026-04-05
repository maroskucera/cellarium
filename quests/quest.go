// Cellarium Quests — core quest logic (failure, recurrence)
// Copyright (C) 2026 Maroš Kučera
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package main

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/maroskucera/cellarium/quests/db/sqlc"
)

// txRunner abstracts transactional execution. The function passed to RunInTx
// receives a Querier scoped to the transaction; returning a non-nil error
// rolls back the transaction.
type txRunner interface {
	RunInTx(ctx context.Context, fn func(sqlc.Querier) error) error
}

// ensureFailedQuests marks any active quest with quest_date < today as failed
// and creates the next instance for recurring quests. The entire operation runs
// in a single transaction so that a recurrence-creation failure leaves the
// quest active for retry on the next tick.
func ensureFailedQuests(ctx context.Context, tx txRunner, today pgtype.Date) error {
	return tx.RunInTx(ctx, func(q sqlc.Querier) error {
		failed, err := q.FailOverdueQuests(ctx, today)
		if err != nil {
			return err
		}
		for _, quest := range failed {
			if err := createNextRecurrence(ctx, q, quest, today.Time, today.Time); err != nil {
				return fmt.Errorf("createNextRecurrence for quest %d: %w", quest.ID, err)
			}
		}
		return nil
	})
}

// createNextRecurrence creates the next instance of a recurring quest.
// baseTime is the reference point for "after_completion" recurrence (completion or failure time).
// today is used to skip forward past dates for "every" recurrence so the next instance is in the future.
func createNextRecurrence(ctx context.Context, q sqlc.Querier, quest sqlc.QuestsQuest, baseTime time.Time, today time.Time) error {
	if !quest.RecurrenceType.Valid || quest.RecurrenceType.String == "" {
		return nil
	}

	var base time.Time
	switch quest.RecurrenceType.String {
	case "every":
		// next date from the current quest_date
		if quest.QuestDate.Valid {
			base = quest.QuestDate.Time
		} else {
			base = baseTime // defensive fallback; unreachable via FailOverdueQuests (quest_date IS NOT NULL)
		}
	case "after_completion":
		base = baseTime
	default:
		return nil
	}

	n := int(quest.RecurrenceN.Int32)
	if n <= 0 {
		n = 1
	}

	addInterval := func(t time.Time) time.Time {
		switch quest.RecurrenceUnit.String {
		case "days":
			return t.AddDate(0, 0, n)
		case "weeks":
			return t.AddDate(0, 0, n*7)
		case "months":
			return t.AddDate(0, n, 0)
		}
		return t
	}

	nextDate := addInterval(base)
	if nextDate.Equal(base) {
		// Unknown unit — addInterval returned unchanged time.
		// Safe because n >= 1, so known units always advance the date.
		return nil
	}

	// For "every" recurrence, skip forward until nextDate is today or later.
	if quest.RecurrenceType.String == "every" {
		for nextDate.Before(today) {
			nextDate = addInterval(nextDate)
		}
	}

	_, err := q.CreateQuest(ctx, sqlc.CreateQuestParams{
		Title:          quest.Title,
		Description:    quest.Description,
		QuestType:      quest.QuestType,
		QuestDate:      pgtype.Date{Time: nextDate, Valid: true},
		QuestLineID:    quest.QuestLineID,
		QuestGiver:     quest.QuestGiver,
		ReminderTime:   quest.ReminderTime,
		SortOrder:      quest.SortOrder,
		RecurrenceType: quest.RecurrenceType,
		RecurrenceN:    quest.RecurrenceN,
		RecurrenceUnit: quest.RecurrenceUnit,
	})
	return err
}
