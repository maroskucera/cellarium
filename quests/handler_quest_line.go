// Cellarium Quests — quest line CRUD handlers
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
	"html/template"
	"net/http"
	"strconv"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/maroskucera/cellarium/quests/db/sqlc"
)

type questLinesData struct {
	Nav        string
	QuestLines []sqlc.ListQuestLinesWithCountRow
}

type questLineFormData struct {
	Nav    string
	Title  string
	Action string
	Line   *sqlc.GetQuestLineRow
	Errors map[string]string
}

type questLineDetailData struct {
	Nav    string
	Line   sqlc.GetQuestLineRow
	Quests []questDisplay
}

func handleQuestLineDetail(q sqlc.Querier, tmpl *template.Template) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}
		ctx := r.Context()
		line, err := q.GetQuestLine(ctx, id)
		if err != nil {
			http.Error(w, "quest line not found", http.StatusNotFound)
			return
		}
		quests, err := q.ListQuestsByLine(ctx, pgtype.Int8{Int64: id, Valid: true})
		if err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}
		displays := make([]questDisplay, len(quests))
		for i, quest := range quests {
			displays[i] = toQuestDisplay(quest)
		}
		data := questLineDetailData{
			Nav:    "quest-lines",
			Line:   line,
			Quests: displays,
		}
		w.Header().Set("Cache-Control", "no-store")
		renderTemplate(w, tmpl, "quest_line_detail", data)
	})
}

func handleQuestLines(q sqlc.Querier, tmpl *template.Template) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lines, err := q.ListQuestLinesWithCount(r.Context())
		if err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}
		data := questLinesData{Nav: "quest-lines", QuestLines: lines}
		w.Header().Set("Cache-Control", "no-store")
		renderTemplate(w, tmpl, "quest_lines", data)
	})
}

func parseQuestLineType(s string) pgtype.Text {
	if s == "main" || s == "side" || s == "daily" {
		return pgtype.Text{String: s, Valid: true}
	}
	return pgtype.Text{}
}

func handleNewQuestLine(q sqlc.Querier, tmpl *template.Template) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			if err := r.ParseForm(); err != nil {
				http.Error(w, "invalid form", http.StatusBadRequest)
				return
			}
			name := r.FormValue("name")
			if name == "" {
				data := questLineFormData{
					Nav:    "quest-lines",
					Title:  "New Quest Line",
					Action: "/quest-lines/new",
					Errors: map[string]string{"name": "Name is required"},
				}
				renderTemplate(w, tmpl, "quest_line_new", data)
				return
			}
			var desc pgtype.Text
			if d := r.FormValue("description"); d != "" {
				desc = pgtype.Text{String: d, Valid: true}
			}
			var sortOrder int32
			if s := r.FormValue("sort_order"); s != "" {
				n, err := strconv.ParseInt(s, 10, 32)
				if err == nil {
					sortOrder = int32(n)
				}
			}
			if _, err := q.CreateQuestLine(r.Context(), sqlc.CreateQuestLineParams{
				Name:        name,
				Description: desc,
				SortOrder:   sortOrder,
				QuestType:   parseQuestLineType(r.FormValue("quest_type")),
			}); err != nil {
				http.Error(w, "database error", http.StatusInternalServerError)
				return
			}
			http.Redirect(w, r, "/quest-lines", http.StatusSeeOther)
			return
		}
		data := questLineFormData{
			Nav:    "quest-lines",
			Title:  "New Quest Line",
			Action: "/quest-lines/new",
		}
		w.Header().Set("Cache-Control", "no-store")
		renderTemplate(w, tmpl, "quest_line_new", data)
	})
}

func handleEditQuestLine(q sqlc.Querier, tmpl *template.Template) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		idStr := r.PathValue("id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}
		ctx := r.Context()
		line, err := q.GetQuestLine(ctx, id)
		if err != nil {
			http.Error(w, "quest line not found", http.StatusNotFound)
			return
		}

		if r.Method == http.MethodPost {
			if err := r.ParseForm(); err != nil {
				http.Error(w, "invalid form", http.StatusBadRequest)
				return
			}
			name := r.FormValue("name")
			if name == "" {
				data := questLineFormData{
					Nav:    "quest-lines",
					Title:  "Edit Quest Line",
					Action: "/quest-lines/" + idStr + "/edit",
					Line:   &line,
					Errors: map[string]string{"name": "Name is required"},
				}
				renderTemplate(w, tmpl, "quest_line_edit", data)
				return
			}
			var desc pgtype.Text
			if d := r.FormValue("description"); d != "" {
				desc = pgtype.Text{String: d, Valid: true}
			}
			sortOrder := line.SortOrder
			if s := r.FormValue("sort_order"); s != "" {
				n, err := strconv.ParseInt(s, 10, 32)
				if err == nil {
					sortOrder = int32(n)
				}
			}
			questType := parseQuestLineType(r.FormValue("quest_type"))
			if err := q.UpdateQuestLine(ctx, sqlc.UpdateQuestLineParams{
				ID:          id,
				Name:        name,
				Description: desc,
				SortOrder:   sortOrder,
				QuestType:   questType,
			}); err != nil {
				http.Error(w, "database error", http.StatusInternalServerError)
				return
			}
			// Propagate type to all quests in this line when type is set
			if questType.Valid {
				if err := q.UpdateQuestTypeByLine(ctx, sqlc.UpdateQuestTypeByLineParams{
					QuestType:   questType.String,
					QuestLineID: pgtype.Int8{Int64: id, Valid: true},
				}); err != nil {
					http.Error(w, "database error", http.StatusInternalServerError)
					return
				}
			}
			http.Redirect(w, r, "/quest-lines", http.StatusSeeOther)
			return
		}

		data := questLineFormData{
			Nav:    "quest-lines",
			Title:  "Edit Quest Line",
			Action: "/quest-lines/" + idStr + "/edit",
			Line:   &line,
		}
		w.Header().Set("Cache-Control", "no-store")
		renderTemplate(w, tmpl, "quest_line_edit", data)
	})
}

func handleReorderQuestLine(q sqlc.Querier) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req reorderRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if err := q.UpdateQuestLineSortOrder(r.Context(), sqlc.UpdateQuestLineSortOrderParams{
			ID:        req.ID,
			SortOrder: req.SortOrder,
		}); err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}

func handleDeleteQuestLine(q sqlc.Querier) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}
		if err := q.DeleteQuestLine(r.Context(), id); err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/quest-lines", http.StatusSeeOther)
	})
}
