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
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/maroskucera/cellarium/quests/db/sqlc"
)

// ensureFailedQuests marks any active quest with quest_date < today as failed.
// Called lazily on page loads and by the background ticker.
func ensureFailedQuests(ctx context.Context, q sqlc.Querier, today pgtype.Date) error {
	return q.FailOverdueQuests(ctx, today)
}

// createNextRecurrence creates the next instance of a recurring quest after completion.
func createNextRecurrence(ctx context.Context, q sqlc.Querier, quest sqlc.QuestsQuest, completedAt time.Time) error {
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
			base = completedAt
		}
	case "after_completion":
		base = completedAt
	default:
		return nil
	}

	n := int(quest.RecurrenceN.Int32)
	if n <= 0 {
		n = 1
	}

	var nextDate time.Time
	switch quest.RecurrenceUnit.String {
	case "days":
		nextDate = base.AddDate(0, 0, n)
	case "weeks":
		nextDate = base.AddDate(0, 0, n*7)
	case "months":
		nextDate = base.AddDate(0, n, 0)
	default:
		return nil
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
