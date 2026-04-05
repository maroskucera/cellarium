// Cellarium Quests — quest CRUD handlers
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
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/maroskucera/cellarium/quests/db/sqlc"
)

type questLineOption struct {
	ID        int64
	Name      string
	Selected  bool
	QuestType string // empty if the quest line has no fixed type
}

type questFormData struct {
	Nav        string
	Title      string
	Action     string
	Quest      questFormValues
	QuestLines []questLineOption
}

type questFormValues struct {
	ID             int64
	Title          string
	Description    string
	QuestType      string
	QuestDate      string
	QuestLineID    int64
	QuestGiver     string
	ReminderTime   string
	RecurrenceType string
	RecurrenceN    string
	RecurrenceUnit string
}

func parseQuestForm(r *http.Request) (sqlc.CreateQuestParams, error) {
	if err := r.ParseForm(); err != nil {
		return sqlc.CreateQuestParams{}, err
	}

	params := sqlc.CreateQuestParams{
		Title:     r.FormValue("title"),
		QuestType: r.FormValue("quest_type"),
	}

	if params.QuestType == "" {
		params.QuestType = "main"
	}

	if desc := r.FormValue("description"); desc != "" {
		params.Description = pgtype.Text{String: desc, Valid: true}
	}

	if giver := r.FormValue("quest_giver"); giver != "" {
		params.QuestGiver = pgtype.Text{String: giver, Valid: true}
	}

	if dateStr := r.FormValue("quest_date"); dateStr != "" {
		d, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			return sqlc.CreateQuestParams{}, fmt.Errorf("invalid date: %w", err)
		}
		params.QuestDate = pgtype.Date{Time: d, Valid: true}
	}

	if qlIDStr := r.FormValue("quest_line_id"); qlIDStr != "" && qlIDStr != "0" {
		id, err := strconv.ParseInt(qlIDStr, 10, 64)
		if err == nil && id > 0 {
			params.QuestLineID = pgtype.Int8{Int64: id, Valid: true}
		}
	}

	if rtStr := r.FormValue("reminder_time"); rtStr != "" {
		t, err := time.Parse("15:04", rtStr)
		if err == nil {
			params.ReminderTime = pgtype.Time{
				Microseconds: int64(t.Hour())*3600_000_000 + int64(t.Minute())*60_000_000,
				Valid:        true,
			}
		}
	}

	if recType := r.FormValue("recurrence_type"); recType != "" && recType != "none" {
		params.RecurrenceType = pgtype.Text{String: recType, Valid: true}
		if recNStr := r.FormValue("recurrence_n"); recNStr != "" {
			n, err := strconv.ParseInt(recNStr, 10, 32)
			if err == nil && n > 0 {
				params.RecurrenceN = pgtype.Int4{Int32: int32(n), Valid: true}
			}
		}
		if recUnit := r.FormValue("recurrence_unit"); recUnit != "" {
			params.RecurrenceUnit = pgtype.Text{String: recUnit, Valid: true}
		}
	}

	return params, nil
}

func loadQuestLines(q sqlc.Querier, r *http.Request, selectedID int64) ([]questLineOption, error) {
	lines, err := q.ListQuestLines(r.Context())
	if err != nil {
		return nil, err
	}
	opts := make([]questLineOption, 0, len(lines))
	for _, l := range lines {
		qt := ""
		if l.QuestType.Valid {
			qt = l.QuestType.String
		}
		opts = append(opts, questLineOption{
			ID:        l.ID,
			Name:      l.Name,
			Selected:  l.ID == selectedID,
			QuestType: qt,
		})
	}
	return opts, nil
}

func handleNewQuest(q sqlc.Querier, tmpl *template.Template) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			params, err := parseQuestForm(r)
			if err != nil || params.Title == "" {
				http.Error(w, "invalid form data", http.StatusBadRequest)
				return
			}
			// Enforce type from quest line when it has a fixed type
			if params.QuestLineID.Valid {
				if line, err := q.GetQuestLine(r.Context(), params.QuestLineID.Int64); err == nil && line.QuestType.Valid {
					params.QuestType = line.QuestType.String
				}
			}
			if _, err := q.CreateQuest(r.Context(), params); err != nil {
				http.Error(w, "database error", http.StatusInternalServerError)
				return
			}
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		fv := questFormValues{
			QuestType:      "main",
			RecurrenceType: "none",
			RecurrenceN:    "1",
			RecurrenceUnit: "days",
		}

		lines, err := loadQuestLines(q, r, 0)
		if err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}
		data := questFormData{
			Nav:        "today",
			Title:      "New Quest",
			Action:     "/quests/new",
			QuestLines: lines,
			Quest:      fv,
		}
		w.Header().Set("Cache-Control", "no-store")
		renderTemplate(w, tmpl, "quest_new", data)
	})
}

type questDetailData struct {
	Nav   string
	Quest questDetailDisplay
}

type questDetailDisplay struct {
	ID             int64
	Title          string
	Description    string
	QuestType      string
	QuestDate      string
	QuestLineName  string
	QuestGiver     string
	ReminderTime   string
	Status         string
	CompletedAt    string
	FailedAt       string
	RecurrenceType string
	RecurrenceN    string
	RecurrenceUnit string
}

