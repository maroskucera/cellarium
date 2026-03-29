// Cellarium Quests — HTTP handler tests
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
	"html/template"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/maroskucera/cellarium/quests/db/sqlc"
)

func init() {
	timeNow = fixedTime
}

func fixedTime() time.Time {
	return time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
}

// stubQuerier implements sqlc.Querier for testing.
type stubQuerier struct {
	quests     []sqlc.QuestsQuest
	quest      sqlc.QuestsQuest
	questLines []sqlc.QuestsQuestLine
	questLine  sqlc.QuestsQuestLine
	pushSubs   []sqlc.QuestsPushSubscription
	givers     []pgtype.Text
	createdID  int64
	err        error

	failOverdueQuestsCall  *time.Time
	createQuestCalled      bool
	lastCreateQuestParams  sqlc.CreateQuestParams
	completeCalled         bool
	failCalled             bool
	sortOrderUpdate        *sqlc.UpdateQuestSortOrderParams
	listDueRemindersCalled bool
	markReminderSentCalled bool
	dueReminders           []sqlc.QuestsQuest
}

func (s *stubQuerier) CompleteQuest(_ context.Context, _ int64) error {
	s.completeCalled = true
	return s.err
}

func (s *stubQuerier) CreatePushSubscription(_ context.Context, _ sqlc.CreatePushSubscriptionParams) (int64, error) {
	return s.createdID, s.err
}

func (s *stubQuerier) CreateQuest(_ context.Context, arg sqlc.CreateQuestParams) (int64, error) {
	s.createQuestCalled = true
	s.lastCreateQuestParams = arg
	return s.createdID, s.err
}

func (s *stubQuerier) CreateQuestLine(_ context.Context, _ sqlc.CreateQuestLineParams) (int64, error) {
	return s.createdID, s.err
}

func (s *stubQuerier) DeletePushSubscription(_ context.Context, _ string) error {
	return s.err
}

func (s *stubQuerier) DeleteQuest(_ context.Context, _ int64) error {
	return s.err
}

func (s *stubQuerier) DeleteQuestLine(_ context.Context, _ int64) error {
	return s.err
}

func (s *stubQuerier) FailOverdueQuests(_ context.Context, today pgtype.Date) error {
	t := today.Time
	s.failOverdueQuestsCall = &t
	return s.err
}

func (s *stubQuerier) FailQuest(_ context.Context, _ int64) error {
	s.failCalled = true
	return s.err
}

func (s *stubQuerier) GetQuest(_ context.Context, _ int64) (sqlc.QuestsQuest, error) {
	return s.quest, s.err
}

func (s *stubQuerier) GetQuestLine(_ context.Context, _ int64) (sqlc.QuestsQuestLine, error) {
	return s.questLine, s.err
}

func (s *stubQuerier) ListActiveQuests(_ context.Context) ([]sqlc.QuestsQuest, error) {
	return s.quests, s.err
}

func (s *stubQuerier) ListDueReminders(_ context.Context, _ sqlc.ListDueRemindersParams) ([]sqlc.QuestsQuest, error) {
	s.listDueRemindersCalled = true
	return s.dueReminders, s.err
}

func (s *stubQuerier) ListPushSubscriptions(_ context.Context) ([]sqlc.QuestsPushSubscription, error) {
	return s.pushSubs, s.err
}

func (s *stubQuerier) ListQuestGivers(_ context.Context) ([]pgtype.Text, error) {
	return s.givers, s.err
}

func (s *stubQuerier) ListQuestLines(_ context.Context) ([]sqlc.QuestsQuestLine, error) {
	return s.questLines, s.err
}

func (s *stubQuerier) ListQuestLog(_ context.Context) ([]sqlc.QuestsQuest, error) {
	return s.quests, s.err
}

func (s *stubQuerier) ListTodayQuests(_ context.Context, _ pgtype.Date) ([]sqlc.QuestsQuest, error) {
	return s.quests, s.err
}

func (s *stubQuerier) MarkReminderSent(_ context.Context, _ sqlc.MarkReminderSentParams) error {
	s.markReminderSentCalled = true
	return s.err
}

func (s *stubQuerier) UpdateQuest(_ context.Context, _ sqlc.UpdateQuestParams) error {
	return s.err
}

func (s *stubQuerier) UpdateQuestLine(_ context.Context, _ sqlc.UpdateQuestLineParams) error {
	return s.err
}

func (s *stubQuerier) UpdateQuestSortOrder(_ context.Context, arg sqlc.UpdateQuestSortOrderParams) error {
	s.sortOrderUpdate = &arg
	return s.err
}

// minimal template set for tests
const testTemplates = `
{{define "today"}}<html><body>today {{.Date}} main={{len .MainQuests}} side={{len .SideQuests}} daily={{len .DailyQuests}}</body></html>{{end}}
{{define "quest_new"}}<html><body>quest-new</body></html>{{end}}
{{define "quest_edit"}}<html><body>quest-edit id={{.Quest.ID}}</body></html>{{end}}
{{define "all_quests"}}<html><body>all-quests groups={{len .Groups}} ungrouped={{len .Ungrouped}}</body></html>{{end}}
{{define "quest_log"}}<html><body>quest-log days={{len .Days}}</body></html>{{end}}
{{define "quest_lines"}}<html><body>quest-lines count={{len .QuestLines}}</body></html>{{end}}
{{define "quest_line_new"}}<html><body>quest-line-new</body></html>{{end}}
{{define "quest_line_edit"}}<html><body>quest-line-edit</body></html>{{end}}
`

