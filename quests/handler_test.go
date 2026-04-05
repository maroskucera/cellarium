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
	"embed"
	"errors"
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
	appLocation = time.UTC
}

func fixedTime() time.Time {
	return time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
}

// stubQuerier implements sqlc.Querier for testing.
type stubQuerier struct {
	quests       []sqlc.QuestsQuest
	quest        sqlc.QuestsQuest
	questLines   []sqlc.ListQuestLinesRow
	questLine    sqlc.GetQuestLineRow
	questsByLine []sqlc.QuestsQuest
	pushSubs     []sqlc.QuestsPushSubscription
	givers       []pgtype.Text
	createdID    int64
	err          error

	failedQuests                     []sqlc.QuestsQuest
	failOverdueQuestsCall            *time.Time
	createQuestCalled                bool
	lastCreateQuestParams            sqlc.CreateQuestParams
	createQuestCalls                 []sqlc.CreateQuestParams
	completeCalled                   bool
	failCalled                       bool
	uncompleteCalled                 bool
	uncompleteResetDateCalled        bool
	lastUncompleteResetDateParam     sqlc.UncompleteQuestAndResetDateParams
	listActiveCalled                 bool
	sortOrderUpdate                  *sqlc.UpdateQuestSortOrderParams
	listDueRemindersCalled           bool
	markReminderSentCalled           bool
	dueReminders                     []sqlc.QuestsQuest
	lastCreateQuestLineParams        sqlc.CreateQuestLineParams
	updateQuestTypeByLineCalled      bool
	lastUpdateQuestTypeByLineParams  sqlc.UpdateQuestTypeByLineParams
	lastCreatePushSubscriptionParams sqlc.CreatePushSubscriptionParams
}

func (s *stubQuerier) CompleteQuest(_ context.Context, _ int64) error {
	s.completeCalled = true
	return s.err
}

func (s *stubQuerier) CreatePushSubscription(_ context.Context, arg sqlc.CreatePushSubscriptionParams) (int64, error) {
	s.lastCreatePushSubscriptionParams = arg
	return s.createdID, s.err
}

func (s *stubQuerier) CreateQuest(_ context.Context, arg sqlc.CreateQuestParams) (int64, error) {
	s.createQuestCalled = true
	s.lastCreateQuestParams = arg
	s.createQuestCalls = append(s.createQuestCalls, arg)
	return s.createdID, s.err
}

