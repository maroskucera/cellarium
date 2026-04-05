// Cellarium Quests — push notification subscription handlers
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
	"log"
	"net/http"

	"github.com/maroskucera/cellarium/quests/db/sqlc"
)

type pushSubscribeRequest struct {
	Endpoint string `json:"endpoint"`
	Keys     struct {
		P256dh string `json:"p256dh"`
		Auth   string `json:"auth"`
	} `json:"keys"`
}

type pushUnsubscribeRequest struct {
	Endpoint string `json:"endpoint"`
}

func handlePushSubscribe(q sqlc.Querier) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req pushSubscribeRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if req.Endpoint == "" || req.Keys.P256dh == "" || req.Keys.Auth == "" {
			http.Error(w, "missing fields", http.StatusBadRequest)
			return
		}
		if _, err := q.CreatePushSubscription(r.Context(), sqlc.CreatePushSubscriptionParams{
			Endpoint: req.Endpoint,
			P256dh:   req.Keys.P256dh,
			Auth:     req.Keys.Auth,
		}); err != nil {
			log.Printf("handlePushSubscribe: failed to store subscription for %s: %v", req.Endpoint, err)
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}

func handlePushUnsubscribe(q sqlc.Querier) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req pushUnsubscribeRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if req.Endpoint == "" {
			http.Error(w, "missing endpoint", http.StatusBadRequest)
			return
		}
		if err := q.DeletePushSubscription(r.Context(), req.Endpoint); err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}

func handlePushTest(q sqlc.Querier, cfg notifyConfig) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if cfg.VAPIDPrivateKey == "" {
			http.Error(w, "VAPID keys not configured", http.StatusServiceUnavailable)
			return
		}
		subs, err := q.ListPushSubscriptions(r.Context())
		if err != nil {
			log.Printf("handlePushTest: ListPushSubscriptions error: %v", err)
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}
		if len(subs) == 0 {
			http.Error(w, "no push subscriptions found", http.StatusNotFound)
			return
		}
		payload, _ := json.Marshal(map[string]string{
			"title": "Test Notification",
			"body":  "Push notifications are working!",
		})
		var results []pushResult
		for _, sub := range subs {
			result := sendPush(sub, string(payload), cfg)
			results = append(results, result)
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(results); err != nil {
			log.Printf("handlePushTest: failed to encode response: %v", err)
		}
	})
}

func handlePushVapidKey(vapidPublicKey string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(vapidPublicKey))
	})
}
