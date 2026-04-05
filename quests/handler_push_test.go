// Cellarium Quests — push notification handler tests
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
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	webpush "github.com/SherClockHolmes/webpush-go"
	"github.com/maroskucera/cellarium/quests/db/sqlc"
)

func TestHandlePushSubscribe_Success(t *testing.T) {
	stub := &stubQuerier{}
	h := handlePushSubscribe(stub)

	body := `{"endpoint":"https://example.com/push","keys":{"p256dh":"key1","auth":"auth1"}}`
	req := httptest.NewRequest("POST", "/api/push/subscribe", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
	if stub.lastCreatePushSubscriptionParams.Endpoint != "https://example.com/push" {
		t.Errorf("expected endpoint to be stored, got %q", stub.lastCreatePushSubscriptionParams.Endpoint)
	}
	if stub.lastCreatePushSubscriptionParams.P256dh != "key1" {
		t.Errorf("expected p256dh 'key1', got %q", stub.lastCreatePushSubscriptionParams.P256dh)
	}
	if stub.lastCreatePushSubscriptionParams.Auth != "auth1" {
		t.Errorf("expected auth 'auth1', got %q", stub.lastCreatePushSubscriptionParams.Auth)
	}
}

func TestHandlePushSubscribe_InvalidJSON(t *testing.T) {
	stub := &stubQuerier{}
	h := handlePushSubscribe(stub)

	req := httptest.NewRequest("POST", "/api/push/subscribe", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandlePushSubscribe_MissingFields(t *testing.T) {
	stub := &stubQuerier{}
	h := handlePushSubscribe(stub)

	body := `{"endpoint":"","keys":{"p256dh":"","auth":""}}`
	req := httptest.NewRequest("POST", "/api/push/subscribe", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandlePushSubscribe_DBError(t *testing.T) {
	stub := &stubQuerier{err: errors.New("db error")}
	h := handlePushSubscribe(stub)

	body := `{"endpoint":"https://example.com/push","keys":{"p256dh":"key1","auth":"auth1"}}`
	req := httptest.NewRequest("POST", "/api/push/subscribe", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestHandlePushUnsubscribe_Success(t *testing.T) {
	stub := &stubQuerier{}
	h := handlePushUnsubscribe(stub)

	body := `{"endpoint":"https://example.com/push"}`
	req := httptest.NewRequest("POST", "/api/push/unsubscribe", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
}

func TestHandlePushUnsubscribe_MissingEndpoint(t *testing.T) {
	stub := &stubQuerier{}
	h := handlePushUnsubscribe(stub)

	body := `{"endpoint":""}`
	req := httptest.NewRequest("POST", "/api/push/unsubscribe", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandlePushVapidKey(t *testing.T) {
	h := handlePushVapidKey("test-vapid-key")

	req := httptest.NewRequest("GET", "/api/push/vapid-public-key", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if w.Header().Get("Content-Type") != "text/plain" {
		t.Errorf("expected text/plain content type, got %q", w.Header().Get("Content-Type"))
	}
	if w.Body.String() != "test-vapid-key" {
		t.Errorf("expected 'test-vapid-key', got %q", w.Body.String())
	}
}

func TestHandlePushTest_NoVAPID(t *testing.T) {
	stub := &stubQuerier{}
	h := handlePushTest(stub, notifyConfig{})

	req := httptest.NewRequest("POST", "/api/push/test", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
}

func TestHandlePushTest_NoSubscriptions(t *testing.T) {
	stub := &stubQuerier{
		pushSubs: []sqlc.QuestsPushSubscription{},
	}
	cfg := notifyConfig{
		VAPIDPrivateKey: "fake-private-key",
		VAPIDPublicKey:  "fake-public-key",
		VAPIDSubject:    "mailto:test@example.com",
	}
	h := handlePushTest(stub, cfg)

	req := httptest.NewRequest("POST", "/api/push/test", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestHandlePushTest_DBError(t *testing.T) {
	stub := &stubQuerier{
		err: errors.New("db error"),
	}
	cfg := notifyConfig{
		VAPIDPrivateKey: "fake-private-key",
		VAPIDPublicKey:  "fake-public-key",
		VAPIDSubject:    "mailto:test@example.com",
	}
	h := handlePushTest(stub, cfg)

	req := httptest.NewRequest("POST", "/api/push/test", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestHandlePushTest_Success(t *testing.T) {
	orig := webpushSend
	t.Cleanup(func() { webpushSend = orig })

	webpushSend = func(message []byte, s *webpush.Subscription, options *webpush.Options) (*http.Response, error) {
		return &http.Response{
			StatusCode: 201,
			Body:       io.NopCloser(strings.NewReader("")),
		}, nil
	}

	stub := &stubQuerier{
		pushSubs: []sqlc.QuestsPushSubscription{
			{
				ID:       1,
				Endpoint: "https://example.com/push/test-endpoint",
				P256dh:   "test-p256dh",
				Auth:     "test-auth",
			},
		},
	}
	cfg := notifyConfig{
		VAPIDPrivateKey: "fake-private-key",
		VAPIDPublicKey:  "fake-public-key",
		VAPIDSubject:    "mailto:test@example.com",
	}
	h := handlePushTest(stub, cfg)

	req := httptest.NewRequest("POST", "/api/push/test", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var results []pushResult
	if err := json.NewDecoder(w.Body).Decode(&results); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Endpoint != "https://example.com/push/test-endpoint" {
		t.Errorf("expected endpoint %q, got %q", "https://example.com/push/test-endpoint", results[0].Endpoint)
	}
	if results[0].StatusCode != 201 {
		t.Errorf("expected StatusCode 201, got %d", results[0].StatusCode)
	}
}
