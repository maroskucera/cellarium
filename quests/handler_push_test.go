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
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
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
