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
	tx := &stubTxRunner{q: stub}
	err := ensureFailedQuests(context.Background(), tx, today)
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
		err := createNextRecurrence(context.Background(), stub, quest, completedAt, baseDate)
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
		err := createNextRecurrence(context.Background(), stub, quest, completedAt, baseDate)
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
		err := createNextRecurrence(context.Background(), stub, quest, completedAt, baseDate)
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
		err := createNextRecurrence(context.Background(), stub, quest, completedAt, baseDate)
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
		err := createNextRecurrence(context.Background(), stub, quest, completedAt, baseDate)
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
		err := createNextRecurrence(context.Background(), stub, quest, completedAt, baseDate)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if stub.createQuestCalled {
			t.Error("expected CreateQuest not to be called when unit is empty")
		}
	})

	t.Run("every skips past dates", func(t *testing.T) {
		stub := &stubQuerier{}
		questDate := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
		today := time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC)
		quest := sqlc.QuestsQuest{
			Title:          "Weekly",
			QuestType:      "side",
			QuestDate:      pgtype.Date{Time: questDate, Valid: true},
			RecurrenceType: pgtype.Text{String: "every", Valid: true},
			RecurrenceN:    pgtype.Int4{Int32: 7, Valid: true},
			RecurrenceUnit: pgtype.Text{String: "days", Valid: true},
		}
		err := createNextRecurrence(context.Background(), stub, quest, questDate, today)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stub.createQuestCalled {
			t.Fatal("expected CreateQuest to be called")
		}
		// March 1 + 7 = March 8, + 7 = March 15, + 7 = March 22 (first date after March 20)
		wantDate := time.Date(2026, 3, 22, 0, 0, 0, 0, time.UTC)
		if !stub.lastCreateQuestParams.QuestDate.Time.Equal(wantDate) {
			t.Errorf("expected date %v, got %v", wantDate, stub.lastCreateQuestParams.QuestDate.Time)
		}
	})
}

