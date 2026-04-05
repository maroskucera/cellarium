// Cellarium Quests — quest log handler (completed + failed)
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

type logQuestDisplay struct {
	ID             int64
	Title          string
	Status         string
	HasDescription bool
	QuestDate      string
	QuestGiver     string
	Recurring      bool
}

type logDayGroup struct {
	Date   string // "Monday, 2 January 2006"
	Quests []logQuestDisplay
}

type questLogData struct {
	Nav  string
	Days []logDayGroup
}

func handleQuestLog(q sqlc.Querier, tmpl *template.Template) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		now := timeNow()
		today := pgtype.Date{Time: now.Truncate(24 * time.Hour), Valid: true}

		if err := ensureFailedQuests(ctx, q, today); err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}

		quests, err := q.ListQuestLog(ctx)
		if err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}

		data := questLogData{Nav: "log"}
		dayIndex := make(map[string]int)

		for _, quest := range quests {
			var resolvedTime time.Time
			if quest.Status == "completed" && quest.CompletedAt.Valid {
				resolvedTime = quest.CompletedAt.Time
			} else if quest.FailedAt.Valid {
				resolvedTime = quest.FailedAt.Time
			} else {
				continue
			}

			dayKey := resolvedTime.Format("2006-01-02")
			dayLabel := resolvedTime.Format("Monday, 2 January 2006")

			d := logQuestDisplay{
				ID:             quest.ID,
				Title:          quest.Title,
				Status:         quest.Status,
				HasDescription: quest.Description.Valid && quest.Description.String != "",
				QuestGiver:     "",
				Recurring:      quest.RecurrenceType.Valid && quest.RecurrenceType.String != "",
			}

			if quest.QuestGiver.Valid {
				d.QuestGiver = quest.QuestGiver.String
			}

			if quest.QuestDate.Valid {
				d.QuestDate = quest.QuestDate.Time.Format("2 Jan")
			}

			if idx, ok := dayIndex[dayKey]; ok {
				data.Days[idx].Quests = append(data.Days[idx].Quests, d)
			} else {
				dayIndex[dayKey] = len(data.Days)
				data.Days = append(data.Days, logDayGroup{
					Date:   dayLabel,
					Quests: []logQuestDisplay{d},
				})
			}
		}

		w.Header().Set("Cache-Control", "no-store")
		renderTemplate(w, tmpl, "quest_log", data)
	})
}