func handleViewQuest(q sqlc.Querier, tmpl *template.Template) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		idStr := r.PathValue("id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			http.Error(w, "invalid quest id", http.StatusBadRequest)
			return
		}

		ctx := r.Context()
		quest, err := q.GetQuest(ctx, id)
		if err != nil {
			http.Error(w, "quest not found", http.StatusNotFound)
			return
		}

		d := questDetailDisplay{
			ID:        quest.ID,
			Title:     quest.Title,
			QuestType: quest.QuestType,
			Status:    quest.Status,
		}

		if quest.Description.Valid {
			d.Description = quest.Description.String
		}
		if quest.QuestDate.Valid {
			d.QuestDate = quest.QuestDate.Time.Format("Monday, 2 January 2006")
		}
		if quest.QuestGiver.Valid {
			d.QuestGiver = quest.QuestGiver.String
		}
		if quest.ReminderTime.Valid {
			h := quest.ReminderTime.Microseconds / 3600_000_000
			m := (quest.ReminderTime.Microseconds % 3600_000_000) / 60_000_000
			d.ReminderTime = time.Date(0, 1, 1, int(h), int(m), 0, 0, time.UTC).Format("15:04")
		}
		if quest.CompletedAt.Valid {
			d.CompletedAt = quest.CompletedAt.Time.In(appLocation).Format("2 Jan 2006, 15:04")
		}
		if quest.FailedAt.Valid {
			d.FailedAt = quest.FailedAt.Time.In(appLocation).Format("2 Jan 2006, 15:04")
		}
		if quest.RecurrenceType.Valid && quest.RecurrenceType.String != "" {
			d.RecurrenceType = quest.RecurrenceType.String
		}
		if quest.RecurrenceN.Valid {
			d.RecurrenceN = strconv.Itoa(int(quest.RecurrenceN.Int32))
		}
		if quest.RecurrenceUnit.Valid {
			d.RecurrenceUnit = quest.RecurrenceUnit.String
		}

		if quest.QuestLineID.Valid {
			if line, err := q.GetQuestLine(ctx, quest.QuestLineID.Int64); err == nil {
				d.QuestLineName = line.Name
			}
		}

		data := questDetailData{
			Nav:   "log",
			Quest: d,
		}
		w.Header().Set("Cache-Control", "no-store")
		renderTemplate(w, tmpl, "quest_detail", data)
	})
}

type retryFormData struct {
	Nav       string
	QuestID   int64
	QuestName string
	QuestDate string
}

func handleRetryQuest(q sqlc.Querier, tmpl *template.Template) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		idStr := r.PathValue("id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			http.Error(w, "invalid quest id", http.StatusBadRequest)
			return
		}

		quest, err := q.GetQuest(r.Context(), id)
		if err != nil {
			http.Error(w, "quest not found", http.StatusNotFound)
			return
		}

		if quest.Status != "failed" {
			http.Redirect(w, r, "/quests/"+idStr+"/edit", http.StatusSeeOther)
			return
		}

		if r.Method == http.MethodPost {
			if err := r.ParseForm(); err != nil {
				http.Error(w, "invalid form data", http.StatusBadRequest)
				return
			}

			params := sqlc.CreateQuestParams{
				Title:          quest.Title,
				QuestType:      quest.QuestType,
				Description:    quest.Description,
				QuestLineID:    quest.QuestLineID,
				QuestGiver:     quest.QuestGiver,
				ReminderTime:   quest.ReminderTime,
				RecurrenceType: quest.RecurrenceType,
				RecurrenceN:    quest.RecurrenceN,
				RecurrenceUnit: quest.RecurrenceUnit,
			}

			if dateStr := r.FormValue("quest_date"); dateStr != "" {
				d, err := time.Parse("2006-01-02", dateStr)
				if err != nil {
					http.Error(w, "invalid date format", http.StatusBadRequest)
					return
				}
				params.QuestDate = pgtype.Date{Time: d, Valid: true}
			}

			if _, err := q.CreateQuest(r.Context(), params); err != nil {
				http.Error(w, "database error", http.StatusInternalServerError)
				return
			}
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		today := timeNow().In(appLocation).Format("2006-01-02")
		data := retryFormData{
			Nav:       "log",
			QuestID:   id,
			QuestName: quest.Title,
			QuestDate: today,
		}
		w.Header().Set("Cache-Control", "no-store")
		renderTemplate(w, tmpl, "quest_retry", data)
	})
}