func TestCreateNextRecurrence_EndSettings(t *testing.T) {
	t.Run("every with end date within range", func(t *testing.T) {
		stub := &stubQuerier{}
		questDate := time.Date(2026, 3, 28, 0, 0, 0, 0, time.UTC)
		endDate := time.Date(2026, 4, 30, 0, 0, 0, 0, time.UTC)
		today := time.Date(2026, 3, 28, 0, 0, 0, 0, time.UTC)
		quest := sqlc.QuestsQuest{
			Title:             "Daily with end date",
			QuestType:         "daily",
			QuestDate:         pgtype.Date{Time: questDate, Valid: true},
			RecurrenceType:    pgtype.Text{String: "every", Valid: true},
			RecurrenceN:       pgtype.Int4{Int32: 1, Valid: true},
			RecurrenceUnit:    pgtype.Text{String: "days", Valid: true},
			RecurrenceEndDate: pgtype.Date{Time: endDate, Valid: true},
		}
		err := createNextRecurrence(context.Background(), stub, quest, questDate, today)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stub.createQuestCalled {
			t.Fatal("expected CreateQuest to be called")
		}
		wantDate := time.Date(2026, 3, 29, 0, 0, 0, 0, time.UTC)
		if !stub.lastCreateQuestParams.QuestDate.Time.Equal(wantDate) {
			t.Errorf("expected date %v, got %v", wantDate, stub.lastCreateQuestParams.QuestDate.Time)
		}
		if !stub.lastCreateQuestParams.RecurrenceEndDate.Valid {
			t.Fatal("expected RecurrenceEndDate to be valid")
		}
		if !stub.lastCreateQuestParams.RecurrenceEndDate.Time.Equal(endDate) {
			t.Errorf("expected end date %v, got %v", endDate, stub.lastCreateQuestParams.RecurrenceEndDate.Time)
		}
	})

	t.Run("every with end date past end", func(t *testing.T) {
		stub := &stubQuerier{}
		questDate := time.Date(2026, 3, 28, 0, 0, 0, 0, time.UTC)
		endDate := time.Date(2026, 3, 28, 0, 0, 0, 0, time.UTC)
		today := time.Date(2026, 3, 28, 0, 0, 0, 0, time.UTC)
		quest := sqlc.QuestsQuest{
			Title:             "Daily past end",
			QuestType:         "daily",
			QuestDate:         pgtype.Date{Time: questDate, Valid: true},
			RecurrenceType:    pgtype.Text{String: "every", Valid: true},
			RecurrenceN:       pgtype.Int4{Int32: 1, Valid: true},
			RecurrenceUnit:    pgtype.Text{String: "days", Valid: true},
			RecurrenceEndDate: pgtype.Date{Time: endDate, Valid: true},
		}
		err := createNextRecurrence(context.Background(), stub, quest, questDate, today)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if stub.createQuestCalled {
			t.Error("expected CreateQuest not to be called when next date is past end date")
		}
	})

	t.Run("every with end date equals next date", func(t *testing.T) {
		stub := &stubQuerier{}
		questDate := time.Date(2026, 3, 27, 0, 0, 0, 0, time.UTC)
		endDate := time.Date(2026, 3, 28, 0, 0, 0, 0, time.UTC)
		today := time.Date(2026, 3, 27, 0, 0, 0, 0, time.UTC)
		quest := sqlc.QuestsQuest{
			Title:             "Daily at boundary",
			QuestType:         "daily",
			QuestDate:         pgtype.Date{Time: questDate, Valid: true},
			RecurrenceType:    pgtype.Text{String: "every", Valid: true},
			RecurrenceN:       pgtype.Int4{Int32: 1, Valid: true},
			RecurrenceUnit:    pgtype.Text{String: "days", Valid: true},
			RecurrenceEndDate: pgtype.Date{Time: endDate, Valid: true},
		}
		err := createNextRecurrence(context.Background(), stub, quest, questDate, today)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stub.createQuestCalled {
			t.Fatal("expected CreateQuest to be called when next date equals end date (inclusive)")
		}
		wantDate := time.Date(2026, 3, 28, 0, 0, 0, 0, time.UTC)
		if !stub.lastCreateQuestParams.QuestDate.Time.Equal(wantDate) {
			t.Errorf("expected date %v, got %v", wantDate, stub.lastCreateQuestParams.QuestDate.Time)
		}
		if !stub.lastCreateQuestParams.RecurrenceEndDate.Valid {
			t.Fatal("expected RecurrenceEndDate to be inherited")
		}
		if !stub.lastCreateQuestParams.RecurrenceEndDate.Time.Equal(endDate) {
			t.Errorf("expected end date %v, got %v", endDate, stub.lastCreateQuestParams.RecurrenceEndDate.Time)
		}
	})

	t.Run("every with end date and skip-forward within range", func(t *testing.T) {
		stub := &stubQuerier{}
		questDate := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
		endDate := time.Date(2026, 3, 25, 0, 0, 0, 0, time.UTC)
		today := time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC)
		quest := sqlc.QuestsQuest{
			Title:             "Weekly skip within range",
			QuestType:         "side",
			QuestDate:         pgtype.Date{Time: questDate, Valid: true},
			RecurrenceType:    pgtype.Text{String: "every", Valid: true},
			RecurrenceN:       pgtype.Int4{Int32: 7, Valid: true},
			RecurrenceUnit:    pgtype.Text{String: "days", Valid: true},
			RecurrenceEndDate: pgtype.Date{Time: endDate, Valid: true},
		}
		err := createNextRecurrence(context.Background(), stub, quest, questDate, today)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stub.createQuestCalled {
			t.Fatal("expected CreateQuest to be called")
		}
		// March 1 + 7 = 8, + 7 = 15, + 7 = 22; 22 <= 25 so it's within range
		wantDate := time.Date(2026, 3, 22, 0, 0, 0, 0, time.UTC)
		if !stub.lastCreateQuestParams.QuestDate.Time.Equal(wantDate) {
			t.Errorf("expected date %v, got %v", wantDate, stub.lastCreateQuestParams.QuestDate.Time)
		}
		if !stub.lastCreateQuestParams.RecurrenceEndDate.Valid {
			t.Fatal("expected RecurrenceEndDate to be inherited")
		}
		if !stub.lastCreateQuestParams.RecurrenceEndDate.Time.Equal(endDate) {
			t.Errorf("expected end date %v, got %v", endDate, stub.lastCreateQuestParams.RecurrenceEndDate.Time)
		}
	})

	t.Run("every with end date and skip-forward past end", func(t *testing.T) {
		stub := &stubQuerier{}
		questDate := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
		endDate := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
		today := time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC)
		quest := sqlc.QuestsQuest{
			Title:             "Weekly skip past end",
			QuestType:         "side",
			QuestDate:         pgtype.Date{Time: questDate, Valid: true},
			RecurrenceType:    pgtype.Text{String: "every", Valid: true},
			RecurrenceN:       pgtype.Int4{Int32: 7, Valid: true},
			RecurrenceUnit:    pgtype.Text{String: "days", Valid: true},
			RecurrenceEndDate: pgtype.Date{Time: endDate, Valid: true},
		}
		err := createNextRecurrence(context.Background(), stub, quest, questDate, today)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if stub.createQuestCalled {
			t.Error("expected CreateQuest not to be called when skip-forward lands past end date")
		}
	})

	t.Run("every with max instances within limit", func(t *testing.T) {
		stub := &stubQuerier{}
		questDate := time.Date(2026, 3, 28, 0, 0, 0, 0, time.UTC)
		today := time.Date(2026, 3, 28, 0, 0, 0, 0, time.UTC)
		quest := sqlc.QuestsQuest{
			Title:                  "Daily with max",
			QuestType:              "daily",
			QuestDate:              pgtype.Date{Time: questDate, Valid: true},
			RecurrenceType:         pgtype.Text{String: "every", Valid: true},
			RecurrenceN:            pgtype.Int4{Int32: 1, Valid: true},
			RecurrenceUnit:         pgtype.Text{String: "days", Valid: true},
			RecurrenceInstance:     pgtype.Int4{Int32: 3, Valid: true},
			RecurrenceMaxInstances: pgtype.Int4{Int32: 5, Valid: true},
		}
		err := createNextRecurrence(context.Background(), stub, quest, questDate, today)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stub.createQuestCalled {
			t.Fatal("expected CreateQuest to be called")
		}
		if !stub.lastCreateQuestParams.RecurrenceInstance.Valid || stub.lastCreateQuestParams.RecurrenceInstance.Int32 != 4 {
			t.Errorf("expected instance 4, got %v", stub.lastCreateQuestParams.RecurrenceInstance)
		}
		if !stub.lastCreateQuestParams.RecurrenceMaxInstances.Valid || stub.lastCreateQuestParams.RecurrenceMaxInstances.Int32 != 5 {
			t.Errorf("expected max instances 5, got %v", stub.lastCreateQuestParams.RecurrenceMaxInstances)
		}
	})

	t.Run("every with max instances at limit", func(t *testing.T) {
		stub := &stubQuerier{}
		questDate := time.Date(2026, 3, 28, 0, 0, 0, 0, time.UTC)
		today := time.Date(2026, 3, 28, 0, 0, 0, 0, time.UTC)
		quest := sqlc.QuestsQuest{
			Title:                  "Daily at max",
			QuestType:              "daily",
			QuestDate:              pgtype.Date{Time: questDate, Valid: true},
			RecurrenceType:         pgtype.Text{String: "every", Valid: true},
			RecurrenceN:            pgtype.Int4{Int32: 1, Valid: true},
			RecurrenceUnit:         pgtype.Text{String: "days", Valid: true},
			RecurrenceInstance:     pgtype.Int4{Int32: 5, Valid: true},
			RecurrenceMaxInstances: pgtype.Int4{Int32: 5, Valid: true},
		}
		err := createNextRecurrence(context.Background(), stub, quest, questDate, today)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if stub.createQuestCalled {
			t.Error("expected CreateQuest not to be called when instance equals max instances")
		}
	})

	t.Run("every with max instances skip-forward exhausts", func(t *testing.T) {
		stub := &stubQuerier{}
		questDate := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
		today := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
		quest := sqlc.QuestsQuest{
			Title:                  "Daily skip exhausts",
			QuestType:              "daily",
			QuestDate:              pgtype.Date{Time: questDate, Valid: true},
			RecurrenceType:         pgtype.Text{String: "every", Valid: true},
			RecurrenceN:            pgtype.Int4{Int32: 1, Valid: true},
			RecurrenceUnit:         pgtype.Text{String: "days", Valid: true},
			RecurrenceInstance:     pgtype.Int4{Int32: 1, Valid: true},
			RecurrenceMaxInstances: pgtype.Int4{Int32: 5, Valid: true},
		}
		err := createNextRecurrence(context.Background(), stub, quest, questDate, today)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if stub.createQuestCalled {
			t.Error("expected CreateQuest not to be called when skip-forward exhausts max instances")
		}
	})

	t.Run("every with max instances skip-forward has room", func(t *testing.T) {
		stub := &stubQuerier{}
		questDate := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
		today := time.Date(2026, 3, 5, 0, 0, 0, 0, time.UTC)
		quest := sqlc.QuestsQuest{
			Title:                  "Daily skip with room",
			QuestType:              "daily",
			QuestDate:              pgtype.Date{Time: questDate, Valid: true},
			RecurrenceType:         pgtype.Text{String: "every", Valid: true},
			RecurrenceN:            pgtype.Int4{Int32: 1, Valid: true},
			RecurrenceUnit:         pgtype.Text{String: "days", Valid: true},
			RecurrenceInstance:     pgtype.Int4{Int32: 1, Valid: true},
			RecurrenceMaxInstances: pgtype.Int4{Int32: 10, Valid: true},
		}
		err := createNextRecurrence(context.Background(), stub, quest, questDate, today)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stub.createQuestCalled {
			t.Fatal("expected CreateQuest to be called")
		}
		// March 1 + 1 = 2 (inst 2), + 1 = 3 (inst 3), + 1 = 4 (inst 4), + 1 = 5 (not before today, exit)
		// instanceNum = 4 + 1 = 5
		wantDate := time.Date(2026, 3, 5, 0, 0, 0, 0, time.UTC)
		if !stub.lastCreateQuestParams.QuestDate.Time.Equal(wantDate) {
			t.Errorf("expected date %v, got %v", wantDate, stub.lastCreateQuestParams.QuestDate.Time)
		}
		if !stub.lastCreateQuestParams.RecurrenceInstance.Valid || stub.lastCreateQuestParams.RecurrenceInstance.Int32 != 5 {
			t.Errorf("expected instance 5, got %v", stub.lastCreateQuestParams.RecurrenceInstance)
		}
	})

	t.Run("after_completion ignores end settings", func(t *testing.T) {
		stub := &stubQuerier{}
		questDate := time.Date(2026, 3, 28, 0, 0, 0, 0, time.UTC)
		endDate := time.Date(2026, 3, 28, 0, 0, 0, 0, time.UTC)
		baseTime := time.Date(2026, 3, 29, 12, 0, 0, 0, time.UTC)
		today := time.Date(2026, 3, 29, 0, 0, 0, 0, time.UTC)
		quest := sqlc.QuestsQuest{
			Title:                  "After completion ignores end",
			QuestType:              "side",
			QuestDate:              pgtype.Date{Time: questDate, Valid: true},
			RecurrenceType:         pgtype.Text{String: "after_completion", Valid: true},
			RecurrenceN:            pgtype.Int4{Int32: 3, Valid: true},
			RecurrenceUnit:         pgtype.Text{String: "days", Valid: true},
			RecurrenceEndDate:      pgtype.Date{Time: endDate, Valid: true},
			RecurrenceInstance:     pgtype.Int4{Int32: 1, Valid: true},
			RecurrenceMaxInstances: pgtype.Int4{Int32: 1, Valid: true},
		}
		err := createNextRecurrence(context.Background(), stub, quest, baseTime, today)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !stub.createQuestCalled {
			t.Fatal("expected CreateQuest to be called (after_completion ignores end settings)")
		}
		// baseTime (March 29 12:00) + 3 days = April 1 12:00
		wantDate := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
		if !stub.lastCreateQuestParams.QuestDate.Time.Equal(wantDate) {
			t.Errorf("expected date %v, got %v", wantDate, stub.lastCreateQuestParams.QuestDate.Time)
		}
	})
}