func mustParseTestTemplates(t *testing.T) *template.Template {
	t.Helper()
	tmpl, err := template.New("test").Parse(testTemplates)
	if err != nil {
		t.Fatalf("failed to parse test templates: %v", err)
	}
	return tmpl
}

func TestHandleToday_empty(t *testing.T) {
	stub := &stubQuerier{}
	tmpl := mustParseTestTemplates(t)
	h := handleToday(stub, tmpl)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "today") {
		t.Errorf("expected 'today' in body, got: %s", body)
	}
	if stub.failOverdueQuestsCall == nil {
		t.Error("expected FailOverdueQuests to be called")
	}
}

func TestHandleToday_withQuests(t *testing.T) {
	stub := &stubQuerier{
		quests: []sqlc.QuestsQuest{
			{ID: 1, Title: "Main quest", QuestType: "main", Status: "active"},
			{ID: 2, Title: "Side quest", QuestType: "side", Status: "active"},
			{ID: 3, Title: "Daily quest", QuestType: "daily", Status: "active"},
		},
	}
	tmpl := mustParseTestTemplates(t)
	h := handleToday(stub, tmpl)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "main=1") {
		t.Errorf("expected main=1 in body, got: %s", body)
	}
	if !strings.Contains(body, "side=1") {
		t.Errorf("expected side=1 in body, got: %s", body)
	}
	if !strings.Contains(body, "daily=1") {
		t.Errorf("expected daily=1 in body, got: %s", body)
	}
}

func TestHandleNewQuest_get(t *testing.T) {
	stub := &stubQuerier{}
	tmpl := mustParseTestTemplates(t)
	h := handleNewQuest(stub, tmpl)

	req := httptest.NewRequest("GET", "/quests/new", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "quest-new") {
		t.Errorf("expected quest-new in body, got: %s", w.Body.String())
	}
}

func TestHandleCreateQuest_post(t *testing.T) {
	stub := &stubQuerier{}
	tmpl := mustParseTestTemplates(t)
	h := handleNewQuest(stub, tmpl)

	form := url.Values{}
	form.Set("title", "My Quest")
	form.Set("quest_type", "side")
	req := httptest.NewRequest("POST", "/quests/new", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusSeeOther {
		t.Errorf("expected 303, got %d", w.Code)
	}
	if !stub.createQuestCalled {
		t.Error("expected CreateQuest to be called")
	}
	if stub.lastCreateQuestParams.Title != "My Quest" {
		t.Errorf("expected title 'My Quest', got %q", stub.lastCreateQuestParams.Title)
	}
}

func TestHandleCompleteQuest(t *testing.T) {
	stub := &stubQuerier{
		quest: sqlc.QuestsQuest{ID: 1, Title: "Test", QuestType: "side", Status: "active"},
	}
	h := handleCompleteQuest(stub)

	req := httptest.NewRequest("POST", "/quests/1/complete", nil)
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusSeeOther {
		t.Errorf("expected 303, got %d", w.Code)
	}
	if !stub.completeCalled {
		t.Error("expected CompleteQuest to be called")
	}
}

func TestHandleFailQuest(t *testing.T) {
	stub := &stubQuerier{}
	h := handleFailQuest(stub)

	req := httptest.NewRequest("POST", "/quests/1/fail", nil)
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusSeeOther {
		t.Errorf("expected 303, got %d", w.Code)
	}
	if !stub.failCalled {
		t.Error("expected FailQuest to be called")
	}
}

func TestHandleAllQuests(t *testing.T) {
	stub := &stubQuerier{
		quests: []sqlc.QuestsQuest{
			{ID: 1, Title: "Quest A", QuestType: "side", Status: "active"},
		},
		questLines: []sqlc.QuestsQuestLine{
			{ID: 1, Name: "Line 1"},
		},
	}
	tmpl := mustParseTestTemplates(t)
	h := handleAllQuests(stub, tmpl)

	req := httptest.NewRequest("GET", "/quests", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "all-quests") {
		t.Errorf("expected all-quests in body, got: %s", w.Body.String())
	}
}

func TestHandleQuestLog(t *testing.T) {
	completedAt := pgtype.Timestamptz{Time: time.Date(2026, 3, 14, 9, 0, 0, 0, time.UTC), Valid: true}
	stub := &stubQuerier{
		quests: []sqlc.QuestsQuest{
			{ID: 1, Title: "Done quest", QuestType: "side", Status: "completed", CompletedAt: completedAt},
		},
	}
	tmpl := mustParseTestTemplates(t)
	h := handleQuestLog(stub, tmpl)

	req := httptest.NewRequest("GET", "/log", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "quest-log") {
		t.Errorf("expected quest-log in body, got: %s", w.Body.String())
	}
}

func TestHandleQuestLines(t *testing.T) {
	stub := &stubQuerier{
		questLines: []sqlc.QuestsQuestLine{
			{ID: 1, Name: "Main Story"},
			{ID: 2, Name: "Side Arc"},
		},
	}
	tmpl := mustParseTestTemplates(t)
	h := handleQuestLines(stub, tmpl)

	req := httptest.NewRequest("GET", "/quest-lines", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "count=2") {
		t.Errorf("expected count=2 in body, got: %s", w.Body.String())
	}
}

func TestHandleQuestGivers(t *testing.T) {
	stub := &stubQuerier{
		givers: []pgtype.Text{
			{String: "The King", Valid: true},
			{String: "The Wizard", Valid: true},
		},
	}
	h := handleQuestGivers(stub)

	req := httptest.NewRequest("GET", "/api/quest-givers", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "The King") {
		t.Errorf("expected 'The King' in response, got: %s", body)
	}
	if !strings.Contains(body, "The Wizard") {
		t.Errorf("expected 'The Wizard' in response, got: %s", body)
	}
}
