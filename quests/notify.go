// Cellarium Quests — background ticker for reminders and failure detection
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
	"encoding/json"
	"log"
	"time"

	"net/http"

	webpush "github.com/SherClockHolmes/webpush-go"
	"github.com/maroskucera/cellarium/quests/db/sqlc"
)

type pushResult struct {
	Endpoint   string `json:"endpoint"`
	StatusCode int    `json:"status_code"`
	Error      string `json:"error,omitempty"`
}

// webpushSend is the function used to send push notifications. Override in tests.
var webpushSend = webpush.SendNotification

// defaultPushTTL is the default time-to-live (in seconds) for web push messages.
// The push service holds the message for this long if the device is unreachable.
const defaultPushTTL = 86400

type notifyConfig struct {
	VAPIDPrivateKey string
	VAPIDPublicKey  string
	VAPIDSubject    string
	TTL             int // push message TTL in seconds; 0 = defaultPushTTL
}

// startTicker runs the background goroutine that checks reminders and failures.
// Call with go startTicker(ctx, q, cfg).
func startTicker(ctx context.Context, q sqlc.Querier, tx txRunner, cfg notifyConfig) {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case t := <-ticker.C:
			runTickerTasks(ctx, q, tx, cfg, t)
		}
	}
}

func runTickerTasks(ctx context.Context, q sqlc.Querier, tx txRunner, cfg notifyConfig, now time.Time) {
	today := localToday(now)
	if err := ensureFailedQuests(ctx, tx, today); err != nil {
		log.Printf("ensureFailedQuests error: %v", err)
	}
	if cfg.VAPIDPrivateKey == "" {
		return
	}
	nowTime := localTime(now)
	dueQuests, err := q.ListDueReminders(ctx, sqlc.ListDueRemindersParams{
		NowTime: nowTime,
		Today:   today,
	})
	if err != nil {
		log.Printf("ListDueReminders error: %v", err)
		return
	}
	if len(dueQuests) == 0 {
		return
	}
	subs, err := q.ListPushSubscriptions(ctx)
	if err != nil {
		log.Printf("ListPushSubscriptions error: %v", err)
		return
	}
	gone := map[string]bool{}
	for _, quest := range dueQuests {
		payload, _ := json.Marshal(map[string]string{
			"title": quest.Title,
			"body":  "Quest reminder",
		})
		delivered := false
		for _, sub := range subs {
			if gone[sub.Endpoint] {
				continue
			}
			result := sendPush(sub, string(payload), cfg)
			if result.StatusCode >= http.StatusOK && result.StatusCode < http.StatusMultipleChoices {
				delivered = true
			}
			if result.StatusCode == http.StatusGone {
				if err := q.DeletePushSubscription(ctx, sub.Endpoint); err != nil {
					log.Printf("DeletePushSubscription error for %s: %v", sub.Endpoint, err)
				}
				gone[sub.Endpoint] = true
			}
		}
		if delivered {
			if err := q.MarkReminderSent(ctx, sqlc.MarkReminderSentParams{
				Today: today,
				ID:    quest.ID,
			}); err != nil {
				log.Printf("MarkReminderSent error: %v", err)
			}
		}
	}
}

func sendPush(sub sqlc.QuestsPushSubscription, payload string, cfg notifyConfig) pushResult {
	ttl := cfg.TTL
	if ttl == 0 {
		ttl = defaultPushTTL
	}
	resp, err := webpushSend([]byte(payload), &webpush.Subscription{
		Endpoint: sub.Endpoint,
		Keys: webpush.Keys{
			Auth:   sub.Auth,
			P256dh: sub.P256dh,
		},
	}, &webpush.Options{
		VAPIDPrivateKey: cfg.VAPIDPrivateKey,
		VAPIDPublicKey:  cfg.VAPIDPublicKey,
		Subscriber:      cfg.VAPIDSubject,
		TTL:             ttl,
	})
	if err != nil {
		log.Printf("sendPush error for %s: %v", sub.Endpoint, err)
		return pushResult{Endpoint: sub.Endpoint, Error: err.Error()}
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		log.Printf("sendPush: %s returned HTTP %d", sub.Endpoint, resp.StatusCode)
	}
	return pushResult{Endpoint: sub.Endpoint, StatusCode: resp.StatusCode}
}