func TestEnsureFailedQuests_createsNextRecurrence(t *testing.T) {
	today := pgtype.Date{Time: time.Date(2026, 3, 29, 0, 0, 0, 0, time.UTC), Valid: true}
	stub := &stubQuerier{
		failedQuests: []sqlc.QuestsQuest{
			{
				Title:          "Recurring daily",
				QuestType:      "daily",
				QuestDate:      pgtype.Date{Time: time.Date(2026, 3, 28, 0, 0, 0, 0, time.UTC), Valid: true},
				RecurrenceType: pgtype.Text{String: "every", Valid: true},
				RecurrenceN:    pgtype.Int4{Int32: 1, Valid: true},
				RecurrenceUnit: pgtype.Text{String: "days", Valid: true},
			},
		},
	}
	tx := &stubTxRunner{q: stub}
	err := ensureFailedQuests(context.Background(), tx, today)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stub.createQuestCalled {
		t.Fatal("expected CreateQuest to be called for recurring failed quest")
	}
	// March 28 + 1 day = March 29
	wantDate := time.Date(2026, 3, 29, 0, 0, 0, 0, time.UTC)
	if !stub.lastCreateQuestParams.QuestDate.Time.Equal(wantDate) {
		t.Errorf("expected date %v, got %v", wantDate, stub.lastCreateQuestParams.QuestDate.Time)
	}
}

