// Cellarium Quests — all active quests handler
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

type questLineGroup struct {
	ID     int64
	Name   string
	Quests []questDisplay
}

type allQuestsData struct {
	Nav       string
	Groups    []questLineGroup
	Ungrouped []questDisplay
}

func handleAllQuests(q sqlc.Querier, tmpl *template.Template) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		now := timeNow()
		today := pgtype.Date{Time: now.Truncate(24 * time.Hour), Valid: true}

		if err := ensureFailedQuests(ctx, q, today); err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}

		quests, err := q.ListActiveQuests(ctx)
		if err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}

		lines, err := q.ListQuestLines(ctx)
		if err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}

		// Build a map of quest line ID -> group index
		lineIndex := make(map[int64]int, len(lines))
		groups := make([]questLineGroup, 0, len(lines))
		for _, l := range lines {
			lineIndex[l.ID] = len(groups)
			groups = append(groups, questLineGroup{ID: l.ID, Name: l.Name})
		}

		data := allQuestsData{Nav: "quests"}
		for _, quest := range quests {
			d := toQuestDisplay(quest)
			if quest.QuestLineID.Valid {
				if idx, ok := lineIndex[quest.QuestLineID.Int64]; ok {
					groups[idx].Quests = append(groups[idx].Quests, d)
					continue
				}
			}
			data.Ungrouped = append(data.Ungrouped, d)
		}

		// Only include groups that have quests
		for _, g := range groups {
			if len(g.Quests) > 0 {
				data.Groups = append(data.Groups, g)
			}
		}

		w.Header().Set("Cache-Control", "no-store")
		renderTemplate(w, tmpl, "all_quests", data)
	})
}
