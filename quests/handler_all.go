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
	"sort"

	"github.com/maroskucera/cellarium/quests/db/sqlc"
)

type questLineGroup struct {
	ID     int64
	Name   string
	Quests []questDisplay
}

type typeGroup struct {
	TypeName  string
	Lines     []questLineGroup
	Ungrouped []questDisplay
}

type allQuestsData struct {
	Nav   string
	Types []typeGroup
}

var typeOrder = []string{"main", "side", "daily"}

var typeDisplayNames = map[string]string{
	"main":  "Main",
	"side":  "Side",
	"daily": "Daily",
}

func handleAllQuests(q sqlc.Querier, tx txRunner, tmpl *template.Template) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		now := timeNow()
		today := localToday(now)

		if err := ensureFailedQuests(ctx, tx, today); err != nil {
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

		// Build line name lookup and stable sort-order index
		lineNames := make(map[int64]string, len(lines))
		lineOrder := make(map[int64]int, len(lines))
		for i, l := range lines {
			lineNames[l.ID] = l.Name
			lineOrder[l.ID] = i
		}

		// Build type groups (keyed by type name)
		typeGroupMap := make(map[string]*typeGroup, len(typeOrder))
		for _, t := range typeOrder {
			typeGroupMap[t] = &typeGroup{TypeName: typeDisplayNames[t]}
		}

		// Track quest line groups within each type: (typeName, lineID) -> index in tg.Lines
		type lineKey struct {
			typeName string
			lineID   int64
		}
		lineGroupIdx := make(map[lineKey]int)

		for _, quest := range quests {
			tg := typeGroupMap[quest.QuestType]
			if tg == nil {
				continue
			}
			d := toQuestDisplay(quest)
			if quest.QuestLineID.Valid {
				lk := lineKey{quest.QuestType, quest.QuestLineID.Int64}
				idx, ok := lineGroupIdx[lk]
				if !ok {
					idx = len(tg.Lines)
					tg.Lines = append(tg.Lines, questLineGroup{
						ID:   quest.QuestLineID.Int64,
						Name: lineNames[quest.QuestLineID.Int64],
					})
					lineGroupIdx[lk] = idx
				}
				tg.Lines[idx].Quests = append(tg.Lines[idx].Quests, d)
			} else {
				tg.Ungrouped = append(tg.Ungrouped, d)
			}
		}

		data := allQuestsData{Nav: "quests"}
		for _, t := range typeOrder {
			tg := typeGroupMap[t]
			sort.Slice(tg.Lines, func(i, j int) bool {
				return lineOrder[tg.Lines[i].ID] < lineOrder[tg.Lines[j].ID]
			})
			if len(tg.Lines) > 0 || len(tg.Ungrouped) > 0 {
				data.Types = append(data.Types, *tg)
			}
		}

		w.Header().Set("Cache-Control", "no-store")
		renderTemplate(w, tmpl, "all_quests", data)
	})
}