func TestEnsureFailedQuests_nonRecurringNoCreate(t *testing.T) {
	today := pgtype.Date{Time: time.Date(2026, 3, 29, 0, 0, 0, 0, time.UTC), Valid: true}
	stub := &stubQuerier{
		failedQuests: []sqlc.QuestsQuest{
			{
				Title:     "One-off quest",
				QuestType: "side",
				QuestDate: pgtype.Date{Time: time.Date(2026, 3, 28, 0, 0, 0, 0, time.UTC), Valid: true},
			},
		},
	}
	tx := &stubTxRunner{q: stub}
	err := ensureFailedQuests(context.Background(), tx, today)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stub.createQuestCalled {
		t.Error("expected CreateQuest not to be called for non-recurring quest")
	}
}

func TestEnsureFailedQuests_multipleQuests(t *testing.T) {
	today := pgtype.Date{Time: time.Date(2026, 3, 29, 0, 0, 0, 0, time.UTC), Valid: true}
	stub := &stubQuerier{
		failedQuests: []sqlc.QuestsQuest{
			{
				ID:             1,
				Title:          "Daily recurring",
				QuestType:      "daily",
				QuestDate:      pgtype.Date{Time: time.Date(2026, 3, 28, 0, 0, 0, 0, time.UTC), Valid: true},
				RecurrenceType: pgtype.Text{String: "every", Valid: true},
				RecurrenceN:    pgtype.Int4{Int32: 1, Valid: true},
				RecurrenceUnit: pgtype.Text{String: "days", Valid: true},
			},
			{
				ID:        2,
				Title:     "One-off task",
				QuestType: "side",
				QuestDate: pgtype.Date{Time: time.Date(2026, 3, 28, 0, 0, 0, 0, time.UTC), Valid: true},
			},
			{
				ID:             3,
				Title:          "Weekly recurring",
				QuestType:      "side",
				QuestDate:      pgtype.Date{Time: time.Date(2026, 3, 22, 0, 0, 0, 0, time.UTC), Valid: true},
				RecurrenceType: pgtype.Text{String: "every", Valid: true},
				RecurrenceN:    pgtype.Int4{Int32: 1, Valid: true},
				RecurrenceUnit: pgtype.Text{String: "weeks", Valid: true},
			},
		},
	}
	tx := &stubTxRunner{q: stub}
	err := ensureFailedQuests(context.Background(), tx, today)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stub.createQuestCalls) != 2 {
		t.Fatalf("expected 2 CreateQuest calls, got %d", len(stub.createQuestCalls))
	}
	// First: daily recurring, March 28 + 1 = March 29
	if stub.createQuestCalls[0].Title != "Daily recurring" {
		t.Errorf("call 0: expected title %q, got %q", "Daily recurring", stub.createQuestCalls[0].Title)
	}
	wantDate1 := time.Date(2026, 3, 29, 0, 0, 0, 0, time.UTC)
	if !stub.createQuestCalls[0].QuestDate.Time.Equal(wantDate1) {
		t.Errorf("call 0: expected date %v, got %v", wantDate1, stub.createQuestCalls[0].QuestDate.Time)
	}
	// Second: weekly recurring, March 22 + 7 = March 29
	if stub.createQuestCalls[1].Title != "Weekly recurring" {
		t.Errorf("call 1: expected title %q, got %q", "Weekly recurring", stub.createQuestCalls[1].Title)
	}
	wantDate2 := time.Date(2026, 3, 29, 0, 0, 0, 0, time.UTC)
	if !stub.createQuestCalls[1].QuestDate.Time.Equal(wantDate2) {
		t.Errorf("call 1: expected date %v, got %v", wantDate2, stub.createQuestCalls[1].QuestDate.Time)
	}
}
