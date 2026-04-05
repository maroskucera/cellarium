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
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	webpush "github.com/SherClockHolmes/webpush-go"
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
	appLocation = time.UTC
	orig := webpushSend
	t.Cleanup(func() { webpushSend = orig })
	webpushSend = func(message []byte, s *webpush.Subscription, options *webpush.Options) (*http.Response, error) {
		return &http.Response{
			StatusCode: 201,
			Body:       io.NopCloser(strings.NewReader("")),
		}, nil
	}

	stub := &stubQuerier{
		dueReminders: []sqlc.QuestsQuest{
			{ID: 42, Title: "Urgent quest"},
		},
		pushSubs: []sqlc.QuestsPushSubscription{
			{Endpoint: "https://example.com/push/1", P256dh: "p", Auth: "a"},
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

func TestRunTickerTasks_noSubs_doesNotMarkSent(t *testing.T) {
	appLocation = time.UTC
	orig := webpushSend
	t.Cleanup(func() { webpushSend = orig })
	webpushSend = func(message []byte, s *webpush.Subscription, options *webpush.Options) (*http.Response, error) {
		t.Fatal("sendPush should not be called when there are no subscriptions")
		return nil, nil
	}

	stub := &stubQuerier{
		dueReminders: []sqlc.QuestsQuest{
			{ID: 42, Title: "Urgent quest"},
		},
		// pushSubs intentionally empty — no subscriptions
	}
	cfg := notifyConfig{
		VAPIDPrivateKey: "fake-private-key",
		VAPIDPublicKey:  "fake-public-key",
		VAPIDSubject:    "mailto:test@example.com",
	}
	now := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	runTickerTasks(nil, stub, &stubTxRunner{q: stub}, cfg, now) //nolint:staticcheck
	if stub.markReminderSentCalled {
		t.Error("expected MarkReminderSent NOT to be called when there are no push subscriptions")
	}
}

func TestRunTickerTasks_allPushFail_doesNotMarkSent(t *testing.T) {
	appLocation = time.UTC
	orig := webpushSend
	t.Cleanup(func() { webpushSend = orig })
	webpushSend = func(message []byte, s *webpush.Subscription, options *webpush.Options) (*http.Response, error) {
		return nil, errors.New("connection refused")
	}

	stub := &stubQuerier{
		dueReminders: []sqlc.QuestsQuest{
			{ID: 42, Title: "Urgent quest"},
		},
		pushSubs: []sqlc.QuestsPushSubscription{
			{Endpoint: "https://example.com/push/1", P256dh: "p", Auth: "a"},
		},
	}
	cfg := notifyConfig{
		VAPIDPrivateKey: "fake-private-key",
		VAPIDPublicKey:  "fake-public-key",
		VAPIDSubject:    "mailto:test@example.com",
	}
	now := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	runTickerTasks(nil, stub, &stubTxRunner{q: stub}, cfg, now) //nolint:staticcheck
	if stub.markReminderSentCalled {
		t.Error("expected MarkReminderSent NOT to be called when all pushes fail with transport error")
	}
}

func TestRunTickerTasks_allGone_doesNotMarkSent(t *testing.T) {
	appLocation = time.UTC
	orig := webpushSend
	t.Cleanup(func() { webpushSend = orig })
	webpushSend = func(message []byte, s *webpush.Subscription, options *webpush.Options) (*http.Response, error) {
		return &http.Response{
			StatusCode: 410,
			Body:       io.NopCloser(strings.NewReader("")),
		}, nil
	}

	stub := &stubQuerier{
		dueReminders: []sqlc.QuestsQuest{
			{ID: 42, Title: "Urgent quest"},
		},
		pushSubs: []sqlc.QuestsPushSubscription{
			{Endpoint: "https://example.com/push/1", P256dh: "p", Auth: "a"},
		},
	}
	cfg := notifyConfig{
		VAPIDPrivateKey: "fake-private-key",
		VAPIDPublicKey:  "fake-public-key",
		VAPIDSubject:    "mailto:test@example.com",
	}
	now := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	runTickerTasks(nil, stub, &stubTxRunner{q: stub}, cfg, now) //nolint:staticcheck
	if stub.markReminderSentCalled {
		t.Error("expected MarkReminderSent NOT to be called when all subscriptions are gone (410)")
	}
}

func TestRunTickerTasks_partialSuccess_marksSent(t *testing.T) {
	appLocation = time.UTC
	orig := webpushSend
	t.Cleanup(func() { webpushSend = orig })
	callCount := 0
	webpushSend = func(message []byte, s *webpush.Subscription, options *webpush.Options) (*http.Response, error) {
		callCount++
		if callCount == 1 {
			return nil, errors.New("connection refused")
		}
		return &http.Response{
			StatusCode: 201,
			Body:       io.NopCloser(strings.NewReader("")),
		}, nil
	}

	stub := &stubQuerier{
		dueReminders: []sqlc.QuestsQuest{
			{ID: 42, Title: "Urgent quest"},
		},
		pushSubs: []sqlc.QuestsPushSubscription{
			{Endpoint: "https://example.com/push/1", P256dh: "p", Auth: "a"},
			{Endpoint: "https://example.com/push/2", P256dh: "p", Auth: "a"},
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
		t.Error("expected MarkReminderSent to be called when at least one push succeeds")
	}
}

func TestSendPush_Success(t *testing.T) {
	orig := webpushSend
	t.Cleanup(func() { webpushSend = orig })

	webpushSend = func(message []byte, s *webpush.Subscription, options *webpush.Options) (*http.Response, error) {
		return &http.Response{
			StatusCode: 201,
			Body:       io.NopCloser(strings.NewReader("")),
		}, nil
	}

	sub := sqlc.QuestsPushSubscription{
		Endpoint: "https://example.com/push/abc",
		P256dh:   "test-p256dh",
		Auth:     "test-auth",
	}
	cfg := notifyConfig{
		VAPIDPrivateKey: "fake-private-key",
		VAPIDPublicKey:  "fake-public-key",
		VAPIDSubject:    "mailto:test@example.com",
	}

	result := sendPush(sub, "test payload", cfg)

	if result.StatusCode != 201 {
		t.Errorf("expected StatusCode 201, got %d", result.StatusCode)
	}
	if result.Error != "" {
		t.Errorf("expected no error, got %q", result.Error)
	}
	if result.Endpoint != "https://example.com/push/abc" {
		t.Errorf("expected endpoint %q, got %q", "https://example.com/push/abc", result.Endpoint)
	}
}

func TestSendPush_GoneSubscription(t *testing.T) {
	orig := webpushSend
	t.Cleanup(func() { webpushSend = orig })

	webpushSend = func(message []byte, s *webpush.Subscription, options *webpush.Options) (*http.Response, error) {
		return &http.Response{
			StatusCode: 410,
			Body:       io.NopCloser(strings.NewReader("")),
		}, nil
	}

	sub := sqlc.QuestsPushSubscription{
		Endpoint: "https://example.com/push/gone",
		P256dh:   "test-p256dh",
		Auth:     "test-auth",
	}
	cfg := notifyConfig{
		VAPIDPrivateKey: "fake-private-key",
		VAPIDPublicKey:  "fake-public-key",
		VAPIDSubject:    "mailto:test@example.com",
	}

	result := sendPush(sub, "test payload", cfg)

	if result.StatusCode != 410 {
		t.Errorf("expected StatusCode 410, got %d", result.StatusCode)
	}
}

func TestSendPush_TransportError(t *testing.T) {
	orig := webpushSend
	t.Cleanup(func() { webpushSend = orig })

	webpushSend = func(message []byte, s *webpush.Subscription, options *webpush.Options) (*http.Response, error) {
		return nil, errors.New("connection refused")
	}

	sub := sqlc.QuestsPushSubscription{
		Endpoint: "https://example.com/push/fail",
		P256dh:   "test-p256dh",
		Auth:     "test-auth",
	}
	cfg := notifyConfig{
		VAPIDPrivateKey: "fake-private-key",
		VAPIDPublicKey:  "fake-public-key",
		VAPIDSubject:    "mailto:test@example.com",
	}

	result := sendPush(sub, "test payload", cfg)

	if result.Error != "connection refused" {
		t.Errorf("expected error %q, got %q", "connection refused", result.Error)
	}
	if result.StatusCode != 0 {
		t.Errorf("expected StatusCode 0, got %d", result.StatusCode)
	}
}

func TestSendPush_ClosesResponseBody(t *testing.T) {
	orig := webpushSend
	t.Cleanup(func() { webpushSend = orig })

	closed := false
	body := &trackingReadCloser{
		Reader:  strings.NewReader(""),
		onClose: func() { closed = true },
	}

	webpushSend = func(message []byte, s *webpush.Subscription, options *webpush.Options) (*http.Response, error) {
		return &http.Response{
			StatusCode: 201,
			Body:       body,
		}, nil
	}

	sub := sqlc.QuestsPushSubscription{
		Endpoint: "https://example.com/push/close-test",
		P256dh:   "test-p256dh",
		Auth:     "test-auth",
	}
	cfg := notifyConfig{
		VAPIDPrivateKey: "fake-private-key",
		VAPIDPublicKey:  "fake-public-key",
		VAPIDSubject:    "mailto:test@example.com",
	}

	sendPush(sub, "test payload", cfg)

	if !closed {
		t.Error("expected response body to be closed")
	}
}

func TestSendPush_DefaultTTL(t *testing.T) {
	orig := webpushSend
	t.Cleanup(func() { webpushSend = orig })

	var capturedTTL int
	webpushSend = func(message []byte, s *webpush.Subscription, options *webpush.Options) (*http.Response, error) {
		capturedTTL = options.TTL
		return &http.Response{
			StatusCode: 201,
			Body:       io.NopCloser(strings.NewReader("")),
		}, nil
	}

	sub := sqlc.QuestsPushSubscription{
		Endpoint: "https://example.com/push/ttl-test",
		P256dh:   "test-p256dh",
		Auth:     "test-auth",
	}
	cfg := notifyConfig{
		VAPIDPrivateKey: "fake-private-key",
		VAPIDPublicKey:  "fake-public-key",
		VAPIDSubject:    "mailto:test@example.com",
	}

	sendPush(sub, "test payload", cfg)

	if capturedTTL != defaultPushTTL {
		t.Errorf("expected default TTL %d, got %d", defaultPushTTL, capturedTTL)
	}
}

// trackingReadCloser wraps an io.Reader and tracks whether Close was called.
type trackingReadCloser struct {
	io.Reader
	onClose func()
}

func (t *trackingReadCloser) Close() error {
	t.onClose()
	return nil
}
