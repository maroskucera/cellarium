// Cellarium Quests — tests for core quest logic
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
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/maroskucera/cellarium/quests/db/sqlc"
)

func TestEnsureFailedQuests(t *testing.T) {
	today := pgtype.Date{Time: time.Date(2026, 3, 29, 0, 0, 0, 0, time.UTC), Valid: true}
	stub := &stubQuerier{}
	err := ensureFailedQuests(context.Background(), stub, today)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stub.failOverdueQuestsCall == nil {
		t.Fatal("expected FailOverdueQuests to be called")
	}
	if !stub.failOverdueQuestsCall.Equal(today.Time) {
		t.Errorf("expected today %v, got %v", today.Time, stub.failOverdueQuestsCall)
	}
}

func TestCreateNextRecurrence(t *testing.T) {
	completedAt := time.Date(2026, 3, 29, 12, 0, 0, 0, time.UTC)
	baseDate := time.Date(2026, 3, 29, 0, 0, 0, 0, time.UTC)

	t.Run("no recurrence", func(t *testing.T) {
		stub := &stubQuerier{}
		quest := sqlc.QuestsQuest{
			Title:     "My quest",
			QuestType: "side",
		}
		err := createNextRecurrence(context.Background(), stub, quest, completedAt)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if stub.createQuestCalled {
			t.Error("expected CreateQuest not to be called")
		}
	})

	t.Run("every N days", func(t *testing.T) {
		stub := &stubQuerier{}
		quest := sqlc.QuestsQuest{
			Title:          "Daily",
			QuestType:      "daily",
			QuestDate:      pgtype.Date{Time: baseDate, Valid: true},
			RecurrenceType: pgtype.Text{String: "every", Valid: true},
			RecurrenceN:    pgtype.Int4{Int32: 1, Valid: true},
			RecurrenceUnit: pgtype.Text{String: "days", Valid: true},
		}
		err := createNextRecurrence(context.Background(), stub, quest, completedAt)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stub.createQuestCalled {
			t.Fatal("expected CreateQuest to be called")
		}
		wantDate := baseDate.AddDate(0, 0, 1)
		if !stub.lastCreateQuestParams.QuestDate.Time.Equal(wantDate) {
			t.Errorf("expected date %v, got %v", wantDate, stub.lastCreateQuestParams.QuestDate.Time)
		}
	})

	t.Run("every N weeks", func(t *testing.T) {
		stub := &stubQuerier{}
		quest := sqlc.QuestsQuest{
			Title:          "Weekly quest",
			QuestType:      "side",
			QuestDate:      pgtype.Date{Time: baseDate, Valid: true},
			RecurrenceType: pgtype.Text{String: "every", Valid: true},
			RecurrenceN:    pgtype.Int4{Int32: 2, Valid: true},
			RecurrenceUnit: pgtype.Text{String: "weeks", Valid: true},
		}
		err := createNextRecurrence(context.Background(), stub, quest, completedAt)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		wantDate := baseDate.AddDate(0, 0, 14)
		if !stub.lastCreateQuestParams.QuestDate.Time.Equal(wantDate) {
			t.Errorf("expected date %v, got %v", wantDate, stub.lastCreateQuestParams.QuestDate.Time)
		}
	})

	t.Run("every N months", func(t *testing.T) {
		stub := &stubQuerier{}
		quest := sqlc.QuestsQuest{
			Title:          "Monthly quest",
			QuestType:      "side",
			QuestDate:      pgtype.Date{Time: baseDate, Valid: true},
			RecurrenceType: pgtype.Text{String: "every", Valid: true},
			RecurrenceN:    pgtype.Int4{Int32: 1, Valid: true},
			RecurrenceUnit: pgtype.Text{String: "months", Valid: true},
		}
		err := createNextRecurrence(context.Background(), stub, quest, completedAt)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		wantDate := baseDate.AddDate(0, 1, 0)
		if !stub.lastCreateQuestParams.QuestDate.Time.Equal(wantDate) {
			t.Errorf("expected date %v, got %v", wantDate, stub.lastCreateQuestParams.QuestDate.Time)
		}
	})

	t.Run("after_completion N days", func(t *testing.T) {
		stub := &stubQuerier{}
		quest := sqlc.QuestsQuest{
			Title:          "Flexible quest",
			QuestType:      "side",
			QuestDate:      pgtype.Date{Time: baseDate, Valid: true},
			RecurrenceType: pgtype.Text{String: "after_completion", Valid: true},
			RecurrenceN:    pgtype.Int4{Int32: 3, Valid: true},
			RecurrenceUnit: pgtype.Text{String: "days", Valid: true},
		}
		err := createNextRecurrence(context.Background(), stub, quest, completedAt)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// base is completedAt, not quest date
		wantDate := completedAt.AddDate(0, 0, 3)
		if !stub.lastCreateQuestParams.QuestDate.Time.Equal(wantDate) {
			t.Errorf("expected date %v, got %v", wantDate, stub.lastCreateQuestParams.QuestDate.Time)
		}
	})

	t.Run("recurrence type but no unit — no quest created", func(t *testing.T) {
		stub := &stubQuerier{}
		quest := sqlc.QuestsQuest{
			Title:          "Broken quest",
			QuestType:      "side",
			RecurrenceType: pgtype.Text{String: "every", Valid: true},
			RecurrenceN:    pgtype.Int4{Int32: 1, Valid: true},
			RecurrenceUnit: pgtype.Text{String: "", Valid: false},
		}
		err := createNextRecurrence(context.Background(), stub, quest, completedAt)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if stub.createQuestCalled {
			t.Error("expected CreateQuest not to be called when unit is empty")
		}
	})
}