func (s *stubQuerier) CreateQuestLine(_ context.Context, arg sqlc.CreateQuestLineParams) (int64, error) {
	s.lastCreateQuestLineParams = arg
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

func (s *stubQuerier) FailOverdueQuests(_ context.Context, today pgtype.Date) ([]sqlc.QuestsQuest, error) {
	t := today.Time
	s.failOverdueQuestsCall = &t
	return s.failedQuests, s.err
}

func (s *stubQuerier) FailQuest(_ context.Context, _ int64) error {
	s.failCalled = true
	return s.err
}

func (s *stubQuerier) GetQuest(_ context.Context, _ int64) (sqlc.QuestsQuest, error) {
	return s.quest, s.err
}

func (s *stubQuerier) GetQuestLine(_ context.Context, _ int64) (sqlc.GetQuestLineRow, error) {
	return s.questLine, s.err
}

func (s *stubQuerier) ListActiveAndCompletedQuests(_ context.Context) ([]sqlc.QuestsQuest, error) {
	return s.quests, s.err
}

func (s *stubQuerier) ListActiveQuests(_ context.Context) ([]sqlc.QuestsQuest, error) {
	s.listActiveCalled = true
	return s.quests, s.err
}

func (s *stubQuerier) ListQuestsByLine(_ context.Context, _ pgtype.Int8) ([]sqlc.QuestsQuest, error) {
	return s.questsByLine, s.err
}

func (s *stubQuerier) UncompleteQuest(_ context.Context, _ int64) error {
	s.uncompleteCalled = true
	return s.err
}

func (s *stubQuerier) UncompleteQuestAndResetDate(_ context.Context, arg sqlc.UncompleteQuestAndResetDateParams) error {
	s.uncompleteResetDateCalled = true
	s.lastUncompleteResetDateParam = arg
	return s.err
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

func (s *stubQuerier) ListQuestLines(_ context.Context) ([]sqlc.ListQuestLinesRow, error) {
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

func (s *stubQuerier) UpdateQuestLineSortOrder(_ context.Context, _ sqlc.UpdateQuestLineSortOrderParams) error {
	return s.err
}

func (s *stubQuerier) UpdateQuestTypeByLine(_ context.Context, arg sqlc.UpdateQuestTypeByLineParams) error {
	s.updateQuestTypeByLineCalled = true
	s.lastUpdateQuestTypeByLineParams = arg
	return s.err
}

// stubTxRunner wraps a stubQuerier for transaction-based tests.
type stubTxRunner struct {
	q *stubQuerier
}

func (r *stubTxRunner) RunInTx(_ context.Context, fn func(sqlc.Querier) error) error {
	return fn(r.q)
}

//go:embed templates/*
var testTemplatesFS embed.FS

// minimal template set for tests
const testTemplates = `
{{define "today"}}<html><body>today {{.Date}} main={{len .MainQuests}} side={{len .SideQuests}} daily={{len .DailyQuests}}{{range .MainQuests}}<a href="/quests/{{.ID}}/edit" class="quest-card-link">{{.Title}}{{if .HasDescription}} 📝{{end}}</a>{{end}}</body></html>{{end}}
{{define "quest_new"}}<html><body>quest-new title={{.Quest.Title}}</body></html>{{end}}
{{define "quest_edit"}}<html><body>quest-edit id={{.Quest.ID}}<div class="btn-row"><button type="submit">Save Changes</button><a href="/" class="btn btn-secondary">Cancel</a><form method="post" action="/quests/{{.Quest.ID}}/delete"><button type="submit" class="btn-danger-sm">Delete</button></form></div></body></html>{{end}}
{{define "all_quests"}}<html><body>all-quests types={{len .Types}}</body></html>{{end}}
{{define "quest_log"}}<html><body>quest-log days={{len .Days}}{{range .Days}}{{range .Quests}}<a href="/quests/{{.ID}}">{{.Title}}</a>{{if eq .Status "failed"}}<a href="/quests/{{.ID}}/retry" class="btn quest-retry-btn">↻</a>{{end}}{{end}}{{end}}</body></html>{{end}}
{{define "quest_detail"}}<html><body>quest-detail id={{.Quest.ID}} title={{.Quest.Title}} status={{.Quest.Status}}{{if .Quest.Description}} desc={{.Quest.Description}}{{end}}{{if .Quest.QuestLineName}} line={{.Quest.QuestLineName}}{{end}}{{if .Quest.QuestGiver}} giver={{.Quest.QuestGiver}}{{end}}{{if .Quest.QuestDate}} date={{.Quest.QuestDate}}{{end}}{{if .Quest.ReminderTime}} reminder={{.Quest.ReminderTime}}{{end}}{{if .Quest.RecurrenceType}} recurrence={{.Quest.RecurrenceType}}{{end}}{{if eq .Quest.Status "failed"}} <a href="/quests/{{.Quest.ID}}/retry">Retry</a>{{end}}</body></html>{{end}}
{{define "quest_retry"}}<html><body>quest-retry id={{.QuestID}} name={{.QuestName}} date={{.QuestDate}}</body></html>{{end}}
{{define "quest_lines"}}<html><body>quest-lines count={{len .QuestLines}}</body></html>{{end}}
{{define "quest_line_new"}}<html><body>quest-line-new</body></html>{{end}}
{{define "quest_line_edit"}}<html><body>quest-line-edit</body></html>{{end}}
{{define "quest_line_detail"}}<html><body>quest-line-detail name={{.Line.Name}} count={{len .Quests}}{{range .Quests}} [{{.Title}}:{{.Status}}]{{end}}</body></html>{{end}}
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
	h := handleToday(stub, &stubTxRunner{q: stub}, tmpl, false)

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
			{ID: 1, Title: "Main", QuestType: "main", Status: "active"},
			{ID: 2, Title: "Side", QuestType: "side", Status: "active"},
			{ID: 3, Title: "Daily", QuestType: "daily", Status: "active"},
		},
	}
	tmpl := mustParseTestTemplates(t)
	h := handleToday(stub, &stubTxRunner{q: stub}, tmpl, false)

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

func TestHandleCreateQuest_postInvalidDate(t *testing.T) {
	stub := &stubQuerier{}
	tmpl := mustParseTestTemplates(t)
	h := handleNewQuest(stub, tmpl)

	form := url.Values{}
	form.Set("title", "My Quest")
	form.Set("quest_date", "not-a-date")
	req := httptest.NewRequest("POST", "/quests/new", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
	if stub.createQuestCalled {
		t.Error("expected CreateQuest NOT to be called")
	}
}

func TestHandleNewQuest_typeInheritance(t *testing.T) {
	// Quest line has type "main"; submitted form has type "side" — server should override to "main"
	stub := &stubQuerier{
		questLine: sqlc.GetQuestLineRow{
			ID:        1,
			Name:      "Main Line",
			QuestType: pgtype.Text{String: "main", Valid: true},
		},
	}
	tmpl := mustParseTestTemplates(t)
	h := handleNewQuest(stub, tmpl)

	form := url.Values{}
	form.Set("title", "Inherited Quest")
	form.Set("quest_type", "side")
	form.Set("quest_line_id", "1")
	req := httptest.NewRequest("POST", "/quests/new", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusSeeOther {
		t.Errorf("expected 303, got %d", w.Code)
	}
	if stub.lastCreateQuestParams.QuestType != "main" {
		t.Errorf("expected quest type 'main' (inherited from line), got %q", stub.lastCreateQuestParams.QuestType)
	}
}

func TestHandleCreateQuest_endDateMode(t *testing.T) {
	stub := &stubQuerier{}
	tmpl := mustParseTestTemplates(t)
	h := handleNewQuest(stub, tmpl)

	form := url.Values{}
	form.Set("title", "Recurring end date")
	form.Set("quest_type", "side")
	form.Set("quest_date", "2026-04-01")
	form.Set("recurrence_type", "every")
	form.Set("recurrence_n", "1")
	form.Set("recurrence_unit", "days")
	form.Set("recurrence_end_mode", "end_date")
	form.Set("recurrence_end_date", "2026-04-30")
	req := httptest.NewRequest("POST", "/quests/new", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusSeeOther {
		t.Errorf("expected 303, got %d", w.Code)
	}
	if !stub.lastCreateQuestParams.RecurrenceEndDate.Valid {
		t.Fatal("expected RecurrenceEndDate to be set")
	}
	wantEnd := time.Date(2026, 4, 30, 0, 0, 0, 0, time.UTC)
	if !stub.lastCreateQuestParams.RecurrenceEndDate.Time.Equal(wantEnd) {
		t.Errorf("expected end date %v, got %v", wantEnd, stub.lastCreateQuestParams.RecurrenceEndDate.Time)
	}
	if stub.lastCreateQuestParams.RecurrenceInstance.Valid {
		t.Error("expected RecurrenceInstance to be NULL for end_date mode")
	}
	if stub.lastCreateQuestParams.RecurrenceMaxInstances.Valid {
		t.Error("expected RecurrenceMaxInstances to be NULL for end_date mode")
	}
}

func TestHandleCreateQuest_endAfterMode(t *testing.T) {
	stub := &stubQuerier{}
	tmpl := mustParseTestTemplates(t)
	h := handleNewQuest(stub, tmpl)

	form := url.Values{}
	form.Set("title", "Recurring end after")
	form.Set("quest_type", "side")
	form.Set("quest_date", "2026-04-01")
	form.Set("recurrence_type", "every")
	form.Set("recurrence_n", "1")
	form.Set("recurrence_unit", "days")
	form.Set("recurrence_end_mode", "end_after")
	form.Set("recurrence_max_instances", "10")
	req := httptest.NewRequest("POST", "/quests/new", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusSeeOther {
		t.Errorf("expected 303, got %d", w.Code)
	}
	if !stub.lastCreateQuestParams.RecurrenceMaxInstances.Valid || stub.lastCreateQuestParams.RecurrenceMaxInstances.Int32 != 10 {
		t.Errorf("expected max instances 10, got %v", stub.lastCreateQuestParams.RecurrenceMaxInstances)
	}
	if !stub.lastCreateQuestParams.RecurrenceInstance.Valid || stub.lastCreateQuestParams.RecurrenceInstance.Int32 != 1 {
		t.Errorf("expected instance 1, got %v", stub.lastCreateQuestParams.RecurrenceInstance)
	}
	if stub.lastCreateQuestParams.RecurrenceEndDate.Valid {
		t.Error("expected RecurrenceEndDate to be NULL for end_after mode")
	}
}

func TestHandleCreateQuest_endNeverMode(t *testing.T) {
	stub := &stubQuerier{}
	tmpl := mustParseTestTemplates(t)
	h := handleNewQuest(stub, tmpl)

	form := url.Values{}
	form.Set("title", "Recurring never end")
	form.Set("quest_type", "side")
	form.Set("recurrence_type", "every")
	form.Set("recurrence_n", "1")
	form.Set("recurrence_unit", "days")
	form.Set("recurrence_end_mode", "never")
	req := httptest.NewRequest("POST", "/quests/new", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusSeeOther {
		t.Errorf("expected 303, got %d", w.Code)
	}
	if stub.lastCreateQuestParams.RecurrenceEndDate.Valid {
		t.Error("expected RecurrenceEndDate to be NULL for never mode")
	}
	if stub.lastCreateQuestParams.RecurrenceMaxInstances.Valid {
		t.Error("expected RecurrenceMaxInstances to be NULL for never mode")
	}
	if stub.lastCreateQuestParams.RecurrenceInstance.Valid {
		t.Error("expected RecurrenceInstance to be NULL for never mode")
	}
}

func TestHandleCreateQuest_afterCompletionIgnoresEndSettings(t *testing.T) {
	stub := &stubQuerier{}
	tmpl := mustParseTestTemplates(t)
	h := handleNewQuest(stub, tmpl)

	form := url.Values{}
	form.Set("title", "After completion")
	form.Set("quest_type", "side")
	form.Set("recurrence_type", "after_completion")
	form.Set("recurrence_n", "3")
	form.Set("recurrence_unit", "days")
	form.Set("recurrence_end_mode", "end_after")
	form.Set("recurrence_max_instances", "5")
	req := httptest.NewRequest("POST", "/quests/new", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusSeeOther {
		t.Errorf("expected 303, got %d", w.Code)
	}
	if stub.lastCreateQuestParams.RecurrenceEndDate.Valid {
		t.Error("expected RecurrenceEndDate to be NULL for after_completion")
	}
	if stub.lastCreateQuestParams.RecurrenceMaxInstances.Valid {
		t.Error("expected RecurrenceMaxInstances to be NULL for after_completion")
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

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
	if !stub.completeCalled {
		t.Error("expected CompleteQuest to be called")
	}
}

func TestHandleUncompleteQuest_noDate(t *testing.T) {
	stub := &stubQuerier{
		quest: sqlc.QuestsQuest{ID: 1, Title: "Test", QuestType: "side", Status: "completed"},
	}
	h := handleUncompleteQuest(stub)

	req := httptest.NewRequest("POST", "/quests/1/uncomplete", nil)
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
	if !stub.uncompleteCalled {
		t.Error("expected UncompleteQuest to be called")
	}
	if stub.uncompleteResetDateCalled {
		t.Error("expected UncompleteQuestAndResetDate NOT to be called")
	}
}

func TestHandleUncompleteQuest_futureDate(t *testing.T) {
	// quest_date >= today: use UncompleteQuest (no date reset)
	futureDate := pgtype.Date{Time: time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC), Valid: true}
	stub := &stubQuerier{
		quest: sqlc.QuestsQuest{ID: 2, Title: "Future", QuestType: "main", Status: "completed", QuestDate: futureDate},
	}
	h := handleUncompleteQuest(stub)

	req := httptest.NewRequest("POST", "/quests/2/uncomplete", nil)
	req.SetPathValue("id", "2")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
	if !stub.uncompleteCalled {
		t.Error("expected UncompleteQuest to be called")
	}
}

func TestHandleUncompleteQuest_pastDate(t *testing.T) {
	// quest_date < today: use UncompleteQuestAndResetDate with today
	pastDate := pgtype.Date{Time: time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC), Valid: true}
	stub := &stubQuerier{
		quest: sqlc.QuestsQuest{ID: 3, Title: "Past", QuestType: "daily", Status: "completed", QuestDate: pastDate},
	}
	h := handleUncompleteQuest(stub)

	req := httptest.NewRequest("POST", "/quests/3/uncomplete", nil)
	req.SetPathValue("id", "3")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
	if stub.uncompleteCalled {
		t.Error("expected UncompleteQuest NOT to be called")
	}
	if !stub.uncompleteResetDateCalled {
		t.Error("expected UncompleteQuestAndResetDate to be called")
	}
	expectedDate := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)
	if stub.lastUncompleteResetDateParam.QuestDate.Time != expectedDate {
		t.Errorf("expected date %v, got %v", expectedDate, stub.lastUncompleteResetDateParam.QuestDate.Time)
	}
}

func TestHandleUncompleteQuest_invalidID(t *testing.T) {
	stub := &stubQuerier{}
	h := handleUncompleteQuest(stub)

	req := httptest.NewRequest("POST", "/quests/abc/uncomplete", nil)
	req.SetPathValue("id", "abc")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandleAllQuests(t *testing.T) {
	stub := &stubQuerier{
		quests: []sqlc.QuestsQuest{
			{ID: 1, Title: "Quest A", QuestType: "side", Status: "active"},
		},
		questLines: []sqlc.ListQuestLinesRow{
			{ID: 1, Name: "Line 1"},
		},
	}
	tmpl := mustParseTestTemplates(t)
	h := handleAllQuests(stub, &stubTxRunner{q: stub}, tmpl)

	req := httptest.NewRequest("GET", "/quests", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "all-quests") {
		t.Errorf("expected all-quests in body, got: %s", w.Body.String())
	}
	if !stub.listActiveCalled {
		t.Error("expected ListActiveQuests to be called")
	}
}

func TestHandleAllQuests_typeGrouping(t *testing.T) {
	// One main quest in a quest line, one side quest ungrouped → two type groups
	stub := &stubQuerier{
		quests: []sqlc.QuestsQuest{
			{ID: 1, Title: "Main quest", QuestType: "main", Status: "active", QuestLineID: pgtype.Int8{Int64: 1, Valid: true}},
			{ID: 2, Title: "Side quest", QuestType: "side", Status: "active"},
		},
		questLines: []sqlc.ListQuestLinesRow{
			{ID: 1, Name: "Main Line"},
		},
	}
	tmpl := mustParseTestTemplates(t)
	h := handleAllQuests(stub, &stubTxRunner{q: stub}, tmpl)

	req := httptest.NewRequest("GET", "/quests", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "types=2") {
		t.Errorf("expected types=2 in body, got: %s", body)
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
	h := handleQuestLog(stub, &stubTxRunner{q: stub}, tmpl)

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
		questLines: []sqlc.ListQuestLinesRow{
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

func TestHandleNewQuestLine_withType(t *testing.T) {
	stub := &stubQuerier{}
	tmpl := mustParseTestTemplates(t)
	h := handleNewQuestLine(stub, tmpl)

	form := url.Values{}
	form.Set("name", "Main Line")
	form.Set("quest_type", "main")
	req := httptest.NewRequest("POST", "/quest-lines/new", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusSeeOther {
		t.Errorf("expected 303, got %d", w.Code)
	}
	if !stub.lastCreateQuestLineParams.QuestType.Valid {
		t.Error("expected quest_type to be set")
	}
	if stub.lastCreateQuestLineParams.QuestType.String != "main" {
		t.Errorf("expected quest_type 'main', got %q", stub.lastCreateQuestLineParams.QuestType.String)
	}
}

func TestHandleEditQuestLine_propagatesType(t *testing.T) {
	stub := &stubQuerier{
		questLine: sqlc.GetQuestLineRow{ID: 1, Name: "Arc", SortOrder: 0},
	}
	tmpl := mustParseTestTemplates(t)
	h := handleEditQuestLine(stub, tmpl)

	form := url.Values{}
	form.Set("name", "Arc")
	form.Set("quest_type", "side")
	req := httptest.NewRequest("POST", "/quest-lines/1/edit", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusSeeOther {
		t.Errorf("expected 303, got %d", w.Code)
	}
	if !stub.updateQuestTypeByLineCalled {
		t.Error("expected UpdateQuestTypeByLine to be called when type is set")
	}
	if stub.lastUpdateQuestTypeByLineParams.QuestType != "side" {
		t.Errorf("expected quest_type 'side', got %q", stub.lastUpdateQuestTypeByLineParams.QuestType)
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

func TestHandleToday_noQuestDateInOutput(t *testing.T) {
	// Test that quest date (📅) is NOT displayed in today view
	// We check for the specific pattern "📅 20 Mar" (formatted date) rather than just the emoji,
	// so quests with 📅 in their title won't cause false positives
	stub := &stubQuerier{
		quests: []sqlc.QuestsQuest{
			{ID: 1, Title: "Quest with 📅 emoji in title", QuestType: "main", Status: "active", QuestDate: pgtype.Date{Time: time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC), Valid: true}},
		},
	}
	tmpl := mustParseTestTemplates(t)
	h := handleToday(stub, &stubTxRunner{q: stub}, tmpl, false)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	// Check that the formatted date pattern isn't present (emoji + space + day + space + month)
	if strings.Contains(body, "📅 20 Mar") {
		t.Errorf("expected no formatted quest date (📅 20 Mar) in output, got: %s", body)
	}
}

func TestHandleEditQuest_buttonsOnSameLine(t *testing.T) {
	// Test that Save, Cancel, Delete buttons are siblings in a flex container
	stub := &stubQuerier{
		quest: sqlc.QuestsQuest{ID: 1, Title: "Test", QuestType: "side", Status: "active"},
	}
	tmpl := mustParseTestTemplates(t)
	h := handleEditQuest(stub, tmpl)

	req := httptest.NewRequest("GET", "/quests/1/edit", nil)
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	// Check that buttons are present
	if !strings.Contains(body, "Save Changes") {
		t.Errorf("expected 'Save Changes' button, got: %s", body)
	}
	if !strings.Contains(body, "Cancel") {
		t.Errorf("expected 'Cancel' link, got: %s", body)
	}
	if !strings.Contains(body, "Delete") {
		t.Errorf("expected 'Delete' button, got: %s", body)
	}
	// Check for flex container class
	if !strings.Contains(body, "btn-row") {
		t.Errorf("expected 'btn-row' class for flex container, got: %s", body)
	}
}

func TestHandleToday_questCardClickable(t *testing.T) {
	// Test that quest card body is wrapped in a link to edit page
	stub := &stubQuerier{
		quests: []sqlc.QuestsQuest{
			{ID: 1, Title: "Clickable Quest", QuestType: "main", Status: "active"},
		},
	}
	tmpl := mustParseTestTemplates(t)
	h := handleToday(stub, &stubTxRunner{q: stub}, tmpl, false)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	// Check for quest-card-link class
	if !strings.Contains(body, "quest-card-link") {
		t.Errorf("expected 'quest-card-link' class on clickable card, got: %s", body)
	}
	// Check that Edit button is NOT present
	if strings.Contains(body, "Edit") {
		t.Errorf("expected no 'Edit' button in output, got: %s", body)
	}
	// Check that the link goes to /quests/1/edit
	if !strings.Contains(body, "/quests/1/edit") {
		t.Errorf("expected link to /quests/1/edit, got: %s", body)
	}
}

func TestHandleAllQuests_questCardClickable(t *testing.T) {
	// Test that quest card body is wrapped in a link to edit page on all_quests page
	stub := &stubQuerier{
		quests: []sqlc.QuestsQuest{
			{ID: 1, Title: "Clickable Quest", QuestType: "side", Status: "active"},
		},
		questLines: []sqlc.ListQuestLinesRow{
			{ID: 1, Name: "Test Line"},
		},
	}
	tmpl := mustParseTestTemplates(t)
	h := handleAllQuests(stub, &stubTxRunner{q: stub}, tmpl)

	req := httptest.NewRequest("GET", "/quests", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	// Test templates don't render quest cards with edit links, so we just verify the template renders
	if !strings.Contains(body, "all-quests") {
		t.Errorf("expected 'all-quests' in body, got: %s", body)
	}
}

func TestHandleToday_descriptionIcon(t *testing.T) {
	// Test that 📝 icon appears when quest has description, but description text is hidden
	desc := pgtype.Text{String: "This is a description", Valid: true}
	stub := &stubQuerier{
		quests: []sqlc.QuestsQuest{
			{ID: 1, Title: "Quest with desc", QuestType: "main", Status: "active", Description: desc},
		},
	}
	tmpl := mustParseTestTemplates(t)
	h := handleToday(stub, &stubTxRunner{q: stub}, tmpl, false)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	// Description text should NOT be in output
	if strings.Contains(body, "This is a description") {
		t.Errorf("expected no description text in output, got: %s", body)
	}
	// 📝 icon should be in output
	if !strings.Contains(body, "📝") {
		t.Errorf("expected memo icon 📝 in output, got: %s", body)
	}
}

func TestHandleToday_noDescriptionIcon(t *testing.T) {
	// Test that 📝 icon does NOT appear when quest has no description
	stub := &stubQuerier{
		quests: []sqlc.QuestsQuest{
			{ID: 1, Title: "Quest without desc", QuestType: "main", Status: "active"},
		},
	}
	tmpl := mustParseTestTemplates(t)
	h := handleToday(stub, &stubTxRunner{q: stub}, tmpl, false)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	// 📝 icon should NOT be in output
	if strings.Contains(body, "📝") {
		t.Errorf("expected no memo icon 📝 in output, got: %s", body)
	}
}

func TestHandleEditQuest_failedRedirects(t *testing.T) {
	stub := &stubQuerier{
		quest: sqlc.QuestsQuest{
			ID:        1,
			Title:     "Failed",
			QuestType: "main",
			Status:    "failed",
		},
	}
	tmpl := mustParseTestTemplates(t)
	h := handleEditQuest(stub, tmpl)

	req := httptest.NewRequest("GET", "/quests/1/edit", nil)
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusSeeOther {
		t.Errorf("expected 303, got %d", w.Code)
	}
	loc := w.Header().Get("Location")
	if loc != "/quests/1/retry" {
		t.Errorf("expected Location '/quests/1/retry', got %q", loc)
	}
}

func TestHandleEditQuest_failedPostRedirects(t *testing.T) {
	stub := &stubQuerier{
		quest: sqlc.QuestsQuest{
			ID:        1,
			Title:     "Failed",
			QuestType: "main",
			Status:    "failed",
		},
	}
	tmpl := mustParseTestTemplates(t)
	h := handleEditQuest(stub, tmpl)

	req := httptest.NewRequest("POST", "/quests/1/edit", nil)
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusSeeOther {
		t.Errorf("expected 303, got %d", w.Code)
	}
	loc := w.Header().Get("Location")
	if loc != "/quests/1/retry" {
		t.Errorf("expected Location '/quests/1/retry', got %q", loc)
	}
}

func TestHandleQuestLog_failedQuestRetryLink(t *testing.T) {
	stub := &stubQuerier{
		quests: []sqlc.QuestsQuest{
			{
				ID:        5,
				Title:     "Failed Quest",
				QuestType: "main",
				Status:    "failed",
				FailedAt:  pgtype.Timestamptz{Time: time.Date(2026, 3, 14, 9, 0, 0, 0, time.UTC), Valid: true},
			},
		},
	}
	tmpl := mustParseTestTemplates(t)
	h := handleQuestLog(stub, &stubTxRunner{q: stub}, tmpl)

	req := httptest.NewRequest("GET", "/log", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, `<a href="/quests/5">`) {
		t.Errorf("expected detail link '/quests/5' in body, got: %s", body)
	}
	if !strings.Contains(body, "/quests/5/retry") {
		t.Errorf("expected retry button '/quests/5/retry' in body, got: %s", body)
	}
	if strings.Contains(body, "/quests/5/edit") {
		t.Errorf("expected no edit link '/quests/5/edit' for failed quest, got: %s", body)
	}
}

func TestHandleRetryQuest_get(t *testing.T) {
	stub := &stubQuerier{
		quest: sqlc.QuestsQuest{
			ID:          1,
			Title:       "Failed Quest",
			QuestType:   "main",
			Status:      "failed",
			Description: pgtype.Text{String: "desc", Valid: true},
			QuestGiver:  pgtype.Text{String: "The King", Valid: true},
		},
	}
	tmpl := mustParseTestTemplates(t)
	h := handleRetryQuest(stub, tmpl)

	req := httptest.NewRequest("GET", "/quests/1/retry", nil)
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "Failed Quest") {
		t.Errorf("expected 'Failed Quest' in body, got: %s", body)
	}
	if !strings.Contains(body, "quest-retry") {
		t.Errorf("expected 'quest-retry' in body, got: %s", body)
	}
}

func TestHandleRetryQuest_getNonFailed(t *testing.T) {
	stub := &stubQuerier{
		quest: sqlc.QuestsQuest{
			ID:        2,
			Title:     "Active Quest",
			QuestType: "side",
			Status:    "active",
		},
	}
	tmpl := mustParseTestTemplates(t)
	h := handleRetryQuest(stub, tmpl)

	req := httptest.NewRequest("GET", "/quests/2/retry", nil)
	req.SetPathValue("id", "2")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusSeeOther {
		t.Errorf("expected 303, got %d", w.Code)
	}
	loc := w.Header().Get("Location")
	if loc != "/quests/2/edit" {
		t.Errorf("expected Location '/quests/2/edit', got %q", loc)
	}
}

func TestHandleRetryQuest_getNotFound(t *testing.T) {
	stub := &stubQuerier{
		err: errors.New("not found"),
	}
	tmpl := mustParseTestTemplates(t)
	h := handleRetryQuest(stub, tmpl)

	req := httptest.NewRequest("GET", "/quests/99/retry", nil)
	req.SetPathValue("id", "99")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestHandleRetryQuest_post(t *testing.T) {
	stub := &stubQuerier{
		quest: sqlc.QuestsQuest{
			ID:             1,
			Title:          "Failed Quest",
			QuestType:      "side",
			Status:         "failed",
			Description:    pgtype.Text{String: "desc", Valid: true},
			QuestGiver:     pgtype.Text{String: "The King", Valid: true},
			QuestLineID:    pgtype.Int8{Int64: 3, Valid: true},
			ReminderTime:   pgtype.Time{Microseconds: 36000000000, Valid: true},
			RecurrenceType: pgtype.Text{String: "every", Valid: true},
			RecurrenceN:    pgtype.Int4{Int32: 7, Valid: true},
			RecurrenceUnit: pgtype.Text{String: "days", Valid: true},
		},
	}
	tmpl := mustParseTestTemplates(t)
	h := handleRetryQuest(stub, tmpl)

	form := url.Values{}
	form.Set("quest_date", "2026-04-10")
	req := httptest.NewRequest("POST", "/quests/1/retry", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusSeeOther {
		t.Errorf("expected 303, got %d", w.Code)
	}
	loc := w.Header().Get("Location")
	if loc != "/" {
		t.Errorf("expected Location '/', got %q", loc)
	}
	if !stub.createQuestCalled {
		t.Error("expected CreateQuest to be called")
	}
	p := stub.lastCreateQuestParams
	if p.Title != "Failed Quest" {
		t.Errorf("expected title 'Failed Quest', got %q", p.Title)
	}
	if p.QuestType != "side" {
		t.Errorf("expected quest_type 'side', got %q", p.QuestType)
	}
	if p.Description.String != "desc" || !p.Description.Valid {
		t.Errorf("expected description 'desc' (valid), got %q (valid=%v)", p.Description.String, p.Description.Valid)
	}
	if p.QuestGiver.String != "The King" || !p.QuestGiver.Valid {
		t.Errorf("expected quest_giver 'The King' (valid), got %q (valid=%v)", p.QuestGiver.String, p.QuestGiver.Valid)
	}
	if p.QuestLineID.Int64 != 3 || !p.QuestLineID.Valid {
		t.Errorf("expected quest_line_id 3 (valid), got %d (valid=%v)", p.QuestLineID.Int64, p.QuestLineID.Valid)
	}
	expectedDate := time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)
	if !p.QuestDate.Valid || p.QuestDate.Time != expectedDate {
		t.Errorf("expected quest_date 2026-04-10 (valid), got %v (valid=%v)", p.QuestDate.Time, p.QuestDate.Valid)
	}
	if p.ReminderTime.Microseconds != 36000000000 || !p.ReminderTime.Valid {
		t.Errorf("expected reminder_time 36000000000 (valid), got %d (valid=%v)", p.ReminderTime.Microseconds, p.ReminderTime.Valid)
	}
	if p.RecurrenceType.String != "every" || !p.RecurrenceType.Valid {
		t.Errorf("expected recurrence_type 'every' (valid), got %q (valid=%v)", p.RecurrenceType.String, p.RecurrenceType.Valid)
	}
	if p.RecurrenceN.Int32 != 7 || !p.RecurrenceN.Valid {
		t.Errorf("expected recurrence_n 7 (valid), got %d (valid=%v)", p.RecurrenceN.Int32, p.RecurrenceN.Valid)
	}
	if p.RecurrenceUnit.String != "days" || !p.RecurrenceUnit.Valid {
		t.Errorf("expected recurrence_unit 'days' (valid), got %q (valid=%v)", p.RecurrenceUnit.String, p.RecurrenceUnit.Valid)
	}
}

func TestHandleRetryQuest_postNoDate(t *testing.T) {
	stub := &stubQuerier{
		quest: sqlc.QuestsQuest{
			ID:        1,
			Title:     "No Date Quest",
			QuestType: "main",
			Status:    "failed",
		},
	}
	tmpl := mustParseTestTemplates(t)
	h := handleRetryQuest(stub, tmpl)

	form := url.Values{}
	req := httptest.NewRequest("POST", "/quests/1/retry", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusSeeOther {
		t.Errorf("expected 303, got %d", w.Code)
	}
	loc := w.Header().Get("Location")
	if loc != "/" {
		t.Errorf("expected Location '/', got %q", loc)
	}
	if !stub.createQuestCalled {
		t.Error("expected CreateQuest to be called")
	}
	if stub.lastCreateQuestParams.QuestDate.Valid {
		t.Errorf("expected quest_date to be invalid (no date), got valid with %v", stub.lastCreateQuestParams.QuestDate.Time)
	}
}

func TestHandleRetryQuest_postInvalidDate(t *testing.T) {
	stub := &stubQuerier{
		quest: sqlc.QuestsQuest{
			ID:        1,
			Title:     "Failed Quest",
			QuestType: "main",
			Status:    "failed",
		},
	}
	tmpl := mustParseTestTemplates(t)
	h := handleRetryQuest(stub, tmpl)

	form := url.Values{}
	form.Set("quest_date", "not-a-date")
	req := httptest.NewRequest("POST", "/quests/1/retry", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
	if stub.createQuestCalled {
		t.Error("expected CreateQuest NOT to be called")
	}
}

func TestHandleViewQuest_completed(t *testing.T) {
	stub := &stubQuerier{
		quest: sqlc.QuestsQuest{
			ID:           1,
			Title:        "Done Quest",
			QuestType:    "main",
			Status:       "completed",
			Description:  pgtype.Text{String: "A description", Valid: true},
			QuestGiver:   pgtype.Text{String: "The King", Valid: true},
			QuestDate:    pgtype.Date{Time: time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC), Valid: true},
			QuestLineID:  pgtype.Int8{Int64: 1, Valid: true},
			ReminderTime: pgtype.Time{Microseconds: 9*3600_000_000 + 30*60_000_000, Valid: true},
			CompletedAt:  pgtype.Timestamptz{Time: time.Date(2026, 3, 10, 14, 30, 0, 0, time.UTC), Valid: true},
		},
		questLine: sqlc.GetQuestLineRow{
			ID:   1,
			Name: "Main Story",
		},
	}
	tmpl := mustParseTestTemplates(t)
	h := handleViewQuest(stub, tmpl)

	req := httptest.NewRequest("GET", "/quests/1", nil)
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "quest-detail") {
		t.Errorf("expected 'quest-detail' in body, got: %s", body)
	}
	if !strings.Contains(body, "Done Quest") {
		t.Errorf("expected 'Done Quest' in body, got: %s", body)
	}
	if !strings.Contains(body, "giver=The King") {
		t.Errorf("expected 'giver=The King' in body, got: %s", body)
	}
	if !strings.Contains(body, "line=Main Story") {
		t.Errorf("expected 'line=Main Story' in body, got: %s", body)
	}
	if !strings.Contains(body, "desc=A description") {
		t.Errorf("expected 'desc=A description' in body, got: %s", body)
	}
	if !strings.Contains(body, "reminder=09:30") {
		t.Errorf("expected 'reminder=09:30' in body, got: %s", body)
	}
	if strings.Contains(body, "/retry") {
		t.Errorf("expected no retry link for completed quest, got: %s", body)
	}
}

func TestHandleViewQuest_failed(t *testing.T) {
	stub := &stubQuerier{
		quest: sqlc.QuestsQuest{
			ID:        2,
			Title:     "Failed Quest",
			QuestType: "side",
			Status:    "failed",
			FailedAt:  pgtype.Timestamptz{Time: time.Date(2026, 3, 14, 9, 0, 0, 0, time.UTC), Valid: true},
		},
	}
	tmpl := mustParseTestTemplates(t)
	h := handleViewQuest(stub, tmpl)

	req := httptest.NewRequest("GET", "/quests/2", nil)
	req.SetPathValue("id", "2")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "quest-detail") {
		t.Errorf("expected 'quest-detail' in body, got: %s", body)
	}
	if !strings.Contains(body, "/quests/2/retry") {
		t.Errorf("expected retry link in body for failed quest, got: %s", body)
	}
}

func TestHandleViewQuest_noQuestLine(t *testing.T) {
	stub := &stubQuerier{
		quest: sqlc.QuestsQuest{
			ID:          3,
			Title:       "Solo Quest",
			QuestType:   "daily",
			Status:      "completed",
			CompletedAt: pgtype.Timestamptz{Time: time.Date(2026, 3, 15, 8, 0, 0, 0, time.UTC), Valid: true},
		},
	}
	tmpl := mustParseTestTemplates(t)
	h := handleViewQuest(stub, tmpl)

	req := httptest.NewRequest("GET", "/quests/3", nil)
	req.SetPathValue("id", "3")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	if strings.Contains(body, "line=") {
		t.Errorf("expected no quest line in body, got: %s", body)
	}
}

func TestHandleViewQuest_notFound(t *testing.T) {
	stub := &stubQuerier{
		err: errors.New("not found"),
	}
	tmpl := mustParseTestTemplates(t)
	h := handleViewQuest(stub, tmpl)

	req := httptest.NewRequest("GET", "/quests/99", nil)
	req.SetPathValue("id", "99")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestHandleViewQuest_invalidID(t *testing.T) {
	stub := &stubQuerier{}
	tmpl := mustParseTestTemplates(t)
	h := handleViewQuest(stub, tmpl)

	req := httptest.NewRequest("GET", "/quests/abc", nil)
	req.SetPathValue("id", "abc")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandleViewQuest_recurrence(t *testing.T) {
	stub := &stubQuerier{
		quest: sqlc.QuestsQuest{
			ID:             4,
			Title:          "Recurring Quest",
			QuestType:      "daily",
			Status:         "completed",
			CompletedAt:    pgtype.Timestamptz{Time: time.Date(2026, 3, 15, 8, 0, 0, 0, time.UTC), Valid: true},
			RecurrenceType: pgtype.Text{String: "every", Valid: true},
			RecurrenceN:    pgtype.Int4{Int32: 3, Valid: true},
			RecurrenceUnit: pgtype.Text{String: "days", Valid: true},
		},
	}
	tmpl := mustParseTestTemplates(t)
	h := handleViewQuest(stub, tmpl)

	req := httptest.NewRequest("GET", "/quests/4", nil)
	req.SetPathValue("id", "4")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "recurrence=every") {
		t.Errorf("expected 'recurrence=every' in body, got: %s", body)
	}
}

func TestHandleQuestLineDetail(t *testing.T) {
	stub := &stubQuerier{
		questLine: sqlc.GetQuestLineRow{
			ID:   1,
			Name: "Main Story",
		},
		questsByLine: []sqlc.QuestsQuest{
			{ID: 10, Title: "Completed Quest", Status: "completed"},
			{ID: 11, Title: "Failed Quest", Status: "failed"},
			{ID: 12, Title: "Active Dated", Status: "active"},
			{ID: 13, Title: "Active Undated", Status: "active"},
		},
	}
	tmpl := mustParseTestTemplates(t)
	h := handleQuestLineDetail(stub, tmpl)

	req := httptest.NewRequest("GET", "/quest-lines/1", nil)
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "name=Main Story") {
		t.Errorf("expected quest line name in body, got: %s", body)
	}
	if !strings.Contains(body, "count=4") {
		t.Errorf("expected count=4 in body, got: %s", body)
	}
	if !strings.Contains(body, "[Completed Quest:completed]") {
		t.Errorf("expected completed quest in body, got: %s", body)
	}
	if !strings.Contains(body, "[Failed Quest:failed]") {
		t.Errorf("expected failed quest in body, got: %s", body)
	}
	if !strings.Contains(body, "[Active Dated:active]") {
		t.Errorf("expected active dated quest in body, got: %s", body)
	}
}

func TestHandleQuestLineDetail_invalidID(t *testing.T) {
	stub := &stubQuerier{}
	tmpl := mustParseTestTemplates(t)
	h := handleQuestLineDetail(stub, tmpl)

	req := httptest.NewRequest("GET", "/quest-lines/abc", nil)
	req.SetPathValue("id", "abc")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandleQuestLineDetail_notFound(t *testing.T) {
	stub := &stubQuerier{err: errors.New("not found")}
	tmpl := mustParseTestTemplates(t)
	h := handleQuestLineDetail(stub, tmpl)

	req := httptest.NewRequest("GET", "/quest-lines/999", nil)
	req.SetPathValue("id", "999")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestHandleQuestLineDetail_empty(t *testing.T) {
	stub := &stubQuerier{
		questLine: sqlc.GetQuestLineRow{ID: 2, Name: "Empty Line"},
	}
	tmpl := mustParseTestTemplates(t)
	h := handleQuestLineDetail(stub, tmpl)

	req := httptest.NewRequest("GET", "/quest-lines/2", nil)
	req.SetPathValue("id", "2")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "count=0") {
		t.Errorf("expected count=0 in body, got: %s", w.Body.String())
	}
}
