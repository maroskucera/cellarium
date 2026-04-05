// Cellarium Quests — tests for background ticker / push notification logic
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
	"testing"
	"time"

	"github.com/maroskucera/cellarium/quests/db/sqlc"
)

func TestRunTickerTasks_callsEnsureFailedQuests(t *testing.T) {
	stub := &stubQuerier{}
	cfg := notifyConfig{}
	now := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	runTickerTasks(nil, stub, &stubTxRunner{q: stub}, cfg, now) //nolint:staticcheck
	if stub.failOverdueQuestsCall == nil {
		t.Fatal("expected FailOverdueQuests to be called")
	}
}

func TestRunTickerTasks_noVAPIDKey_skipsPush(t *testing.T) {
	stub := &stubQuerier{
		dueReminders: []sqlc.QuestsQuest{
			{ID: 1, Title: "Test quest"},
		},
	}
	cfg := notifyConfig{} // no VAPID key
	now := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	runTickerTasks(nil, stub, &stubTxRunner{q: stub}, cfg, now) //nolint:staticcheck
	if stub.listDueRemindersCalled {
		t.Error("expected ListDueReminders not to be called when no VAPID key")
	}
	if stub.markReminderSentCalled {
		t.Error("expected MarkReminderSent not to be called when no VAPID key")
	}
}

func TestRunTickerTasks_marksReminderSent(t *testing.T) {
	stub := &stubQuerier{
		dueReminders: []sqlc.QuestsQuest{
			{ID: 42, Title: "Urgent quest"},
		},
	}
	cfg := notifyConfig{
		VAPIDPrivateKey: "fake-private-key",
		VAPIDPublicKey:  "fake-public-key",
		VAPIDSubject:    "mailto:test@example.com",
	}
	now := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	runTickerTasks(nil, stub, &stubTxRunner{q: stub}, cfg, now) //nolint:staticcheck
	if !stub.markReminderSentCalled {
		t.Fatal("expected MarkReminderSent to be called")
	}
}
