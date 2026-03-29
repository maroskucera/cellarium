// Cellarium Quests — today page handler
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
	"html/template"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/maroskucera/cellarium/quests/db/sqlc"
)

type todayData struct {
	Nav         string
	Date        string // "Monday, 2 January 2006"
	DateISO     string // "2006-01-02"
	MainQuests  []questDisplay
	SideQuests  []questDisplay
	DailyQuests []questDisplay
}

type questDisplay struct {
	ID            int64
	Title         string
	Description   string
	QuestDate     string // "2 Jan" or ""
	QuestGiver    string
	QuestLineID   int64
	QuestLineName string
	HasDate       bool
	Status        string
	Recurring     bool
}

func toQuestDisplay(q sqlc.QuestsQuest) questDisplay {
	d := questDisplay{
		ID:        q.ID,
		Title:     q.Title,
		Status:    q.Status,
		Recurring: q.RecurrenceType.Valid && q.RecurrenceType.String != "",
	}
	if q.Description.Valid {
		d.Description = q.Description.String
	}
	if q.QuestGiver.Valid {
		d.QuestGiver = q.QuestGiver.String
	}
	if q.QuestLineID.Valid {
		d.QuestLineID = q.QuestLineID.Int64
	}
	if q.QuestDate.Valid {
		d.HasDate = true
		d.QuestDate = q.QuestDate.Time.Format("2 Jan")
	}
	return d
}

func handleToday(q sqlc.Querier, tmpl *template.Template) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		now := timeNow()
		today := pgtype.Date{Time: now.Truncate(24 * time.Hour), Valid: true}

		if err := ensureFailedQuests(ctx, q, today); err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}

		quests, err := q.ListTodayQuests(ctx, today)
		if err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}

		data := todayData{
			Nav:     "today",
			Date:    now.Format("Monday, 2 January 2006"),
			DateISO: now.Format("2006-01-02"),
		}

		for _, quest := range quests {
			d := toQuestDisplay(quest)
			switch quest.QuestType {
			case "main":
				data.MainQuests = append(data.MainQuests, d)
			case "daily":
				data.DailyQuests = append(data.DailyQuests, d)
			default:
				data.SideQuests = append(data.SideQuests, d)
			}
		}

		w.Header().Set("Cache-Control", "no-store")
		renderTemplate(w, tmpl, "today", data)
	})
}