func handleEditQuest(q sqlc.Querier, tmpl *template.Template) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		idStr := r.PathValue("id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			http.Error(w, "invalid quest id", http.StatusBadRequest)
			return
		}

		ctx := r.Context()
		quest, err := q.GetQuest(ctx, id)
		if err != nil {
			http.Error(w, "quest not found", http.StatusNotFound)
			return
		}

		if quest.Status == "failed" {
			http.Redirect(w, r, "/quests/"+idStr+"/retry", http.StatusSeeOther)
			return
		}

		if r.Method == http.MethodPost {
			params, err := parseQuestForm(r)
			if err != nil || params.Title == "" {
				http.Error(w, "invalid form data", http.StatusBadRequest)
				return
			}
			// Enforce type from quest line when it has a fixed type
			if params.QuestLineID.Valid {
				if line, err := q.GetQuestLine(ctx, params.QuestLineID.Int64); err == nil && line.QuestType.Valid {
					params.QuestType = line.QuestType.String
				}
			}
			updateParams := sqlc.UpdateQuestParams{
				ID:             id,
				Title:          params.Title,
				Description:    params.Description,
				QuestType:      params.QuestType,
				QuestDate:      params.QuestDate,
				QuestLineID:    params.QuestLineID,
				QuestGiver:     params.QuestGiver,
				ReminderTime:   params.ReminderTime,
				SortOrder:      quest.SortOrder,
				RecurrenceType: params.RecurrenceType,
				RecurrenceN:    params.RecurrenceN,
				RecurrenceUnit: params.RecurrenceUnit,
			}
			if err := q.UpdateQuest(ctx, updateParams); err != nil {
				http.Error(w, "database error", http.StatusInternalServerError)
				return
			}
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		var selectedLineID int64
		if quest.QuestLineID.Valid {
			selectedLineID = quest.QuestLineID.Int64
		}
		lines, err := loadQuestLines(q, r, selectedLineID)
		if err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}

		fv := questFormValues{
			ID:        id,
			Title:     quest.Title,
			QuestType: quest.QuestType,
		}
		if quest.Description.Valid {
			fv.Description = quest.Description.String
		}
		if quest.QuestDate.Valid {
			fv.QuestDate = quest.QuestDate.Time.Format("2006-01-02")
		}
		if quest.QuestLineID.Valid {
			fv.QuestLineID = quest.QuestLineID.Int64
		}
		if quest.QuestGiver.Valid {
			fv.QuestGiver = quest.QuestGiver.String
		}
		if quest.ReminderTime.Valid {
			h := quest.ReminderTime.Microseconds / 3600_000_000
			m := (quest.ReminderTime.Microseconds % 3600_000_000) / 60_000_000
			fv.ReminderTime = time.Date(0, 1, 1, int(h), int(m), 0, 0, time.UTC).Format("15:04")
		}
		if quest.RecurrenceType.Valid && quest.RecurrenceType.String != "" {
			fv.RecurrenceType = quest.RecurrenceType.String
		} else {
			fv.RecurrenceType = "none"
		}
		if quest.RecurrenceN.Valid {
			fv.RecurrenceN = strconv.Itoa(int(quest.RecurrenceN.Int32))
		} else {
			fv.RecurrenceN = "1"
		}
		if quest.RecurrenceUnit.Valid {
			fv.RecurrenceUnit = quest.RecurrenceUnit.String
		} else {
			fv.RecurrenceUnit = "days"
		}

		data := questFormData{
			Nav:        "today",
			Title:      "Edit Quest",
			Action:     "/quests/" + idStr + "/edit",
			QuestLines: lines,
			Quest:      fv,
		}
		w.Header().Set("Cache-Control", "no-store")
		renderTemplate(w, tmpl, "quest_edit", data)
	})
}

func handleDeleteQuest(q sqlc.Querier) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			http.Error(w, "invalid quest id", http.StatusBadRequest)
			return
		}
		if err := q.DeleteQuest(r.Context(), id); err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
	})
}

func handleCompleteQuest(q sqlc.Querier) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			http.Error(w, "invalid quest id", http.StatusBadRequest)
			return
		}
		ctx := r.Context()
		quest, err := q.GetQuest(ctx, id)
		if err != nil {
			http.Error(w, "quest not found", http.StatusNotFound)
			return
		}
		if err := q.CompleteQuest(ctx, id); err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}
		now := timeNow()
		if err := createNextRecurrence(ctx, q, quest, now, localToday(now).Time); err != nil {
			// log but don't fail — quest is already completed
			_ = err
		}
		w.WriteHeader(http.StatusNoContent)
	})
}

func handleUncompleteQuest(q sqlc.Querier) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			http.Error(w, "invalid quest id", http.StatusBadRequest)
			return
		}
		ctx := r.Context()
		quest, err := q.GetQuest(ctx, id)
		if err != nil {
			http.Error(w, "quest not found", http.StatusNotFound)
			return
		}
		today := localToday(timeNow()).Time
		if quest.QuestDate.Valid && quest.QuestDate.Time.Before(today) {
			err = q.UncompleteQuestAndResetDate(ctx, sqlc.UncompleteQuestAndResetDateParams{
				ID:        id,
				QuestDate: pgtype.Date{Time: today, Valid: true},
			})
		} else {
			err = q.UncompleteQuest(ctx, id)
		}
		if err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}

type reorderRequest struct {
	ID        int64 `json:"id"`
	SortOrder int32 `json:"sort_order"`
}

func handleReorderQuest(q sqlc.Querier) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req reorderRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if err := q.UpdateQuestSortOrder(r.Context(), sqlc.UpdateQuestSortOrderParams{
			ID:        req.ID,
			SortOrder: req.SortOrder,
		}); err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}
